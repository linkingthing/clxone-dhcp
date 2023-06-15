package service

import (
	"fmt"

	httputil "github.com/linkingthing/clxone-utils/http"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

const HttpScheme = "https://"

type PromQuery string

const (
	PromQueryName        PromQuery = "%s%s/api/v1/query_range?query=%s&start=%d&end=%d&step=%d"
	PromQueryVersion     PromQuery = "%s%s/api/v1/query_range?query=%s{version='%s'}&start=%d&end=%d&step=%d"
	PromQueryNode        PromQuery = "%s%s/api/v1/query_range?query=%s{node='%s'}&start=%d&end=%d&step=%d"
	PromQueryVersionNode PromQuery = "%s%s/api/v1/query_range?query=%s{version='%s',node='%s'}&start=%d&end=%d&step=%d"
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

var prometheusClient *httputil.Client

func getPrometheusClient() *httputil.Client {
	return prometheusClient
}

func NewPrometheusClient(conf *config.DHCPConfig) error {
	client, err := httputil.NewHttpsClientSkipVerify(conf.Prometheus.CertPem, conf.Prometheus.KeyPem)
	if err != nil {
		return err
	}

	prometheusClient = client.SetBaseAuth(conf.Prometheus.Username, conf.Prometheus.Password)
	return nil
}

func prometheusRequest(ctx *MetricContext) (*PrometheusResponse, error) {
	var resp PrometheusResponse
	if err := getPrometheusClient().Get(genPrometheusUrl(ctx), &resp); err != nil {
		return nil, errorno.ErrNetworkError(errorno.ErrNameMetric, err.Error())
	}

	if resp.Status != "success" {
		return nil, errorno.ErrNetworkError(errorno.ErrNameMetric,
			fmt.Sprintf("get node %s %s failed with status: %s",
				ctx.NodeIP, ctx.MetricName, resp.Status))
	}

	return &resp, nil
}

func genPrometheusUrl(ctx *MetricContext) string {
	switch ctx.PromQuery {
	case PromQueryVersion:
		return fmt.Sprintf(string(ctx.PromQuery), HttpScheme, ctx.PrometheusAddr, ctx.MetricName,
			ctx.Version, ctx.Period.Begin, ctx.Period.End, ctx.Period.Step)
	case PromQueryNode:
		return fmt.Sprintf(string(ctx.PromQuery), HttpScheme, ctx.PrometheusAddr, ctx.MetricName,
			ctx.NodeIP, ctx.Period.Begin, ctx.Period.End, ctx.Period.Step)
	case PromQueryVersionNode:
		return fmt.Sprintf(string(ctx.PromQuery), HttpScheme, ctx.PrometheusAddr, ctx.MetricName,
			ctx.Version, ctx.NodeIP, ctx.Period.Begin, ctx.Period.End, ctx.Period.Step)
	default:
		return fmt.Sprintf(string(ctx.PromQuery), HttpScheme, ctx.PrometheusAddr, ctx.MetricName,
			ctx.Period.Begin, ctx.Period.End, ctx.Period.Step)
	}
}
