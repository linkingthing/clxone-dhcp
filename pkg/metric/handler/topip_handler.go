package handler

import (
	"fmt"
	"strconv"

	resterror "github.com/zdnscloud/gorest/error"

	"github.com/linkingthing/clxone-dhcp/pkg/esclient"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

var (
	TableHeaderTopIP = []string{"请求源地址", "请求次数"}
)

const (
	TopIpAggsName   = "top10Ips"
	MetricNameTopIp = "lx_dns_top10_ips"
)

func getTopTenIps(ctx *MetricContext) (*resource.Dns, *resterror.APIError) {
	ctx.AggsName = TopIpAggsName
	ctx.AggsKeyword = esclient.SrcIpKeyWord
	resp, err := requestElasticsearchWithCtx(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s top ten ips from elasticsearch failed: %s", ctx.NodeIP, err.Error()))
	}

	var topIps []resource.TopIp
	if tops, ok := resp.Aggregations[TopIpAggsName]; ok {
		for _, bucket := range tops.Buckets {
			topIps = append(topIps, resource.TopIp{
				Ip:    bucket.Key,
				Count: bucket.DocCount,
			})
		}
	}

	dns := &resource.Dns{TopTenIps: topIps}
	dns.SetID(resource.ResourceIDTopTenIPs)
	return dns, nil
}

func requestElasticsearchWithCtx(ctx *MetricContext) (*esclient.ElasticsearchResponse, error) {
	req := &esclient.ElasticsearchRequest{
		Size: 0,
		Query: esclient.ElasticsearchQuery{
			Bool: esclient.ElasticsearchBool{
				Must: []interface{}{
					esclient.ElasticsearchMustRange{
						Range: esclient.ElasticsearchRange{
							Timestamp: esclient.ElasticsearchRangeTimestamp{
								GTE:    ctx.Period.From,
								LTE:    ctx.Period.To,
								Format: esclient.TimestampFormat,
							},
						},
					},
				},
			},
		},
		Aggs: map[string]esclient.ElasticsearchAggs{
			ctx.AggsName: esclient.ElasticsearchAggs{
				Term: esclient.ElasticsearchAggsTerm{
					Field: ctx.AggsKeyword,
					Order: esclient.AggsTermOrder,
					Size:  10,
				},
			},
		},
	}

	return esclient.GetESClient().Request(req, "dns_"+ctx.NodeIP)
}

func (h *DnsHandler) exportTopTenIps(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = MetricNameTopIp
	ctx.TableHeader = TableHeaderTopIP
	ctx.AggsName = TopIpAggsName
	ctx.AggsKeyword = esclient.SrcIpKeyWord
	return h.exportTopMetrics(ctx)
}

func (h *DnsHandler) exportTopMetrics(ctx *MetricContext) (interface{}, *resterror.APIError) {
	resp, err := requestElasticsearchWithCtx(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("export node %s %s to csv file failed: %s", ctx.NodeIP, ctx.AggsKeyword, err.Error()))
	}

	var matrix [][]string
	if tops, ok := resp.Aggregations[ctx.AggsName]; ok {
		for _, bucket := range tops.Buckets {
			matrix = append(matrix, []string{bucket.Key, strconv.FormatUint(bucket.DocCount, 10)})
		}
	}

	filepath, err := exportFile(ctx, matrix)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export node %s %s to csv file failed: %s", ctx.NodeIP, ctx.AggsKeyword, err.Error()))
	}

	return &resource.FileInfo{Path: filepath}, nil
}
