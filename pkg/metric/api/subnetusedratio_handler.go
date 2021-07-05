package api

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	dhcpresource "github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type SubnetUsedRatioHandler struct {
	prometheusAddr string
}

func NewSubnetUsedRatioHandler(conf *config.DHCPConfig) *SubnetUsedRatioHandler {
	return &SubnetUsedRatioHandler{prometheusAddr: conf.Prometheus.Addr}
}

func (h *SubnetUsedRatioHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPSubnetUsage,
		PromQuery:      PromQueryVersion,
	}

	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get packet stats failed: %s", err.Error()))
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnets used ratio from prometheus failed: %s", err.Error()))
	}

	subnets, err := getSubnetsFromDB(metricCtx.Version)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnets from db failed: %s", err.Error()))
	}

	nodeIpAndSubnetUsages := make(map[string]resource.SubnetUsages)
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok {
			if subnetId, ok := r.MetricLabels[string(MetricLabelSubnetId)]; ok {
				if subnet, ok := subnets[subnetId]; ok {
					subnets := nodeIpAndSubnetUsages[nodeIp]
					subnets = append(subnets, resource.SubnetUsage{
						Subnet:     subnet,
						UsedRatios: getRatiosWithTimestamp(r.Values, metricCtx.Period),
					})
					nodeIpAndSubnetUsages[nodeIp] = subnets
				}
			}
		}
	}

	var subnetUsedRatios []*resource.SubnetUsedRatio
	for nodeIp, subnets := range nodeIpAndSubnetUsages {
		sort.Sort(subnets)
		subnetUsedRatio := &resource.SubnetUsedRatio{Subnets: subnets}
		subnetUsedRatio.SetID(nodeIp)
		subnetUsedRatios = append(subnetUsedRatios, subnetUsedRatio)
	}

	return subnetUsedRatios, nil
}

func getSubnetsFromDB(version DHCPVersion) (map[string]string, error) {
	idAndSubnets := make(map[string]string)
	if version == DHCPVersion4 {
		var subnets []*dhcpresource.Subnet4
		if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
			return tx.Fill(nil, &subnets)
		}); err != nil {
			return nil, err
		}

		for _, subnet := range subnets {
			idAndSubnets[subnet.GetID()] = subnet.Subnet
		}
	} else {
		var subnets []*dhcpresource.Subnet6
		if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
			return tx.Fill(nil, &subnets)
		}); err != nil {
			return nil, err
		}

		for _, subnet := range subnets {
			idAndSubnets[subnet.GetID()] = subnet.Subnet
		}
	}

	return idAndSubnets, nil
}

func getRatiosWithTimestamp(values [][]interface{}, period *TimePeriod) []resource.RatioWithTimestamp {
	var ratioWithTimestamps []resource.RatioWithTimestamp
	for i := period.Begin; i <= period.End; i += period.Step {
		ratioWithTimestamps = append(ratioWithTimestamps, resource.RatioWithTimestamp{
			Timestamp: restresource.ISOTime(time.Unix(i, 0)),
			Ratio:     "0",
		})
	}

	for _, vs := range values {
		if t, s := getTimestampAndValue(vs); t != 0 {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				ratioWithTimestamps[(t-period.Begin)/period.Step].Ratio = fmt.Sprintf("%.4f", f)
			}
		}
	}

	return ratioWithTimestamps
}

func (h *SubnetUsedRatioHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetUsedRatio := ctx.Resource.(*resource.SubnetUsedRatio)
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPSubnetUsage,
		PromQuery:      PromQueryVersionNode,
		NodeIP:         subnetUsedRatio.GetID(),
	}

	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get packet stats failed: %s", err.Error()))
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnets used ratio from prometheus failed: %s", err.Error()))
	}

	subnets, err := getSubnetsFromDB(metricCtx.Version)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnets from db failed: %s", err.Error()))
	}

	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok && metricCtx.NodeIP == nodeIp {
			if subnetId, ok := r.MetricLabels[string(MetricLabelSubnetId)]; ok {
				if subnet, ok := subnets[subnetId]; ok {
					subnetUsedRatio.Subnets = append(subnetUsedRatio.Subnets, resource.SubnetUsage{
						Subnet:     subnet,
						UsedRatios: getRatiosWithTimestamp(r.Values, metricCtx.Period),
					})
				}
			}
		}
	}

	sort.Sort(resource.SubnetUsages(subnetUsedRatio.Subnets))
	return subnetUsedRatio, nil
}

func (h *SubnetUsedRatioHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.export(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *SubnetUsedRatioHandler) export(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := exportMultiColunms(ctx, &MetricContext{
		NodeIP:         ctx.Resource.GetID(),
		PrometheusAddr: h.prometheusAddr,
		PromQuery:      PromQueryVersionNode,
		MetricName:     MetricNameDHCPSubnetUsage,
		MetricLabel:    MetricLabelSubnetId,
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("subnet usage %s export action failed: %s", ctx.Resource.GetID(), err.Error()))
	} else {
		return result, nil
	}
}
