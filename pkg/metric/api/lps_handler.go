package api

import (
	"fmt"
	"strconv"
	"time"

	"github.com/zdnscloud/cement/log"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/proto/alarm"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type LPSHandler struct {
	prometheusAddr string
}

func NewLPSHandler(conf *config.DHCPConfig) (*LPSHandler, error) {
	alarmService := service.NewAlarmService()
	err := alarmService.RegisterThresholdToKafka(service.RegisterThreshold, alarmService.LpsThreshold)
	if err != nil {
		return nil, err
	}

	go alarmService.HandleUpdateThresholdEvent(service.ThresholdDhcpTopic, alarmService.UpdateLpsThresHold)

	h := &LPSHandler{prometheusAddr: conf.Prometheus.Addr}
	go h.monitor(alarmService.LpsThreshold)

	return h, nil
}

func (h *LPSHandler) monitor(threshold *alarm.RegisterThreshold) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if alarmService := service.NewAlarmService(); alarmService.LpsThreshold.Enabled {
				if err := h.collectLPS(threshold); err != nil {
					log.Warnf("collect lps failed: %s", err.Error())
				}
			}
		}
	}
}

func (h *LPSHandler) collectLPS(threshold *alarm.RegisterThreshold) error {
	alarmService := service.NewAlarmService()
	if alarmService.LpsThreshold.Enabled == false {
		return nil
	}

	now := time.Now().Unix()
	ctx := &MetricContext{
		PromQuery:      PromQueryName,
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLPS,
		Period: &TimePeriod{
			Begin: now - 300,
			End:   now,
			Step:  10,
		},
	}

	resp, err := prometheusRequest(ctx)
	if err != nil {
		return err
	}

	lpsValues := make(map[string]map[string][]resource.ValueWithTimestamp)
	for _, r := range resp.Data.Results {
		if version, ok := r.MetricLabels[string(MetricLabelVersion)]; ok {
			if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok {
				nodeAndValues, ok := lpsValues[version]
				if ok == false {
					nodeAndValues = make(map[string][]resource.ValueWithTimestamp)
				}
				nodeAndValues[nodeIp] = getValuesWithTimestamp(r.Values, ctx.Period)
				lpsValues[version] = nodeAndValues
			}
		}
	}

	if len(lpsValues) == 0 {
		return nil
	}

	var exceedThresholdCount int
	var latestTime time.Time
	var latestValue uint64

	for _, nodeAndValues := range lpsValues {
		for nodeIp, values := range nodeAndValues {
			for _, value := range values {
				if value.Value >= threshold.Value {
					latestTime = time.Time(value.Timestamp)
					latestValue = value.Value
					exceedThresholdCount += 1
				}
			}

			if float64(exceedThresholdCount)/float64(len(values)) > 0.6 {
				alarmService.SendEventWithValues(service.AlarmKeyLps, &alarm.LpsAlarm{
					BaseAlarm: &alarm.BaseAlarm{
						BaseThreshold: alarmService.DhcpThreshold.BaseThreshold,
						Time:          latestTime.Format(time.RFC3339),
						SendMail:      alarmService.DhcpThreshold.SendMail,
						Threshold:     alarmService.LpsThreshold.Value,
					},
					NodeIp:      nodeIp,
					LatestValue: latestValue,
				})
			}
		}
	}

	return nil
}

func (h *LPSHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	nodeIpAndValues, err := getNodeIpAndValuesFromPrometheus(ctx, &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLPS,
		PromQuery:      PromQueryVersion,
	})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			"get lpses from prometheus failed: "+err.Error())
	}

	var lpses []*resource.Lps
	for nodeIp, values := range nodeIpAndValues {
		lps := &resource.Lps{Values: values}
		lps.SetID(nodeIp)
		lpses = append(lpses, lps)
	}

	return lpses, nil
}

func getNodeIpAndValuesFromPrometheus(ctx *restresource.Context, metricCtx *MetricContext) (map[string][]resource.ValueWithTimestamp, error) {
	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, err
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, err
	}

	nodeIpAndValues := make(map[string][]resource.ValueWithTimestamp)
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok {
			nodeIpAndValues[nodeIp] = getValuesWithTimestamp(r.Values, metricCtx.Period)
		}
	}

	return nodeIpAndValues, nil
}

func resetMetricContext(ctx *restresource.Context, metricCtx *MetricContext) (err error) {
	metricCtx.Period, err = getTimePeriodFromFilter(ctx.GetFilters())
	if err != nil {
		return
	}

	metricCtx.Version, err = getDHCPVersionFromDHCPID(ctx.Resource.GetParent().GetID())
	return
}

func getValuesWithTimestamp(values [][]interface{}, period *TimePeriod) []resource.ValueWithTimestamp {
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

func (h *LPSHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	lps := ctx.Resource.(*resource.Lps)
	values, err := getValuesFromPrometheus(ctx, &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLPS,
		PromQuery:      PromQueryVersionNode,
		NodeIP:         lps.GetID(),
	})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("get lps with node %s failed: %s", lps.GetID(), err.Error()))
	}

	lps.Values = values
	return lps, nil
}

func getValuesFromPrometheus(ctx *restresource.Context, metricCtx *MetricContext) ([]resource.ValueWithTimestamp, error) {
	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, err
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, err
	}

	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok && nodeIp == metricCtx.NodeIP {
			return getValuesWithTimestamp(r.Values, metricCtx.Period), nil
		}
	}

	return nil, nil
}

func (h *LPSHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.export(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

var TableHeaderLPS = []string{"日期", "LPS"}

func (h *LPSHandler) export(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := exportTwoColumns(ctx, &MetricContext{
		NodeIP:         ctx.Resource.GetID(),
		PrometheusAddr: h.prometheusAddr,
		PromQuery:      PromQueryVersionNode,
		MetricName:     MetricNameDHCPLPS,
		TableHeader:    TableHeaderLPS,
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("lps %s export action failed: %s", ctx.Resource.GetID(), err.Error()))
	} else {
		return result, nil
	}
}

func exportTwoColumns(ctx *restresource.Context, metricCtx *MetricContext) (interface{}, error) {
	filter, ok := ctx.Resource.GetAction().Input.(*resource.ExportFilter)
	if ok == false {
		return nil, fmt.Errorf("action input is not export filter")
	}

	timePeriod, err := parseTimePeriod(filter.From, filter.To)
	if err != nil {
		return nil, err
	}

	version, err := getDHCPVersionFromDHCPID(ctx.Resource.GetParent().GetID())
	if err != nil {
		return nil, err
	}

	metricCtx.Period = timePeriod
	metricCtx.Version = version
	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, fmt.Errorf("get node %s %s from prometheus failed: %s",
			metricCtx.NodeIP, metricCtx.MetricName, err.Error())
	}

	var result PrometheusDataResult
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok && nodeIp == metricCtx.NodeIP {
			result = r
			break
		}
	}

	return exportTwoColumnsWithResult(metricCtx, result)
}

func exportTwoColumnsWithResult(ctx *MetricContext, result PrometheusDataResult) (interface{}, error) {
	strMatrix := genTwoStrMatrix(result.Values, ctx)
	filepath, err := exportFile(ctx, strMatrix)
	if err != nil {
		return nil, fmt.Errorf("export node %s %s file failed: %s",
			ctx.NodeIP, ctx.MetricName, err.Error())
	}

	return &resource.FileInfo{Path: filepath}, nil
}

func genTwoStrMatrix(values [][]interface{}, ctx *MetricContext) [][]string {
	var strMatrix [][]string
	for i := ctx.Period.Begin; i <= ctx.Period.End; i += ctx.Period.Step {
		strMatrix = append(strMatrix,
			append([]string{time.Unix(int64(i), 0).Format(util.TimeFormat)}, "0"))
	}

	for _, vs := range values {
		if timestamp, value := getTimestampAndValue(vs); timestamp != 0 &&
			timestamp >= ctx.Period.Begin && value != "0" {
			strMatrix[(timestamp-ctx.Period.Begin)/ctx.Period.Step][1] = value
		}
	}

	return strMatrix
}
