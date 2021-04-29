package util

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/zdnscloud/cement/slice"
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

const (
	FilterTimeFrom  = "from"
	FilterTimeTo    = "to"
	TimeFromSuffix  = " 00:00"
	TimeToSuffix    = " 23:59"
	TimeFormatYMD   = "2006-01-02"
	TimeFormatYMDHM = "2006-01-02 15:04"
)

func GenSqlAndArgsByFileters(table restdb.ResourceType, filterNames []string, from int, filters []restresource.Filter) (string, []interface{}) {
	now := time.Now()
	timeFrom := now.AddDate(0, 0, -from).Format(TimeFormatYMD)
	timeTo := now.Format(TimeFormatYMD)
	conditions := make(map[string]string)
	for _, filter := range filters {
		if slice.SliceIndex(filterNames, filter.Name) != -1 {
			if value, ok := GetFilterValueWithEqModifierFromFilter(filter); ok {
				conditions[filter.Name+" = $"] = value
			}
		} else {
			switch filter.Name {
			case FilterTimeFrom:
				if from, ok := GetFilterValueWithEqModifierFromFilter(filter); ok {
					timeFrom = from
				}
			case FilterTimeTo:
				if to, ok := GetFilterValueWithEqModifierFromFilter(filter); ok {
					timeTo = to
				}
			}
		}
	}

	var buf bytes.Buffer
	buf.WriteString("select * from gr_")
	buf.WriteString(string(table))
	buf.WriteString(" where ")
	var args []interface{}
	for cond, arg := range conditions {
		args = append(args, arg)
		buf.WriteString(cond)
		buf.WriteString(strconv.Itoa(len(args)))
		buf.WriteString(" and ")
	}

	buf.WriteString("create_time between ")
	buf.WriteString("'")
	buf.WriteString(timeFrom)
	buf.WriteString(TimeFromSuffix)
	buf.WriteString("'")
	buf.WriteString(" and ")
	buf.WriteString("'")
	buf.WriteString(timeTo)
	buf.WriteString(TimeToSuffix)
	buf.WriteString("'")
	buf.WriteString(" order by create_time desc")
	return buf.String(), args
}

func GenSqlQueryIn(table restdb.ResourceType, column string, conArray []string) string {
	var buf bytes.Buffer
	buf.WriteString("select * from gr_")
	buf.WriteString(string(table))
	buf.WriteString(" where ")
	buf.WriteString(column)
	buf.WriteString(" in ('")
	buf.WriteString(strings.Join(conArray, "','"))
	buf.WriteString("')")
	return buf.String()
}

func GenSqlCountQueryIn(table restdb.ResourceType, column string, conArray []string) string {
	var buf bytes.Buffer
	buf.WriteString("select count(*) from gr_")
	buf.WriteString(string(table))
	buf.WriteString(" where ")
	buf.WriteString(column)
	buf.WriteString(" in ('")
	buf.WriteString(strings.Join(conArray, "','"))
	buf.WriteString("')")
	return buf.String()
}

func GenSqlQueryArray(table restdb.ResourceType, column string, parameter string) string {
	var buf bytes.Buffer
	buf.WriteString("select * from gr_")
	buf.WriteString(string(table))
	buf.WriteString(" where '")
	buf.WriteString(parameter)
	buf.WriteString("'=any( ")
	buf.WriteString(column)
	buf.WriteString(" )")
	return buf.String()
}
