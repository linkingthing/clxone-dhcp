package util

import (
	"bytes"
	"strings"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

const (
	FilterNameName       = "name"
	FilterNameComment    = "comment"
	FilterNameIp         = "ip"
	FilterNameCreateTime = "create_time"
	FilterNameVersion    = "version"
	FileNameSubnet       = "subnet"
)

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
	if filter.Modifier == restresource.Eq && len(filter.Values) == 1 {
		return filter.Values[0], true
	}

	return "", false
}

func GenSqlFromFilters(filters []restresource.Filter, table restdb.ResourceType, columnNames, arrayColumnNames []string, orderbyNames ...string) string {
	var buf bytes.Buffer
	buf.WriteString("SELECT * FROM gr_")
	buf.WriteString(string(table))
	buf.WriteString(" WHERE ")
	for _, filter := range filters {
		found := false
		for _, columnName := range columnNames {
			if value, validValue, foundColumn := getValueFromFilter(columnName, filter); foundColumn {
				if validValue {
					buf.WriteString(" ")
					buf.WriteString(columnName)
					buf.WriteString(" = '")
					buf.WriteString(value)
					buf.WriteString("' and ")
				}
				found = true
				break
			}
		}

		if found == false {
			for _, arrayColumnName := range arrayColumnNames {
				if value, validValue, foundColumn := getValueFromFilter(arrayColumnName, filter); foundColumn {
					if validValue {
						buf.WriteString(" '")
						buf.WriteString(value)
						buf.WriteString("' = any(")
						buf.WriteString(arrayColumnName)
						buf.WriteString(") and ")
					}
					break
				}
			}
		}
	}

	sql := strings.TrimSuffix(buf.String(), "and ")
	sql = strings.TrimSuffix(sql, "WHERE ")
	buf.Reset()
	for i, orderbyName := range orderbyNames {
		if i == 0 {
			buf.WriteString(" order by ")
		}
		buf.WriteString(orderbyName)
		buf.WriteString(",")
	}

	return sql + strings.TrimSuffix(buf.String(), ",")
}

func getValueFromFilter(filterName string, filter restresource.Filter) (string, bool, bool) {
	if filter.Name == filterName && filter.Modifier == restresource.Eq {
		if len(filter.Values) == 1 && len(filter.Values[0]) != 0 {
			return filter.Values[0], true, true
		} else {
			return "", false, true
		}
	}

	return "", false, false
}
