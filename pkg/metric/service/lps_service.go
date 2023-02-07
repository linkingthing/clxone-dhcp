package service

import (
	"fmt"
	"strconv"
	"time"

	"github.com/linkingthing/cement/log"
	pbutil "github.com/linkingthing/clxone-utils/alarm/proto"
	csvutil "github.com/linkingthing/clxone-utils/csv"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	service2 "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
)

type LPSService struct {
	prometheusAddr string
}

func NewLPSService(config *config.DHCPConfig) *LPSService {
	s := &LPSService{prometheusAddr: config.Prometheus.Addr}
	go s.monitor()
	return s
}

func (h *LPSService) monitor() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := h.collectLPS(); err != nil {
				log.Warnf("collect lps failed: %s", err.Error())
			}
		}
	}
}

func (h *LPSService) collectLPS() error {
	threshold := service2.GetAlarmService().GetThreshold(pbutil.ThresholdName_lps)
	if threshold == nil {
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
				if !ok {
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
	var latestValue uint64
	for _, nodeAndValues := range lpsValues {
		for nodeIp, values := range nodeAndValues {
			for _, value := range values {
				if value.Value >= threshold.Value {
					latestValue = value.Value
					exceedThresholdCount += 1
				}
			}

			if float64(exceedThresholdCount)/float64(len(values)) > 0.6 {
				return service2.GetAlarmService().AddLPSAlarm(nodeIp, latestValue)
			}
		}
	}

	return nil
}

func (h *LPSService) List(ctx *restresource.Context) (interface{}, error) {
	nodeIpAndValues, err := getNodeIpAndValuesFromPrometheus(ctx, &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLPS,
		PromQuery:      PromQueryVersion,
	})
	if err != nil {
		return nil, err
	}

	nodeNames, err := service.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID()))
	if err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	}

	lpses := make([]*resource.Lps, 0, len(nodeIpAndValues))
	for nodeIp, values := range nodeIpAndValues {
		lps := &resource.Lps{Values: values, NodeName: nodeNames[nodeIp]}
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

func (h *LPSService) Get(ctx *restresource.Context) (restresource.Resource, error) {
	lps := ctx.Resource.(*resource.Lps)
	values, err := getValuesFromPrometheus(ctx, &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPLPS,
		PromQuery:      PromQueryVersionNode,
		NodeIP:         lps.GetID(),
	})
	if err != nil {
		return nil, err
	}

	if nodeNames, err := service.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID())); err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	} else {
		lps.NodeName = nodeNames[lps.GetID()]
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

var TableHeaderLPS = []string{"日期", "LPS"}

func (h *LPSService) Export(ctx *restresource.Context) (interface{}, error) {
	if result, err := exportTwoColumns(ctx, &MetricContext{
		NodeIP:         ctx.Resource.GetID(),
		PrometheusAddr: h.prometheusAddr,
		PromQuery:      PromQueryVersionNode,
		MetricName:     MetricNameDHCPLPS,
		TableHeader:    TableHeaderLPS,
	}); err != nil {
		return nil, fmt.Errorf("lps %s export action failed: %s", ctx.Resource.GetID(), err.Error())
	} else {
		return result, nil
	}
}

func exportTwoColumns(ctx *restresource.Context, metricCtx *MetricContext) (interface{}, error) {
	filter, ok := ctx.Resource.GetAction().Input.(*resource.ExportFilter)
	if !ok {
		return nil, errorno.ErrInvalidFormat(errorno.ErrNameLPS, errorno.ErrNameExport)
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
		return nil, err
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
		return nil, err
	}

	return &resource.FileInfo{Path: filepath}, nil
}

func genTwoStrMatrix(values [][]interface{}, ctx *MetricContext) [][]string {
	var strMatrix [][]string
	for i := ctx.Period.Begin; i <= ctx.Period.End; i += ctx.Period.Step {
		strMatrix = append(strMatrix,
			append([]string{time.Unix(i, 0).Format(csvutil.TimeFormat)}, "0"))
	}

	for _, vs := range values {
		if timestamp, value := getTimestampAndValue(vs); timestamp != 0 &&
			timestamp >= ctx.Period.Begin && value != "0" {
			strMatrix[(timestamp-ctx.Period.Begin)/ctx.Period.Step][1] = value
		}
	}

	return strMatrix
}
