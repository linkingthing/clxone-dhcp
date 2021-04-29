package handler

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"

	"github.com/linkingthing/clxone-dhcp/pkg/esclient"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

const (
	TopDomainAggsName   = "top10Domains"
	MetricNameTopDomain = "lx_dns_top10_domains"
)

var TableHeaderTopDomain = []string{"请求源域名", "请求次数"}

func getTopTenDomains(ctx *MetricContext) (*resource.Dns, *resterror.APIError) {
	ctx.AggsName = TopDomainAggsName
	ctx.AggsKeyword = esclient.DomainKeyWord
	resp, err := requestElasticsearchWithCtx(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s top ten domains from elasticsearch failed: %s", ctx.NodeIP, err.Error()))
	}

	var topDomains []resource.TopDomain
	if tops, ok := resp.Aggregations[TopDomainAggsName]; ok {
		for _, bucket := range tops.Buckets {
			topDomains = append(topDomains, resource.TopDomain{
				Domain: bucket.Key,
				Count:  bucket.DocCount,
			})
		}
	}

	dns := &resource.Dns{TopTenDomains: topDomains}
	dns.SetID(resource.ResourceIDTopTenDomains)
	return dns, nil
}

func (h *DnsHandler) exportTopTenDomains(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = MetricNameTopDomain
	ctx.TableHeader = TableHeaderTopDomain
	ctx.AggsName = TopDomainAggsName
	ctx.AggsKeyword = esclient.DomainKeyWord
	return h.exportTopMetrics(ctx)
}
