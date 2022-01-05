package api

import (
	"fmt"

	"github.com/linkingthing/cement/log"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/api"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type LeaseHandler struct {
	prometheusAddr string
}

func NewLeaseHandler(conf *config.DHCPConfig) *LeaseHandler {
	return &LeaseHandler{prometheusAddr: conf.Prometheus.Addr}
}

func (h *LeaseHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLeaseCount,
		PromQuery:      PromQueryVersion,
	}

	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnets leases count failed: %s", err.Error()))
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnets leases count from prometheus failed: %s", err.Error()))
	}

	subnets, err := getSubnetsFromDB(metricCtx.Version)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnets from db failed: %s", err.Error()))
	}

	nodeIpAndSubnetLeases := make(map[string][]resource.SubnetLease)
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok {
			if subnet, ok := r.MetricLabels[string(MetricLabelSubnet)]; ok {
				if _, ok := subnets[subnet]; ok {
					subnets := nodeIpAndSubnetLeases[nodeIp]
					subnets = append(subnets, resource.SubnetLease{
						Subnet: subnet,
						Values: getValuesWithTimestamp(r.Values, metricCtx.Period),
					})
					nodeIpAndSubnetLeases[nodeIp] = subnets
				}
			}
		}
	}

	nodeNames, err := api.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID()))
	if err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	}

	var leases []*resource.Lease
	for nodeIp, subnets := range nodeIpAndSubnetLeases {
		lease := &resource.Lease{Subnets: subnets, NodeName: nodeNames[nodeIp]}
		lease.SetID(nodeIp)
		leases = append(leases, lease)
	}

	return leases, nil
}

func (h *LeaseHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	lease := ctx.Resource.(*resource.Lease)
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLeaseCount,
		PromQuery:      PromQueryVersionNode,
		NodeIP:         lease.GetID(),
	}

	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet leases count failed: %s", err.Error()))
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet used ratio from prometheus failed: %s", err.Error()))
	}

	subnets, err := getSubnetsFromDB(metricCtx.Version)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnets from db failed: %s", err.Error()))
	}

	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok && metricCtx.NodeIP == nodeIp {
			if subnet, ok := r.MetricLabels[string(MetricLabelSubnet)]; ok {
				if _, ok := subnets[subnet]; ok {
					lease.Subnets = append(lease.Subnets, resource.SubnetLease{
						Subnet: subnet,
						Values: getValuesWithTimestamp(r.Values, metricCtx.Period),
					})
				}
			}
		}
	}

	if nodeNames, err := api.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID())); err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	} else {
		lease.NodeName = nodeNames[lease.GetID()]
	}

	return lease, nil
}

func (h *LeaseHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.export(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *LeaseHandler) export(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := exportMultiColunms(ctx, &MetricContext{
		NodeIP:         ctx.Resource.GetID(),
		PrometheusAddr: h.prometheusAddr,
		PromQuery:      PromQueryVersionNode,
		MetricName:     MetricNameDHCPLeaseCount,
		MetricLabel:    MetricLabelSubnet,
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("leases count %s export action failed: %s", ctx.Resource.GetID(), err.Error()))
	} else {
		return result, nil
	}
}
