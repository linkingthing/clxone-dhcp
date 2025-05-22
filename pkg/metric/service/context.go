package service

import (
	"time"

	"github.com/linkingthing/clxone-utils/excel"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const DefaultStep = int64(24 * 12)

type MetricLabel string

const (
	MetricLabelNode    MetricLabel = "node"
	MetricLabelType    MetricLabel = "type"
	MetricLabelVersion MetricLabel = "version"
	MetricLabelSubnet  MetricLabel = "subnet"
)

type MetricName string

const (
	MetricNameDHCPLPS             MetricName = "lx_dhcp_lps"
	MetricNameDHCPPacketStats     MetricName = "lx_dhcp_packet_stats"
	MetricNameDHCPLeaseCount      MetricName = "lx_dhcp_lease_count"
	MetricNameDHCPLeaseCountTotal MetricName = "lx_dhcp_lease_count_total"
	MetricNameDHCPSubnetUsage     MetricName = "lx_dhcp_subnet_usage"
)

type DHCPVersion string

const (
	DHCPVersion4    DHCPVersion = "4"
	DHCPVersion6    DHCPVersion = "6"
	DHCPVersionNone DHCPVersion = ""
)

type MetricContext struct {
	PromQuery      PromQuery
	PrometheusAddr string
	NodeIP         string
	Hostname       string
	MetricName     MetricName
	MetricLabel    MetricLabel
	Version        DHCPVersion
	TableHeader    []string
	Period         *TimePeriod
}

type TimePeriod struct {
	Begin int64
	End   int64
	Step  int64
}

func getTimePeriodFromFilter(filters []restresource.Filter) (*TimePeriod, error) {
	timeFrom, _ := util.GetFilterValueWithEqModifierFromFilters(util.FilterNameTimeFrom, filters)
	timeTo, _ := util.GetFilterValueWithEqModifierFromFilters(util.FilterNameTimeTo, filters)
	return parseTimePeriod(timeFrom, timeTo)
}

func parseTimePeriod(from, to string) (*TimePeriod, error) {
	timeTo := time.Now()
	timeFrom := timeTo.AddDate(0, 0, -1)
	if !excel.IsSpaceField(from) {
		timeFrom_, err := time.Parse(util.TimeFormatYMD, from)
		if err != nil {
			return nil, errorno.ErrInvalidParams(errorno.ErrNameTime, from)
		}

		timeFrom = timeFrom_
	}

	if !excel.IsSpaceField(to) {
		timeTo_, err := time.Parse(util.TimeFormatYMD, to)
		if err != nil {
			return nil, errorno.ErrInvalidParams(errorno.ErrNameTime, to)
		}

		timeTo = timeTo_
	}

	return genTimePeriod(timeFrom, timeTo)
}

func genTimePeriod(from, to time.Time) (*TimePeriod, error) {
	if to.Before(from) {
		return nil, errorno.ErrLessThan(errorno.ErrNameTime,
			to.Format(excel.TimeFormat), from.Format(excel.TimeFormat))
	} else if from.Equal(to) {
		from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.Local)
		to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.Local)
	}

	return &TimePeriod{
		Begin: from.Unix(),
		End:   to.Unix(),
		Step:  DefaultStep,
	}, nil
}

func getDHCPVersionFromDHCPID(id string) (DHCPVersion, error) {
	switch DHCPVersion(id) {
	case DHCPVersion4:
		return DHCPVersion4, nil
	case DHCPVersion6:
		return DHCPVersion6, nil
	default:
		return DHCPVersionNone, errorno.ErrInvalidParams(errorno.ErrNameVersion, id)
	}
}

func IsDHCPVersion4(version string) bool {
	return DHCPVersion(version) == DHCPVersion4
}

func IsDHCPVersion6(version string) bool {
	return DHCPVersion(version) == DHCPVersion6
}
