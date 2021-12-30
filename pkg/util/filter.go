package util

import (
	restresource "github.com/linkingthing/gorest/resource"
	"strings"
)

const (
	FilterNameName       = "name"
	FilterNameComment    = "comment"
	FilterNameIp         = "ip"
	FilterNameCreateTime = "create_time"
	FilterNameVersion    = "version"
	FilterNameSubnet     = "subnet"
	FilterNameTimeFrom   = "from"
	FilterNameTimeTo     = "to"

	TimeFromSuffix  = " 00:00"
	TimeToSuffix    = " 23:59"
	TimeFormatYMD   = "2006-01-02"
	TimeFormatYMDHM = "2006-01-02 15:04"
)

func GenStrConditionsFromFilters(filters []restresource.Filter, orderby string, filterNames ...string) map[string]interface{} {
	conditions := make(map[string]interface{})
	if len(orderby) != 0 {
		conditions["orderby"] = orderby
	}

	if len(filters) == 0 {
		return conditions
	}

	for _, filterName := range filterNames {
		if value, ok := GetFilterValueWithEqModifierFromFilters(filterName, filters); ok {
			conditions[filterName] = value
		}
	}

	return conditions
}

func GetFilterValueWithEqModifierFromFilters(filterName string, filters []restresource.Filter) (string, bool) {
	for _, filter := range filters {
		if filter.Name == filterName && filter.Modifier == restresource.Eq {
			if len(filter.Values) == 1 && strings.TrimSpace(filter.Values[0]) != "" {
				return filter.Values[0], true
			}
			break
		}
	}

	return "", false
}

func GetFilterValueWithEqModifierFromFilter(filter restresource.Filter) (string, bool) {
	if filter.Modifier == restresource.Eq && len(filter.Values) == 1 &&
		strings.TrimSpace(filter.Values[0]) != "" {
		return filter.Values[0], true
	}

	return "", false
}
