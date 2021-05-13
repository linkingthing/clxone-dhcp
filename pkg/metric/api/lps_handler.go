package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zdnscloud/cement/log"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/services"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/pb/alarm"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
	"github.com/linkingthing/clxone-dhcp/pkg/util/httpclient"
)

const (
	PromQueryUrl = "http://%s/api/v1/query_range?query=%s{node='%s'}&start=%d&end=%d&step=%d"
)

const (
	MetricLabelNode     = "node"
	MetricLabelType     = "type"
	MetricLabelVersion  = "version"
	MetricLabelView     = "view"
	MetricLabelRcode    = "rcode"
	MetricLabelSubnetId = "subnet_id"

	MetricNameDNSQPS                 = "lx_dns_qps"
	MetricNameDNSQueriesTotal        = "lx_dns_queries_total"
	MetricNameDNSQueryTypeRatios     = "lx_dns_query_type_ratios"
	MetricNameDNSCacheHits           = "lx_dns_cache_hits"
	MetricNameDNSCacheHitsRatioTotal = "lx_dns_cache_hits_ratio_total"
	MetricNameDNSCacheHitsRatio      = "lx_dns_cache_hits_ratio"
	MetricNameDNSResolvedRatios      = "lx_dns_resolved_ratios"

	MetricNameDHCPLPS          = "lx_dhcp_lps"
	MetricNameDHCPPacketsStats = "lx_dhcp_packets_stats"
	MetricNameDHCPLeasesTotal  = "lx_dhcp_leases_total"
	MetricNameDHCPUsages       = "lx_dhcp_usages"
)

type LPSHandler struct {
	prometheusAddr string
	exportPort     int
	LocalIP        string
}

func NewLPSHandler(conf *config.DHCPConfig) *LPSHandler {
	alarmService := services.NewAlarmService()
	err := alarmService.RegisterThresholdToKafka(services.IllegalDhcpAlarm, alarmService.DhcpThreshold)
	if err != nil {
		logrus.Error(err)
	}

	go alarmService.HandleUpdateThresholdEvent(services.UpdateThreshold, alarmService.UpdateLpsThresHold)

	h := &LPSHandler{
		prometheusAddr: conf.Prometheus.Addr,
		exportPort:     conf.Prometheus.ExportPort,
		LocalIP:        conf.Server.IP,
	}
	go h.monitor(alarmService.LpsThreshold)
	return h
}

func (h *LPSHandler) monitor(threshold *alarm.RegisterThreshold) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			nodes, err := services.NewDHCPService().GetNodeList()
			if err != nil {
				logrus.Error(err)
				continue
			}
			h.collectNodesMetric(nodes, threshold)
		}
	}
}

func (h *LPSHandler) collectNodesMetric(nodes []*resource.Node, threshold *alarm.RegisterThreshold) (err error) {
	if len(nodes) == 0 {
		return nil
	}

	period := genMonitorTimestamp(300)
	for _, node := range nodes {

		err = h.collectLPSMetric(node.Ip, threshold, period)

		if err != nil {
			log.Warnf("get node %s metrics failed: %s", node.Ip, err.Error())
		}

	}

	return nil
}

func (h *LPSHandler) collectLPSMetric(nodeIP string,
	threshold *alarm.RegisterThreshold,
	period *TimePeriodParams) error {

	dhcp, err := getLps(&MetricContext{PrometheusAddr: h.prometheusAddr, NodeIP: nodeIP, Period: period})
	if err != nil {
		return fmt.Errorf("get node %s lps failed: %s", nodeIP, err.Error())
	}

	alarmService := services.NewAlarmService()

	var exceedThresholdCount int
	var latestTime time.Time
	var latestValue uint64
	for _, value := range dhcp.Lps.Values {
		if value.Value >= threshold.Value {
			latestTime = time.Time(value.Timestamp)
			latestValue = value.Value
			exceedThresholdCount += 1
		}
	}

	if float64(exceedThresholdCount)/float64(len(dhcp.Lps.Values)) > 0.6 {
		alarmService.SendEventWithValues(&alarm.IllegalLPSAlarm{
			BaseAlarm: &alarm.BaseAlarm{
				BaseThreshold: alarmService.DhcpThreshold.BaseThreshold,
				Time:          latestTime.Format(time.RFC3339),
				SendMail:      alarmService.DhcpThreshold.SendMail,
			},
			LatestValue: latestValue,
		})
	}
	return nil
}

func genMonitorTimestamp(period int64) *TimePeriodParams {
	now := time.Now().Unix()
	return &TimePeriodParams{
		Begin: now - period,
		End:   now,
		Step:  period / 30,
	}
}

var TableHeaderLPS = []string{"日期", "LPS"}

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

func getLps(ctx *MetricContext) (*resource.Dhcp, *resterror.APIError) {
	ctx.MetricName = MetricNameDHCPLPS
	lpsValues, err := getValuesFromPrometheus(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get lps with node %s failed: %s", ctx.NodeIP, err.Error()))
	}

	dhcp := &resource.Dhcp{Lps: resource.Lps{Values: lpsValues}}
	dhcp.SetID(resource.ResourceIDLPS)
	return dhcp, nil
}

func getValuesFromPrometheus(ctx *MetricContext) ([]resource.ValueWithTimestamp, error) {
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[MetricLabelNode]; ok && nodeIp == ctx.NodeIP {
			return getValuesWithTimestamp(r.Values, ctx.Period), nil
		}
	}

	return nil, nil
}

func prometheusRequest(ctx *MetricContext) (*PrometheusResponse, error) {
	var resp PrometheusResponse
	if err := httpclient.GetHttpClient().Get(genPrometheusUrl(ctx), &resp); err != nil {
		return nil, err
	}

	if resp.Status != "success" {
		return nil, fmt.Errorf("get node %s %s failed with status: %s",
			ctx.NodeIP, ctx.MetricName, resp.Status)
	}

	return &resp, nil
}

func genPrometheusUrl(ctx *MetricContext) string {
	return fmt.Sprintf(PromQueryUrl, ctx.PrometheusAddr, ctx.MetricName, ctx.NodeIP,
		ctx.Period.Begin, ctx.Period.End, ctx.Period.Step)
}

func getValuesWithTimestamp(values [][]interface{}, period *TimePeriodParams) []resource.ValueWithTimestamp {
	var valueWithTimestamps []resource.ValueWithTimestamp
	for i := period.Begin; i <= period.End; i += period.Step {
		valueWithTimestamps = append(valueWithTimestamps, resource.ValueWithTimestamp{
			Timestamp: restresource.ISOTime(time.Unix(i, 0)),
			Value:     0,
		})
	}

	for _, vs := range values {
		if t, s := getTimestampAndValue(vs); t != 0 && t >= period.Begin {
			if value, err := strconv.ParseUint(s, 10, 64); err == nil {
				valueWithTimestamps[(t-period.Begin)/period.Step].Value = value
			}
		}
	}

	return valueWithTimestamps
}

func exportLps(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = MetricNameDHCPLPS
	ctx.TableHeader = TableHeaderLPS
	return exportTwoColumns(ctx)
}

func exportTwoColumns(ctx *MetricContext) (interface{}, *resterror.APIError) {
	resp, err := prometheusRequest(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("get node %s %s from prometheus failed: %s", ctx.NodeIP, ctx.MetricName, err.Error()))
	}

	var result PrometheusDataResult
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[MetricLabelNode]; ok && nodeIp == ctx.NodeIP {
			result = r
			break
		}
	}

	return exportTwoColumnsWithResult(ctx, result)
}

func exportTwoColumnsWithResult(ctx *MetricContext, result PrometheusDataResult) (interface{}, *resterror.APIError) {
	strMatrix := genStrMatrix(result.Values, ctx.Period)
	filepath, err := exportFile(ctx, strMatrix)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("export node %s %s failed: %s",
			ctx.NodeIP, ctx.MetricName, err.Error()))
	}

	return &resource.FileInfo{Path: filepath}, nil
}

func genStrMatrix(values [][]interface{}, period *TimePeriodParams) [][]string {
	var strMatrix [][]string
	for i := period.Begin; i <= period.End; i += period.Step {
		strMatrix = append(strMatrix, append([]string{time.Unix(int64(i), 0).Format(util.TimeFormat)}, "0"))
	}

	for _, vs := range values {
		if timestamp, value := getTimestampAndValue(vs); timestamp != 0 && timestamp >= period.Begin {
			strMatrix[(timestamp-period.Begin)/period.Step][1] = value
		}
	}

	return strMatrix
}
