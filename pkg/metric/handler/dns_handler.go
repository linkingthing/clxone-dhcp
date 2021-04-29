package handler

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type DnsHandler struct {
	prometheusAddr string
}

func NewDnsHandler(conf *config.DDIControllerConfig) *DnsHandler {
	return &DnsHandler{
		prometheusAddr: conf.Prometheus.Addr,
	}
}

func (h *DnsHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	topTenIps := &resource.Dns{}
	topTenIps.SetID(resource.ResourceIDTopTenIPs)
	topTenDomains := &resource.Dns{}
	topTenDomains.SetID(resource.ResourceIDTopTenDomains)
	qps := &resource.Dns{}
	qps.SetID(resource.ResourceIDQPS)
	cachehit := &resource.Dns{}
	cachehit.SetID(resource.ResourceIDCacheHitRatio)
	queryTypeRatios := &resource.Dns{}
	queryTypeRatios.SetID(resource.ResourceIDQueryTypeRatios)
	resolvedRatios := &resource.Dns{}
	resolvedRatios.SetID(resource.ResourceIDResolvedRatios)
	return []*resource.Dns{topTenIps, topTenDomains, qps, cachehit, queryTypeRatios, resolvedRatios}, nil
}

func (h *DnsHandler) genDNSMetricContext(nodeIP string, period *TimePeriodParams) *MetricContext {
	return &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		NodeIP:         nodeIP,
		Period:         period,
	}
}

func (h *DnsHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dnsID := ctx.Resource.GetID()
	period, err := getTimePeriodParamFromFilter(ctx.GetFilters())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("invalid time format: %s", err.Error()))
	}

	context := h.genDNSMetricContext(ctx.Resource.GetParent().GetID(), period)
	switch dnsID {
	case resource.ResourceIDTopTenIPs:
		return getTopTenIps(context)
	case resource.ResourceIDTopTenDomains:
		return getTopTenDomains(context)
	case resource.ResourceIDQPS:
		return getQps(context)
	case resource.ResourceIDCacheHitRatio:
		return getCacheHitRatio(context)
	case resource.ResourceIDQueryTypeRatios:
		return getQueryTypeRatios(context)
	case resource.ResourceIDResolvedRatios:
		return getResolvedRatios(context)
	default:
		return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found dns resource %s", dnsID))
	}
}

func (h *DnsHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.export(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *DnsHandler) export(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	period, err := getTimePeriodParamFromActionInput(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("parse action input failed: %s", err.Error()))
	}

	dnsID := ctx.Resource.GetID()
	context := h.genDNSMetricContext(ctx.Resource.GetParent().GetID(), period)
	switch dnsID {
	case resource.ResourceIDTopTenIPs:
		return h.exportTopTenIps(context)
	case resource.ResourceIDTopTenDomains:
		return h.exportTopTenDomains(context)
	case resource.ResourceIDQPS:
		return exportQps(context)
	case resource.ResourceIDCacheHitRatio:
		return exportCacheHitRatio(context)
	case resource.ResourceIDQueryTypeRatios:
		return exportQueryTypeRatios(context)
	case resource.ResourceIDResolvedRatios:
		return exportResolvedRatios(context)
	default:
		return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found dns resource %s", dnsID))
	}
}
