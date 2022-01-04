package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/linkingthing/cement/log"
	csvutil "github.com/linkingthing/clxone-utils/csv"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/api"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

const (
	PacketLabelPrefixVersion4 = "pkt4-"
	PacketLabelPrefixVersion6 = "pkt6-"
)

type PacketStatHandler struct {
	prometheusAddr string
}

func NewPacketStatHandler(conf *config.DHCPConfig) *PacketStatHandler {
	return &PacketStatHandler{prometheusAddr: conf.Prometheus.Addr}
}

func (h *PacketStatHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPPacketStats,
		PromQuery:      PromQueryVersion,
	}

	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get packet stats failed: %s", err.Error()))
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get packet stats from prometheus failed: %s", err.Error()))
	}

	nodeIpAndPackets := make(map[string][]resource.Packet)
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok {
			if ptype, ok := r.MetricLabels[string(MetricLabelType)]; ok {
				if version, ok := r.MetricLabels[string(MetricLabelVersion)]; ok &&
					version == string(metricCtx.Version) {
					packets := nodeIpAndPackets[nodeIp]
					packets = append(packets, resource.Packet{
						Type:   ptype,
						Values: getValuesWithTimestamp(r.Values, metricCtx.Period),
					})
					nodeIpAndPackets[nodeIp] = packets
				}
			}
		}
	}

	nodeNames, err := api.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID()))
	if err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	}

	var stats []*resource.PacketStat
	for nodeIp, packets := range nodeIpAndPackets {
		stat := &resource.PacketStat{Packets: packets, NodeName: nodeNames[nodeIp]}
		stat.SetID(nodeIp)
		stats = append(stats, stat)
	}
	return stats, nil
}

func (h *PacketStatHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	packetStat := ctx.Resource.(*resource.PacketStat)
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPPacketStats,
		PromQuery:      PromQueryVersionNode,
		NodeIP:         packetStat.GetID(),
	}

	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get packet stats failed: %s", err.Error()))
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get packet stats from prometheus failed: %s", err.Error()))
	}

	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok && nodeIp == metricCtx.NodeIP {
			if ptype, ok := r.MetricLabels[string(MetricLabelType)]; ok {
				if version, ok := r.MetricLabels[string(MetricLabelVersion)]; ok &&
					version == string(metricCtx.Version) {
					packetStat.Packets = append(packetStat.Packets, resource.Packet{
						Type:   ptype,
						Values: getValuesWithTimestamp(r.Values, metricCtx.Period),
					})
				}
			}
		}
	}

	if nodeNames, err := api.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID())); err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	} else {
		packetStat.NodeName = nodeNames[packetStat.GetID()]
	}

	return packetStat, nil
}

func (h *PacketStatHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.export(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *PacketStatHandler) export(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := exportMultiColunms(ctx, &MetricContext{
		NodeIP:         ctx.Resource.GetID(),
		PrometheusAddr: h.prometheusAddr,
		PromQuery:      PromQueryVersionNode,
		MetricName:     MetricNameDHCPPacketStats,
		MetricLabel:    MetricLabelType,
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("packet stats %s export action failed: %s", ctx.Resource.GetID(), err.Error()))
	} else {
		return result, nil
	}
}

func exportMultiColunms(ctx *restresource.Context, metricCtx *MetricContext) (interface{}, error) {
	filter, ok := ctx.Resource.GetAction().Input.(*resource.ExportFilter)
	if ok == false {
		return nil, fmt.Errorf("action input is not export filter")
	}

	timePeriod, err := parseTimePeriod(filter.From, filter.To)
	if err != nil {
		return nil, err
	}

	version, err := getDHCPVersionFromDHCPID(ctx.Resource.GetParent().GetID())
	if err != nil {
		return nil, err
	}

	metricCtx.Period = timePeriod
	metricCtx.Version = version
	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, fmt.Errorf("get node %s %s from prometheus failed: %s",
			metricCtx.NodeIP, metricCtx.MetricName, err.Error())
	}

	strMatrix, err := genHeaderAndStrMatrix(metricCtx, resp.Data.Results)
	if err != nil {
		return nil, fmt.Errorf("gen node %s %s header failed: %s",
			metricCtx.NodeIP, metricCtx.MetricName, err.Error())
	}

	filepath, err := exportFile(metricCtx, strMatrix)
	if err != nil {
		return nil, fmt.Errorf("export node %s %s failed: %s",
			metricCtx.NodeIP, metricCtx.MetricName, err.Error())
	}

	return &resource.FileInfo{Path: filepath}, nil
}

func genHeaderAndStrMatrix(ctx *MetricContext, results []PrometheusDataResult) ([][]string, error) {
	headers := []string{"日期"}
	var subnets map[string]struct{}
	if ctx.MetricLabel == MetricLabelSubnet {
		ss, err := getSubnetsFromDB(ctx.Version)
		if err != nil {
			return nil, fmt.Errorf("list subnets failed: %s", err.Error())
		}

		subnets = ss
	}

	var validResults []PrometheusDataResult
	for _, r := range results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok == false || nodeIp != ctx.NodeIP {
			continue
		}

		if label, ok := r.MetricLabels[string(ctx.MetricLabel)]; ok {
			switch ctx.MetricName {
			case MetricNameDHCPSubnetUsage, MetricNameDHCPLeaseCount:
				if _, ok := subnets[label]; ok == false {
					continue
				}
			case MetricNameDHCPPacketStats:
				if version, ok := r.MetricLabels[string(MetricLabelVersion)]; ok {
					if version == string(DHCPVersion4) {
						label = PacketLabelPrefixVersion4 + label
					} else {
						label = PacketLabelPrefixVersion6 + label
					}
				}
			}

			headers = append(headers, label)
			validResults = append(validResults, r)
		}
	}

	ctx.TableHeader = headers
	return genMultiStrMatrix(ctx, validResults), nil
}

func genMultiStrMatrix(ctx *MetricContext, results []PrometheusDataResult) [][]string {
	var values []string
	for i := 0; i < len(results); i++ {
		values = append(values, "0")
	}

	var matrix [][]string
	for i := ctx.Period.Begin; i <= ctx.Period.End; i += ctx.Period.Step {
		matrix = append(matrix, append([]string{time.Unix(int64(i), 0).Format(csvutil.TimeFormat)}, values...))
	}

	for i, r := range results {
		for _, vs := range r.Values {
			if timestamp, value := getTimestampAndValue(vs); timestamp != 0 &&
				timestamp >= ctx.Period.Begin && value != "0" {
				if ctx.MetricName == MetricNameDHCPSubnetUsage {
					if f, err := strconv.ParseFloat(value, 64); err == nil {
						value = fmt.Sprintf("%.4f", f)
					}
				}
				matrix[(timestamp-ctx.Period.Begin)/ctx.Period.Step][i+1] = value
			}
		}
	}

	return matrix
}

func getTimestampAndValue(values []interface{}) (int64, string) {
	var timestamp int64
	var value string
	for _, v := range values {
		if i, ok := v.(float64); ok {
			timestamp = int64(i)
		}

		if s, ok := v.(string); ok {
			value = s
		}
	}

	return timestamp, value
}
