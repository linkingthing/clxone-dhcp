package api

import (
	"fmt"
	"strconv"
	"time"

	resterror "github.com/zdnscloud/gorest/error"

	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	PacketLabelPrefixVersion4 = "pkt4-"
	PacketLabelPrefixVersion6 = "pkt6-"
	DHCPVersion4              = "4"
	DHCP4StatsDiscover        = "pkt4-discover-received"
	DHCP4StatsOffer           = "pkt4-offer-sent"
	DHCP4StatsRequest         = "pkt4-request-received"
	DHCP4StatsAck             = "pkt4-ack-sent"
	DHCP4PacketTypeDiscover   = "discover"
	DHCP4PacketTypeOffer      = "offer"
	DHCP4PacketTypeRequest    = "request"
	DHCP4PacketTypeAck        = "ack"

	DHCPVersion6             = "6"
	DHCP6StatsSolicit        = "pkt6-solicit-received"
	DHCP6StatsAdvertise      = "pkt6-advertise-sent"
	DHCP6StatsRequest        = "pkt6-request-received"
	DHCP6StatsReply          = "pkt6-reply-sent"
	DHCP6PacketTypeSolicit   = "solicit"
	DHCP6PacketTypeAdvertise = "advertise"
	DHCP6PacketTypeRequest   = "request"
	DHCP6PacketTypeReply     = "reply"
)

func getPackets(ctx *MetricContext) (*resource.Dhcp, *resterror.APIError) {
	ctx.MetricName = MetricNameDHCPPacketsStats
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s packet stats failed: %s", ctx.NodeIP, err.Error()))
	}

	var packets []resource.Packet
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[MetricLabelNode]; ok == false || nodeIp != ctx.NodeIP {
			continue
		}

		if ptype, ok := r.MetricLabels[MetricLabelType]; ok {
			if version, ok := r.MetricLabels[MetricLabelVersion]; ok {
				packets = append(packets, resource.Packet{
					Version: version,
					Type:    ptype,
					Values:  getValuesWithTimestamp(r.Values, ctx.Period),
				})
			}
		}
	}

	dhcp := &resource.Dhcp{Packets: packets}
	dhcp.SetID(resource.ResourceIDPackets)
	return dhcp, nil
}

func exportPackets(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = MetricNameDHCPPacketsStats
	ctx.MetricLabel = MetricLabelType
	return exportMultiColunms(ctx)
}

func exportMultiColunms(ctx *MetricContext) (interface{}, *resterror.APIError) {
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("get node %s %s from prometheus failed: %s", ctx.NodeIP, ctx.MetricName, err.Error()))
	}

	strMatrix, err := genHeaderAndStrMatrix(ctx, resp.Data.Results)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s %s from prometheus failed: %s", ctx.NodeIP, ctx.MetricName, err.Error()))
	}

	filepath, err := exportFile(ctx, strMatrix)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("export node %s %s failed: %s",
			ctx.NodeIP, ctx.MetricName, err.Error()))
	}

	return &resource.FileInfo{Path: filepath}, nil
}

func genHeaderAndStrMatrix(ctx *MetricContext, results []PrometheusDataResult) ([][]string, error) {
	headers := []string{"日期"}
	var subnets map[string]string
	if ctx.MetricLabel == MetricLabelSubnetId {
		ss, err := getSubnetsFromDB()
		if err != nil {
			return nil, fmt.Errorf("list subnets failed: %s", err.Error())
		}

		subnets = ss
	}

	var validResults []PrometheusDataResult
	for _, r := range results {
		if nodeIp, ok := r.MetricLabels[MetricLabelNode]; ok == false || nodeIp != ctx.NodeIP {
			continue
		}

		if label, ok := r.MetricLabels[ctx.MetricLabel]; ok {
			if ctx.MetricName == MetricNameDHCPUsages {
				subnet, ok := subnets[label]
				if ok == false {
					continue
				}
				label = subnet
			} else if ctx.MetricName == MetricNameDHCPPacketsStats {
				if version, ok := r.MetricLabels[MetricLabelVersion]; ok {
					if version == DHCPVersion4 {
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
		matrix = append(matrix, append([]string{time.Unix(int64(i), 0).Format(util.TimeFormat)}, values...))
	}

	for i, r := range results {
		for _, vs := range r.Values {
			if timestamp, value := getTimestampAndValue(vs); timestamp != 0 && timestamp >= ctx.Period.Begin {
				if f, err := strconv.ParseFloat(value, 64); err == nil {
					value = fmt.Sprintf("%.4f", f)
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
