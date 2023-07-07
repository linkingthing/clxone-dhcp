package service

import (
	"github.com/linkingthing/cement/log"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type LeaseTotalService struct {
	prometheusAddr string
}

func NewLeaseTotalService(config *config.DHCPConfig) *LeaseTotalService {
	return &LeaseTotalService{prometheusAddr: config.Prometheus.Addr}
}

func (h *LeaseTotalService) List(ctx *restresource.Context) (interface{}, error) {
	nodeIpAndValues, err := getNodeIpAndValuesFromPrometheus(ctx, &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLeaseCountTotal,
		PromQuery:      PromQueryVersion,
	})
	if err != nil {
		return nil, err
	}

	nodeNames, err := service.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID()))
	if err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	}

	leases := make([]*resource.LeaseTotal, 0, len(nodeIpAndValues))
	for nodeIp, values := range nodeIpAndValues {
		lease := &resource.LeaseTotal{Values: values, NodeName: nodeNames[nodeIp]}
		lease.SetID(nodeIp)
		leases = append(leases, lease)
	}

	return leases, nil
}

func (h *LeaseTotalService) Get(ctx *restresource.Context) (restresource.Resource, error) {
	lease := ctx.Resource.(*resource.LeaseTotal)
	values, err := getValuesFromPrometheus(ctx, &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLeaseCountTotal,
		PromQuery:      PromQueryVersionNode,
		NodeIP:         lease.GetID(),
	})
	if err != nil {
		return nil, err
	}

	if nodeNames, err := service.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID())); err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	} else {
		lease.NodeName = nodeNames[lease.GetID()]
	}

	lease.Values = values
	return lease, nil
}

var TableHeaderLeaseTotal = []string{"日期", "租赁总数"}

func (h *LeaseTotalService) Export(ctx *restresource.Context) (interface{}, error) {
	return exportTwoColumns(ctx, &MetricContext{
		NodeIP:         ctx.Resource.GetID(),
		PrometheusAddr: h.prometheusAddr,
		PromQuery:      PromQueryVersionNode,
		MetricName:     MetricNameDHCPLeaseCountTotal,
		TableHeader:    TableHeaderLeaseTotal,
	})
}
