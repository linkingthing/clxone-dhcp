package util

import (
	"strings"

	"github.com/linkingthing/clxone-utils/filter"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

const (
	FilterNameName            = "name"
	FilterNameComment         = "comment"
	FilterNameIp              = "ip"
	FilterNameReservationType = "reservation_type"
	FilterNameCreateTime      = "create_time"
	FilterNameVersion         = "version"
	FilterNameSubnet          = "subnet"
	FilterNameTimeFrom        = "from"
	FilterNameTimeTo          = "to"

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
			if filterName == orderby {
				delete(conditions, "orderby")
			}
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

func SetIgnoreAuditLog(ctx *restresource.Context) {
	ctx.Response.Header().Add(filter.IgnoreAuditLog, filter.IgnoreAuditLog)
}

func StringOr(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

func FormatDbInsertError(errName errorno.ErrName, target string, err error) error {
	if err == nil {
		return nil
	}

	errMsg := pg.Error(err).Error()
	if strings.Contains(errMsg, "unique violation") ||
		strings.Contains(errMsg, "unique constraint") {
		return errorno.ErrDuplicate(errName, target)
	}

	return errorno.ErrDBError(errorno.ErrDBNameInsert, target, errMsg)
}
