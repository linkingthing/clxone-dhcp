package handler

import (
	"fmt"
	"time"

	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	DefaultStep = int64(24 * 12)
)

type MetricContext struct {
	NodeIP         string
	PrometheusAddr string
	MetricName     string
	MetricLabel    string
	TableHeader    []string
	PeriodBegin    string
	Period         *TimePeriodParams
	AggsName       string
	AggsKeyword    string
}

type TimePeriodParams struct {
	From  string
	To    string
	Begin int64
	End   int64
	Step  int64
}

type DhcpHandler struct {
	prometheusAddr string
}

func NewDhcpHandler(conf *config.DDIControllerConfig) *DhcpHandler {
	return &DhcpHandler{
		prometheusAddr: conf.Prometheus.Addr,
	}
}

func (h *DhcpHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	lps := &resource.Dhcp{}
	lps.SetID(resource.ResourceIDLPS)
	lease := &resource.Dhcp{}
	lease.SetID(resource.ResourceIDLease)
	packets := &resource.Dhcp{}
	packets.SetID(resource.ResourceIDPackets)
	subnetUsedRatios := &resource.Dhcp{}
	subnetUsedRatios.SetID(resource.ResourceIDSubnetUsedRatios)
	return []*resource.Dhcp{lps, lease, packets, subnetUsedRatios}, nil
}

func (h *DhcpHandler) genDHCPMetricContext(nodeIP string, period *TimePeriodParams) *MetricContext {
	return &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		NodeIP:         nodeIP,
		Period:         period,
	}
}

func getTimePeriodParamFromFilter(filters []restresource.Filter) (*TimePeriodParams, error) {
	to := time.Now()
	from := to.AddDate(0, 0, -1)

	if timeFrom, exists, err := getTimeFromFilter(util.FilterTimeFrom, filters); err != nil {
		return nil, err
	} else if exists {
		from = timeFrom
	}

	if timeTo, exists, err := getTimeFromFilter(util.FilterTimeTo, filters); err != nil {
		return nil, err
	} else if exists {
		to = timeTo
	}

	return genTimePeriod(from, to)
}

func genTimePeriod(from, to time.Time) (*TimePeriodParams, error) {
	if to.Before(from) {
		return nil, fmt.Errorf("time to %s before from %s",
			to.Format(util.TimeFormat), from.Format(util.TimeFormat))
	} else if from.Equal(to) {
		from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.Local)
		to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.Local)
	}

	return &TimePeriodParams{
		From:  from.Format(util.TimeFormatYMDHM),
		To:    to.Format(util.TimeFormatYMDHM),
		Begin: from.Unix(),
		End:   to.Unix(),
		Step:  DefaultStep,
	}, nil
}

func getTimeFromFilter(filterName string, filters []restresource.Filter) (time.Time, bool, error) {
	timeStr, ok := util.GetFilterValueWithEqModifierFromFilters(filterName, filters)
	if ok == false {
		return time.Time{}, false, nil
	}

	if t, err := time.Parse(util.TimeFormatYMD, timeStr); err != nil {
		return time.Time{}, true, err
	} else {
		return t, true, nil
	}
}

func (h *DhcpHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpID := ctx.Resource.GetID()
	period, err := getTimePeriodParamFromFilter(ctx.GetFilters())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("invalid time format: %s", err.Error()))
	}

	context := h.genDHCPMetricContext(ctx.Resource.GetParent().GetID(), period)
	switch dhcpID {
	case resource.ResourceIDLPS:
		return getLps(context)
	case resource.ResourceIDLease:
		return getLease(context)
	case resource.ResourceIDPackets:
		return getPackets(context)
	case resource.ResourceIDSubnetUsedRatios:
		return getSubnetUsedRatios(context)
	default:
		return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found dhcp resource %s", dhcpID))
	}
}

func (h *DhcpHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.export(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *DhcpHandler) export(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	period, err := getTimePeriodParamFromActionInput(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("parse action input failed: %s", err.Error()))
	}

	context := h.genDHCPMetricContext(ctx.Resource.GetParent().GetID(), period)
	dhcpID := ctx.Resource.GetID()
	switch dhcpID {
	case resource.ResourceIDLPS:
		return exportLps(context)
	case resource.ResourceIDLease:
		return exportLease(context)
	case resource.ResourceIDPackets:
		return exportPackets(context)
	case resource.ResourceIDSubnetUsedRatios:
		return exportSubnetUsedRatios(context)
	default:
		return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found dhcp resource %s", dhcpID))
	}
}

func getTimePeriodParamFromActionInput(ctx *restresource.Context) (*TimePeriodParams, error) {
	filter, ok := ctx.Resource.GetAction().Input.(*resource.ExportFilter)
	if ok == false {
		return nil, fmt.Errorf("action exportcsv input invalid")
	}

	to := time.Now()
	from := to.AddDate(0, 0, -1)
	if util.IsSpaceField(filter.From) == false {
		timeFrom, err := time.Parse(util.TimeFormatYMD, filter.From)
		if err != nil {
			return nil, err
		}

		from = timeFrom
	}

	if util.IsSpaceField(filter.To) == false {
		timeTo, err := time.Parse(util.TimeFormatYMD, filter.To)
		if err != nil {
			return nil, err
		}

		to = timeTo
	}

	return genTimePeriod(from, to)
}
