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

type LeaseTotalHandler struct {
	prometheusAddr string
}

func NewLeaseTotalHandler(conf *config.DHCPConfig) *LeaseTotalHandler {
	return &LeaseTotalHandler{prometheusAddr: conf.Prometheus.Addr}
}

func (h *LeaseTotalHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	nodeIpAndValues, err := getNodeIpAndValuesFromPrometheus(ctx, &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLeaseCountTotal,
		PromQuery:      PromQueryVersion,
	})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			"get leases total count from prometheus failed: "+err.Error())
	}

	nodeNames, err := api.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID()))
	if err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	}

	var leases []*resource.LeaseTotal
	for nodeIp, values := range nodeIpAndValues {
		lease := &resource.LeaseTotal{Values: values, NodeName: nodeNames[nodeIp]}
		lease.SetID(nodeIp)
		leases = append(leases, lease)
	}

	return leases, nil
}

func (h *LeaseTotalHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	lease := ctx.Resource.(*resource.LeaseTotal)
	values, err := getValuesFromPrometheus(ctx, &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLeaseCountTotal,
		PromQuery:      PromQueryVersionNode,
		NodeIP:         lease.GetID(),
	})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("get leases count with node %s failed: %s", lease.GetID(), err.Error()))
	}

	if nodeNames, err := api.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID())); err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	} else {
		lease.NodeName = nodeNames[lease.GetID()]
	}

	lease.Values = values
	return lease, nil
}

func (h *LeaseTotalHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.export(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

var TableHeaderLeaseTotal = []string{"日期", "租赁总数"}

func (h *LeaseTotalHandler) export(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := exportTwoColumns(ctx, &MetricContext{
		NodeIP:         ctx.Resource.GetID(),
		PrometheusAddr: h.prometheusAddr,
		PromQuery:      PromQueryVersionNode,
		MetricName:     MetricNameDHCPLeaseCountTotal,
		TableHeader:    TableHeaderLeaseTotal,
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("leases count %s export action failed: %s", ctx.Resource.GetID(), err.Error()))
	} else {
		return result, nil
	}
}
