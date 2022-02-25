package service

import (
	"fmt"

	"github.com/linkingthing/cement/log"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type LeaseService struct {
	prometheusAddr string
}

func NewLeaseService(config *config.DHCPConfig) *LeaseService {
	return &LeaseService{prometheusAddr: config.Prometheus.Addr}
}

func (h *LeaseService) List(ctx *restresource.Context) (interface{}, error) {
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLeaseCount,
		PromQuery:      PromQueryVersion,
	}

	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, fmt.Errorf("get subnets leases count failed: %s", err.Error())
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, fmt.Errorf("get subnets leases count from prometheus failed: %s", err.Error())
	}

	subnets, err := getSubnetsFromDB(metricCtx.Version)
	if err != nil {
		return nil, fmt.Errorf("list subnets from db failed: %s", err.Error())
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

	nodeNames, err := service.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID()))
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

func (h *LeaseService) Get(ctx *restresource.Context) (restresource.Resource, error) {
	lease := ctx.Resource.(*resource.Lease)
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLeaseCount,
		PromQuery:      PromQueryVersionNode,
		NodeIP:         lease.GetID(),
	}

	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, fmt.Errorf("get subnet leases count failed: %s", err.Error())
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, fmt.Errorf("get subnet used ratio from prometheus failed: %s", err.Error())
	}

	subnets, err := getSubnetsFromDB(metricCtx.Version)
	if err != nil {
		return nil, fmt.Errorf("list subnets from db failed: %s", err.Error())
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

	if nodeNames, err := service.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID())); err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	} else {
		lease.NodeName = nodeNames[lease.GetID()]
	}

	return lease, nil
}

func (h *LeaseService) Export(ctx *restresource.Context) (interface{}, error) {
	if result, err := exportMultiColunms(ctx, &MetricContext{
		NodeIP:         ctx.Resource.GetID(),
		PrometheusAddr: h.prometheusAddr,
		PromQuery:      PromQueryVersionNode,
		MetricName:     MetricNameDHCPLeaseCount,
		MetricLabel:    MetricLabelSubnet,
	}); err != nil {
		return nil, fmt.Errorf("leases %s export ulticolunm failed: %s", ctx.Resource.GetID(), err.Error())
	} else {
		return result, nil
	}
}
