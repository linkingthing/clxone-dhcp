package service

import (
	"fmt"
	"strconv"
	"time"

	"github.com/linkingthing/cement/log"
	"github.com/linkingthing/clxone-utils/excel"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

const (
	PacketLabelPrefixVersion4 = "pkt4-"
	PacketLabelPrefixVersion6 = "pkt6-"
)

type PacketStatService struct {
	prometheusAddr string
}

func NewPacketStatService(config *config.DHCPConfig) *PacketStatService {
	return &PacketStatService{prometheusAddr: config.Prometheus.Addr}
}

func (h *PacketStatService) List(ctx *restresource.Context) (interface{}, error) {
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPPacketStats,
		PromQuery:      PromQueryVersion,
	}

	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, err
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, err
	}

	nodeIpAndPackets := make(map[string][]resource.Packet)
	for _, r := range resp.Data.Results {
		if hostname, ok := r.MetricLabels[string(MetricLabelNode)]; ok {
			if ptype, ok := r.MetricLabels[string(MetricLabelType)]; ok {
				if version, ok := r.MetricLabels[string(MetricLabelVersion)]; ok &&
					version == string(metricCtx.Version) {
					nodeIp, err := getDhcpNodeIP(hostname, IsDHCPVersion4(ctx.Resource.GetParent().GetID()))
					if err != nil {
						log.Warnf("get node ip err: %v", err)
						continue
					}
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

	nodeNames, err := service.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID()))
	if err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	}

	stats := make([]*resource.PacketStat, 0, len(nodeIpAndPackets))
	for nodeIp, packets := range nodeIpAndPackets {
		stat := &resource.PacketStat{Packets: packets, NodeName: nodeNames[nodeIp]}
		stat.SetID(nodeIp)
		stats = append(stats, stat)
	}
	return stats, nil
}

func (h *PacketStatService) Get(ctx *restresource.Context) (restresource.Resource, error) {
	packetStat := ctx.Resource.(*resource.PacketStat)
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPPacketStats,
		PromQuery:      PromQueryVersionNode,
		NodeIP:         packetStat.GetID(),
	}
	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, err
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, err
	}

	for _, r := range resp.Data.Results {
		if hostname, ok := r.MetricLabels[string(MetricLabelNode)]; ok && hostname == metricCtx.Hostname {
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

	if nodeNames, err := service.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID())); err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	} else {
		packetStat.NodeName = nodeNames[packetStat.GetID()]
	}

	return packetStat, nil
}

func (h *PacketStatService) Export(ctx *restresource.Context) (interface{}, error) {
	return exportMultiColunms(ctx, &MetricContext{
		NodeIP:         ctx.Resource.GetID(),
		PrometheusAddr: h.prometheusAddr,
		PromQuery:      PromQueryVersionNode,
		MetricName:     MetricNameDHCPPacketStats,
		MetricLabel:    MetricLabelType,
	})
}

func exportMultiColunms(ctx *restresource.Context, metricCtx *MetricContext) (interface{}, error) {
	filter, ok := ctx.Resource.GetAction().Input.(*resource.ExportFilter)
	if !ok {
		return nil, errorno.ErrInvalidFormat(errorno.ErrNameMetric, errorno.ErrNameExport)
	}

	timePeriod, err := parseTimePeriod(filter.From, filter.To)
	if err != nil {
		return nil, err
	}

	version, err := getDHCPVersionFromDHCPID(ctx.Resource.GetParent().GetID())
	if err != nil {
		return nil, err
	}
	metricCtx.Version = version
	hostname, err := getDhcpHostname(metricCtx.NodeIP, metricCtx.Version == DHCPVersion4)
	if err != nil {
		return nil, err
	}
	metricCtx.Hostname = hostname
	metricCtx.Period = timePeriod

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, err
	}

	strMatrix, err := genHeaderAndStrMatrix(metricCtx, resp.Data.Results)
	if err != nil {
		return nil, err
	}

	filepath, err := exportFile(metricCtx, strMatrix)
	if err != nil {
		return nil, err
	}

	return &resource.FileInfo{Path: filepath}, nil
}

func genHeaderAndStrMatrix(ctx *MetricContext, results []PrometheusDataResult) ([][]string, error) {
	headers := []string{"日期"}
	var subnets map[string]struct{}
	if ctx.MetricLabel == MetricLabelSubnet {
		ss, err := getSubnetsFromDB(ctx.Version)
		if err != nil {
			return nil, err
		}

		subnets = ss
	}

	var validResults []PrometheusDataResult
	for _, r := range results {
		if hostname, ok := r.MetricLabels[string(MetricLabelNode)]; !ok || hostname != ctx.Hostname {
			continue
		}

		if label, ok := r.MetricLabels[string(ctx.MetricLabel)]; ok {
			switch ctx.MetricName {
			case MetricNameDHCPSubnetUsage, MetricNameDHCPLeaseCount:
				if _, ok := subnets[label]; !ok {
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
		matrix = append(matrix, append([]string{time.Unix(i, 0).Format(excel.TimeFormat)}, values...))
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

func getDhcpNodeIP(hostname string, isDhcpV4 bool) (string, error) {
	if len(hostname) == 0 {
		return "", nil
	}

	node, err := kafka.GetDHCPAgentService().GetDHCPNodeByHostname(hostname)
	if err != nil {
		return "", fmt.Errorf("get dhcp hostname %s failed:%s", hostname, err.Error())
	}

	if isDhcpV4 {
		return node.GetIpv4(), nil
	}
	return node.GetIpv6(), nil
}

func getDhcpHostname(nodeIp string, isDhcpV4 bool) (string, error) {
	if len(nodeIp) == 0 {
		return "", nil
	}

	hostname, err := kafka.GetDHCPAgentService().GetDHCPHostnameByNode(nodeIp, isDhcpV4)
	if err != nil {
		return "", fmt.Errorf("get hostname from ip %s failed:%s", nodeIp, err.Error())
	}
	return hostname, nil
}
