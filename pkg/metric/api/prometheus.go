package api

import (
	"fmt"

	"github.com/cuityhj/gohelper/httpclient"
)

type PromQuery string

const (
	PromQueryName        PromQuery = "http://%s/api/v1/query_range?query=%s&start=%d&end=%d&step=%d"
	PromQueryVersion     PromQuery = "http://%s/api/v1/query_range?query=%s{version='%s'}&start=%d&end=%d&step=%d"
	PromQueryNode        PromQuery = "http://%s/api/v1/query_range?query=%s{node='%s'}&start=%d&end=%d&step=%d"
	PromQueryVersionNode PromQuery = "http://%s/api/v1/query_range?query=%s{version='%s',node='%s'}&start=%d&end=%d&step=%d"
)

type PrometheusResponse struct {
	Status string         `json:"status"`
	Data   PrometheusData `json:"data"`
}

type PrometheusData struct {
	Results []PrometheusDataResult `json:"result"`
}

type PrometheusDataResult struct {
	MetricLabels map[string]string `json:"metric"`
	Values       [][]interface{}   `json:"values"`
}

func prometheusRequest(ctx *MetricContext) (*PrometheusResponse, error) {
	var resp PrometheusResponse
	if err := httpclient.GetHttpClient().Get(&httpclient.HttpContext{
		URL:      genPrometheusUrl(ctx),
		Response: &resp,
	}); err != nil {
		return nil, err
	}

	if resp.Status != "success" {
		return nil, fmt.Errorf("get node %s %s failed with status: %s",
			ctx.NodeIP, ctx.MetricName, resp.Status)
	}

	return &resp, nil
}

func genPrometheusUrl(ctx *MetricContext) string {
	switch ctx.PromQuery {
	case PromQueryVersion:
		return fmt.Sprintf(string(ctx.PromQuery), ctx.PrometheusAddr, ctx.MetricName,
			ctx.Version, ctx.Period.Begin, ctx.Period.End, ctx.Period.Step)
	case PromQueryNode:
		return fmt.Sprintf(string(ctx.PromQuery), ctx.PrometheusAddr, ctx.MetricName,
			ctx.NodeIP, ctx.Period.Begin, ctx.Period.End, ctx.Period.Step)
	case PromQueryVersionNode:
		return fmt.Sprintf(string(ctx.PromQuery), ctx.PrometheusAddr, ctx.MetricName,
			ctx.Version, ctx.NodeIP, ctx.Period.Begin, ctx.Period.End, ctx.Period.Step)
	case PromQueryName:
		return fmt.Sprintf(string(ctx.PromQuery), ctx.PrometheusAddr, ctx.MetricName,
			ctx.Period.Begin, ctx.Period.End, ctx.Period.Step)
	default:
		return fmt.Sprintf(string(ctx.PromQuery), ctx.PrometheusAddr, ctx.MetricName,
			ctx.Period.Begin, ctx.Period.End, ctx.Period.Step)
	}
}
