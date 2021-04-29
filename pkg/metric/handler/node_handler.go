package handler

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	alarm "github.com/linkingthing/clxone-dhcp/pkg/alarm/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	logresource "github.com/linkingthing/clxone-dhcp/pkg/log/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util/httpclient"
)

var (
	TableNode                = restdb.ResourceDBType(&resource.Node{})
	instance                 = "instance"
	device                   = "device"
	docker                   = "docker"
	dockerInterface          = "veth"
	schema                   = "http://"
	defaultNodeTimeout int64 = 30
)

type NodeHandler struct {
	prometheusAddr string
	exportPort     int
	LocalIP        string
}

func NewNodeHandler(conf *config.DDIControllerConfig) *NodeHandler {
	h := &NodeHandler{
		prometheusAddr: conf.Prometheus.Addr,
		exportPort:     conf.Prometheus.ExportPort,
		LocalIP:        conf.Server.IP,
	}

	timeout := conf.MonitorNode.TimeOut
	if timeout == 0 {
		timeout = defaultNodeTimeout
	}

	go h.monitor()
	go h.nodeAliveCheck(timeout)
	return h
}

func (h *NodeHandler) nodeAliveCheck(timeout int64) {
	nodeTimeout := time.Second * time.Duration(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			var nodes []*resource.Node
			var thresholds []*alarm.Threshold
			if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
				if err := tx.Fill(nil, &nodes); err != nil {
					return err
				}

				if err := tx.Fill(map[string]interface{}{"name": alarm.ThresholdNameNodeOffline},
					&thresholds); err != nil {
					return err
				}

				now := time.Now()
				for _, node := range nodes {
					if now.Sub(node.UpdateTime) > nodeTimeout {
						node.NodeIsAlive = false
						if _, err := tx.Update(TableNode,
							map[string]interface{}{
								"node_is_alive": false,
								"dhcp_is_alive": false,
								"dns_is_alive":  false},
							map[string]interface{}{restdb.IDField: node.GetID()}); err != nil {
							return err
						}
					}
				}

				return nil
			}); err != nil {
				log.Warnf("node alive check failed: %s", err.Error())
			}

			sendNodeOfflineEventIfNeed(nodes, thresholds)
		}
	}
}

func sendNodeOfflineEventIfNeed(nodes []*resource.Node, thresholds []*alarm.Threshold) {
	if len(thresholds) == 0 {
		return
	}

	for _, node := range nodes {
		if node.NodeIsAlive == false {
			alarm.NewEvent().Node(node.Ip).Name(thresholds[0].Name).Level(thresholds[0].Level).
				ThresholdType(thresholds[0].ThresholdType).Time(time.Now()).
				SendMail(thresholds[0].SendMail).Publish()
		}
	}
}

func (h *NodeHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var nodes []*resource.Node
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&nodes, "select * from gr_node where roles != $1",
			[]string{string(resource.ServiceRoleDataCenter)})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list nodes from db failed: %s", err.Error()))
	}

	for i, node := range nodes {
		if index := slice.SliceIndex(node.Roles, string(resource.ServiceRoleDataCenter)); index != -1 {
			nodes[i].Roles = append(nodes[i].Roles[:index], nodes[i].Roles[index+1:]...)
		}
	}

	period, err := getTimePeriodParamFromFilter(ctx.GetFilters())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("invalid time format: %s", err.Error()))
	}

	if err := h.setNodeMetrics(getNodeAddrsLabel(nodes, h.exportPort), nodes, period); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list nodes metrics from prometheus failed: %s", err.Error()))
	}

	return nodes, nil
}

func (h *NodeHandler) setNodeMetrics(nodesAddrLabel string, nodes []*resource.Node, period *TimePeriodParams) error {
	if err := h.getMemoryRatio(nodesAddrLabel, nodes, period); err != nil {
		return fmt.Errorf("get nodes memory metrics from prometheus failed: %s", err.Error())
	}
	if err := h.getCPURatio(nodesAddrLabel, nodes, period); err != nil {
		return fmt.Errorf("get nodes cpu ratio metrics from prometheus failed: %s", err.Error())
	}

	if err := h.getDiscRatio(nodesAddrLabel, nodes, period); err != nil {
		return fmt.Errorf("get nodes disk ratio metric from prometheus failed: %s", err.Error())
	}

	if err := h.getNetworkRatio(nodesAddrLabel, nodes, period); err != nil {
		return fmt.Errorf("get nodes network metric from prometheus failed: %s", err.Error())
	}
	for i, node := range nodes {
		if !node.NodeIsAlive {
			continue
		}

		if len(node.CpuUsage) >= 1 {
			nodes[i].CpuRatio = node.CpuUsage[len(node.CpuUsage)-1].Ratio
		}
		if len(node.MemoryUsage) >= 1 {
			nodes[i].MemRatio = node.MemoryUsage[len(node.MemoryUsage)-1].Ratio
		}
	}
	return nil
}

func getNodeAddrsLabel(nodes []*resource.Node, port int) string {
	var nodesAddrs []string
	for _, node := range nodes {
		nodesAddrs = append(nodesAddrs, node.GetID()+":"+strconv.Itoa(port))
	}
	return strings.Join(nodesAddrs, "|")
}

func (h *NodeHandler) getMemoryRatio(nodesAddrLabel string, nodes []*resource.Node, period *TimePeriodParams) error {
	pql := "1 - (node_memory_MemFree_bytes{instance=~\"" + nodesAddrLabel +
		"\"}+node_memory_Cached_bytes{instance=~\"" + nodesAddrLabel +
		"\"}+node_memory_Buffers_bytes{instance=~\"" + nodesAddrLabel +
		"\"}) / node_memory_MemTotal_bytes"
	resp, err := h.getPrometheusData(pql, period)
	if err != nil {
		return err
	}
	if err := h.resolveMemoryValues(nodes, resp, period); err != nil {
		return err
	}
	return nil
}

func (h *NodeHandler) getCPURatio(nodesAddrLabel string, nodes []*resource.Node, period *TimePeriodParams) error {
	pql := "1 - (avg(irate(node_cpu_seconds_total{instance=~\"" + nodesAddrLabel +
		"\", mode=\"idle\"}[5m])) by (instance))"
	resp, err := h.getPrometheusData(pql, period)
	if err != nil {
		return err
	}
	if err := h.resolveCPUValues(nodes, resp, period); err != nil {
		return err
	}
	return nil
}

func (h *NodeHandler) getDiscRatio(nodesAddrLabel string, nodes []*resource.Node, period *TimePeriodParams) error {
	pqlFree := "node_filesystem_free_bytes{instance=~\"" + nodesAddrLabel +
		"\", fstype=~\"ext4|xfs\"}"
	respFree, err := h.getPrometheusData(pqlFree, period)
	if err != nil {
		return err
	}

	pqlTotal := "node_filesystem_size_bytes{instance=~\"" + nodesAddrLabel +
		"\", fstype=~\"ext4|xfs\"}"
	respTotal, err := h.getPrometheusData(pqlTotal, period)
	if err != nil {
		return err
	}
	if err := h.resolveDiscValues(nodes, respFree, respTotal, period); err != nil {
		return err
	}
	return nil
}

func (h *NodeHandler) getNetworkRatio(nodesAddrLabel string, nodes []*resource.Node, period *TimePeriodParams) error {
	pql := "irate(node_network_receive_bytes_total{device!=\"lo\", instance=~\"" + nodesAddrLabel +
		"\"}[5m]) + irate(node_network_transmit_bytes_total{device!=\"lo\", instance=~\"" +
		nodesAddrLabel + "\"}[5m])"
	resp, err := h.getPrometheusData(pql, period)
	if err != nil {
		return err
	}
	if err := h.resolveNetworkValues(nodes, resp, period); err != nil {
		return err
	}
	return nil
}

func (h *NodeHandler) getPrometheusData(pql string, period *TimePeriodParams) (*PrometheusResponse, error) {
	param := url.Values{}
	param.Add("query", pql)
	path := schema + h.prometheusAddr + "/api/v1/query_range?" + param.Encode() +
		fmt.Sprintf("&start=%d&end=%d&step=%d", period.Begin, period.End, period.Step)
	var resp PrometheusResponse
	if err := httpclient.GetHttpClient().Get(path, &resp); err != nil {
		return nil, err
	}

	if resp.Status != "success" {
		return nil, fmt.Errorf("get metric failed with status: %s", resp.Status)
	}

	return &resp, nil
}

func (h *NodeHandler) resolveCPUValues(nodes []*resource.Node, resp *PrometheusResponse, period *TimePeriodParams) error {
	for _, node := range nodes {
		for _, r := range resp.Data.Results {
			if node.Ip+":"+strconv.Itoa(h.exportPort) != r.MetricLabels[instance] {
				continue
			}
			node.CpuUsage = getRatiosWithTimestamp(r.Values, period)
		}
	}
	return nil
}

func (h *NodeHandler) resolveDiscValues(nodes []*resource.Node, respFree *PrometheusResponse, respTotal *PrometheusResponse, period *TimePeriodParams) error {
	for _, node := range nodes {
		var dataFree [][]resource.RatioWithTimestamp
		var dataUsed [][]resource.RatioWithTimestamp
		for _, r := range respFree.Data.Results {
			if node.Ip+":"+strconv.Itoa(h.exportPort) != r.MetricLabels[instance] {
				continue
			}
			dataFree = append(dataFree, getRatiosWithTimestamp(r.Values, period))
		}
		for _, r := range respTotal.Data.Results {
			if node.Ip+":"+strconv.Itoa(h.exportPort) != r.MetricLabels[instance] {
				continue
			}
			dataUsed = append(dataUsed, getRatiosWithTimestamp(r.Values, period))
		}
		if len(dataUsed) != len(dataFree) {
			return fmt.Errorf("no data got or data not correct")
		}
		if len(dataUsed) == 0 || len(dataFree) == 0 {
			continue
		}
		for j := 0; j < len(dataFree[0]); j++ {
			tmp := resource.RatioWithTimestamp{Timestamp: dataFree[0][j].Timestamp}
			var sumFree float64
			var sumUsed float64
			for i := 0; i < len(dataFree); i++ {
				newValue, err := strconv.ParseFloat(dataFree[i][j].Ratio, 64)
				if err != nil {
					return err
				}
				sumFree += newValue
			}
			for i := 0; i < len(dataUsed); i++ {
				newValue, err := strconv.ParseFloat(dataUsed[i][j].Ratio, 64)
				if err != nil {
					return err
				}
				sumUsed += newValue
			}
			if sumUsed != 0 {
				tmp.Ratio = fmt.Sprintf("%.2f", (sumUsed-sumFree)/sumUsed)
			} else {
				tmp.Ratio = "0"
			}
			node.DiscUsage = append(node.DiscUsage, tmp)
		}
	}
	return nil
}

func (h *NodeHandler) resolveMemoryValues(nodes []*resource.Node, resp *PrometheusResponse, period *TimePeriodParams) error {
	for _, node := range nodes {
		for _, r := range resp.Data.Results {
			if node.Ip+":"+strconv.Itoa(h.exportPort) != r.MetricLabels[instance] {
				continue
			}
			node.MemoryUsage = getRatiosWithTimestamp(r.Values, period)
		}
	}
	return nil
}

func (h *NodeHandler) resolveNetworkValues(nodes []*resource.Node, resp *PrometheusResponse, period *TimePeriodParams) error {
	for _, node := range nodes {
		var data [][]resource.RatioWithTimestamp
		for _, r := range resp.Data.Results {
			if node.Ip+":"+strconv.Itoa(h.exportPort) != r.MetricLabels[instance] ||
				strings.Index(r.MetricLabels[device], docker) > 0 ||
				strings.Index(r.MetricLabels[device], dockerInterface) > 0 {
				continue
			}
			data = append(data, getRatiosWithTimestamp(r.Values, period))
		}
		if len(data) == 0 {
			continue
		}
		for j := 0; j < len(data[0]); j++ {
			var tmp resource.RatioWithTimestamp
			tmp.Timestamp = data[0][j].Timestamp
			var sum float64
			for i := 0; i < len(data); i++ {
				newValue, err := strconv.ParseFloat(data[i][j].Ratio, 64)
				if err != nil {
					return err
				}
				sum += newValue
			}
			tmp.Ratio = fmt.Sprintf("%.2f", sum/float64(1024))
			node.Network = append(node.Network, tmp)
		}
	}

	return nil
}

func (h *NodeHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ip := ctx.Resource.(*resource.Node).GetID()
	var nodes []*resource.Node
	_, err := restdb.GetResourceWithID(db.GetDB(), ip, &nodes)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s from db failed: %s", ip, err.Error()))
	}

	period, err := getTimePeriodParamFromFilter(ctx.GetFilters())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("invalid time format: %s", err.Error()))
	}

	if err := h.setNodeMetrics(getNodeAddrsLabel(nodes, h.exportPort), nodes, period); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s metrics from prometheus failed: %s", ip, err.Error()))
	}

	return nodes[0], nil
}

func (h *NodeHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ctx.Set(logresource.IgnoreAuditLog, nil)
	switch ctx.Resource.GetAction().Name {
	case resource.ActionRegister:
		return h.Register(ctx)
	case resource.ActionKeepalive:
		return h.Keepalive(ctx)
	case resource.ActionMasterUp:
		return h.MasterUp(ctx)
	case resource.ActionMasterDown:
		return h.MasterDown(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *NodeHandler) Register(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	node, ok := ctx.Resource.GetAction().Input.(*resource.Node)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("resource %s is unknown", ctx.Resource.GetAction().Name))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(TableNode, map[string]interface{}{restdb.IDField: node.ID}); err != nil {
			return err
		} else if exists {
			_, err := tx.Update(TableNode, map[string]interface{}{
				"roles":         node.Roles,
				"host_name":     node.HostName,
				"master":        node.Master,
				"controller_ip": node.ControllerIp,
				"start_time":    node.StartTime,
				"update_time":   time.Now()},
				map[string]interface{}{restdb.IDField: node.ID})
			return err
		} else {
			_, err := tx.Insert(node)
			return err
		}
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("register node %s failed: %s", node.HostName, err.Error()))
	}

	return nil, nil
}

func (h *NodeHandler) Keepalive(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	node, ok := ctx.Resource.GetAction().Input.(*resource.Node)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("resource %s is unknown", ctx.Resource.GetAction().Name))
	}

	var thresholds []*alarm.Threshold
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(TableNode, map[string]interface{}{restdb.IDField: node.ID}); err != nil {
			return err
		} else if !exists {
			_, err := tx.Insert(node)
			return err
		}

		if _, err := tx.Update(TableNode, map[string]interface{}{
			"node_is_alive": true,
			"dns_is_alive":  node.DnsIsAlive,
			"dhcp_is_alive": node.DhcpIsAlive,
			"vip":           node.Vip,
			"update_time":   time.Now(),
		}, map[string]interface{}{restdb.IDField: node.ID}); err != nil {
			return err
		}

		return tx.FillEx(&thresholds, "select * from gr_threshold where name in ($1, $2)",
			alarm.ThresholdNameDNSOffline, alarm.ThresholdNameDHCPOffline)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("register node %s failed: %s", node.HostName, err.Error()))
	}

	sendServiceOfflineEventIfNeed(node, thresholds)
	return nil, nil
}

func sendServiceOfflineEventIfNeed(node *resource.Node, thresholds []*alarm.Threshold) {
	if node.DnsIsAlive && node.DhcpIsAlive {
		return
	}

	for _, threshold := range thresholds {
		switch threshold.Name {
		case alarm.ThresholdNameDNSOffline:
			sendServiceEventIfNeed(node, node.DnsIsAlive, resource.ServiceRoleDNS, threshold)
		case alarm.ThresholdNameDHCPOffline:
			sendServiceEventIfNeed(node, node.DhcpIsAlive, resource.ServiceRoleDHCP, threshold)
		}
	}
}

func sendServiceEventIfNeed(node *resource.Node, online bool, role resource.ServiceRole, threshold *alarm.Threshold) {
	if online || slice.SliceIndex(node.Roles, string(role)) == -1 {
		return
	}

	alarm.NewEvent().Node(node.Ip).Name(threshold.Name).Level(threshold.Level).
		ThresholdType(threshold.ThresholdType).Time(time.Now()).SendMail(threshold.SendMail).Publish()
}

func (h *NodeHandler) MasterUp(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	haRequest, ok := ctx.Resource.GetAction().Input.(*resource.HaRequest)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("resource %s is unknown", ctx.Resource.GetAction().Name))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(TableNode, map[string]interface{}{
			"master":      "",
			"update_time": time.Now(),
			"vip":         haRequest.Vip,
		}, map[string]interface{}{restdb.IDField: haRequest.MasterIP}); err != nil {
			return err
		}

		_, err := tx.Update(TableNode, map[string]interface{}{
			"master":      haRequest.MasterIP,
			"update_time": time.Now(),
			"vip":         "",
		}, map[string]interface{}{restdb.IDField: haRequest.SlaveIP})
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update node %s failed: %s", haRequest.MasterIP, err.Error()))
	}

	sendHAEventIfNeed(resource.HaCmdMasterUp, haRequest)
	return nil, nil
}

func (h *NodeHandler) MasterDown(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	haRequest, ok := ctx.Resource.GetAction().Input.(*resource.HaRequest)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("resource %s is unknown", ctx.Resource.GetAction().Name))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(TableNode, map[string]interface{}{
			"master":      haRequest.SlaveIP,
			"update_time": time.Now(),
			"vip":         "",
		}, map[string]interface{}{restdb.IDField: haRequest.MasterIP}); err != nil {
			return err
		}

		_, err := tx.Update(TableNode, map[string]interface{}{
			"master":      "",
			"update_time": time.Now(),
			"vip":         haRequest.Vip,
		}, map[string]interface{}{restdb.IDField: haRequest.SlaveIP})
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update node %s failed: %s", haRequest.MasterIP, err.Error()))
	}

	sendHAEventIfNeed(resource.HaCmdMasterDown, haRequest)
	return nil, nil
}

func sendHAEventIfNeed(cmd resource.HaCmd, req *resource.HaRequest) {
	var thresholds []*alarm.Threshold
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{
			restdb.IDField: strings.ToLower(string(alarm.ThresholdNameHATrigger))}, &thresholds)
		return err
	}); err != nil {
		log.Warnf("get threshold failed: %s", err.Error())
		return
	}

	if len(thresholds) != 1 {
		return
	}

	alarm.NewEvent().Name(thresholds[0].Name).Level(thresholds[0].Level).
		ThresholdType(thresholds[0].ThresholdType).Time(time.Now()).SendMail(thresholds[0].SendMail).
		HaCmd(string(cmd)).HaRole(string(req.Role)).MasterIp(req.MasterIP).SlaveIp(req.SlaveIP).Publish()
}
