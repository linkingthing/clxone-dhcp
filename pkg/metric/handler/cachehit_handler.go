package handler

import (
	"fmt"

	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	dnsresource "github.com/linkingthing/clxone-dhcp/pkg/dns/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	agentmetric "github.com/linkingthing/ddi-agent/pkg/metric"
)

var TableHeaderCacheHits = []string{"日期", "缓存命中率"}

func getCacheHitRatio(ctx *MetricContext) (*resource.Dns, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDNSCacheHitsRatioTotal
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s cache hit ratios total failed: %s", ctx.NodeIP, err.Error()))
	}

	var cacheHitRatios []resource.RatioWithTimestamp
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok || nodeIp == ctx.NodeIP {
			cacheHitRatios = getRatiosWithTimestamp(r.Values, ctx.Period)
			break
		}
	}

	dns := &resource.Dns{CacheHitRatio: resource.CacheHitRatio{Ratios: cacheHitRatios}}
	dns.SetID(resource.ResourceIDCacheHitRatio)
	var viewCacheHitRatios []resource.ViewCacheHitRatio
	views, err := getViewsFromDB()
	if err != nil {
		log.Warnf("list views from db failed: %s", err.Error())
		return dns, nil
	}

	ctx.MetricName = agentmetric.MetricNameDNSCacheHitsRatio
	resp, err = prometheusRequest(ctx)
	if err != nil {
		log.Warnf("get node %s cache hit ratios for each view failed: %s", ctx.NodeIP, err.Error())
		return dns, nil
	}

	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok == false || nodeIp != ctx.NodeIP {
			continue
		}

		if viewId, ok := r.MetricLabels[agentmetric.MetricLabelView]; ok {
			if _, ok := views[viewId]; ok {
				viewCacheHitRatios = append(viewCacheHitRatios, resource.ViewCacheHitRatio{
					View:   viewId,
					Ratios: getRatiosWithTimestamp(r.Values, ctx.Period),
				})
			}
		}
	}

	dns.CacheHitRatio.ViewRatios = viewCacheHitRatios
	return dns, nil
}

func getViewsFromDB() (map[string]struct{}, error) {
	var views []*dnsresource.View
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &views)
	}); err != nil {
		return nil, err
	}

	idAndViews := make(map[string]struct{})
	for _, view := range views {
		idAndViews[view.GetID()] = struct{}{}
	}

	return idAndViews, nil
}

func exportCacheHitRatio(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = agentmetric.MetricNameDNSCacheHitsRatioTotal
	ctx.TableHeader = TableHeaderCacheHits
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s cache hit ratios total failed: %s", ctx.NodeIP, err.Error()))
	}

	var result PrometheusDataResult
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok || nodeIp == ctx.NodeIP {
			result = r
			break
		}
	}

	views, err := getViewsFromDB()
	if err != nil {
		log.Warnf("list views from db failed: %s", err.Error())
		return exportTwoColumnsWithResult(ctx, result)
	}

	ctx.MetricName = agentmetric.MetricNameDNSCacheHitsRatio
	resp, err = prometheusRequest(ctx)
	if err != nil {
		log.Warnf("get node %s cache hit ratios for each view failed: %s", ctx.NodeIP, err.Error())
		ctx.MetricName = agentmetric.MetricNameDNSCacheHitsRatioTotal
		return exportTwoColumnsWithResult(ctx, result)
	}

	validResults := []PrometheusDataResult{result}
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[agentmetric.MetricLabelNode]; ok == false || nodeIp != ctx.NodeIP {
			continue
		}

		if viewId, ok := r.MetricLabels[agentmetric.MetricLabelView]; ok {
			if _, ok := views[viewId]; ok == false {
				continue
			}

			ctx.TableHeader = append(ctx.TableHeader, viewId)
			validResults = append(validResults, r)
		}
	}

	filepath, err := exportFile(ctx, genMultiStrMatrix(ctx, validResults))
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("export node %s %s failed: %s",
			ctx.NodeIP, ctx.MetricName, err.Error()))
	}

	return &resource.FileInfo{Path: filepath}, nil
}
