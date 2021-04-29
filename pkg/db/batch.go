package db

import (
	"strconv"
	"strings"

	restdb "github.com/zdnscloud/gorest/db"
)

const BatchLimit = 100000

func BatchDelete(tx restdb.Transaction, table restdb.ResourceType, column string, parameters ...string) (int64, error) {
	if len(parameters) == 0 {
		return 0, nil
	}
	var buffer strings.Builder
	var sqlParameters []interface{}
	buffer.WriteString("DELETE FROM gr_")
	buffer.WriteString(string(table))
	buffer.WriteString(" WHERE ")
	buffer.WriteString(column)
	buffer.WriteString(" IN (")
	for i, param := range parameters {
		buffer.WriteString("$")
		buffer.WriteString(strconv.Itoa(i + 1))
		buffer.WriteString(",")
		sqlParameters = append(sqlParameters, param)
	}
	sql := strings.TrimSuffix(buffer.String(), ",") + ")"
	if count, err := tx.Exec(sql, sqlParameters...); err != nil {
		return count, err
	} else {
		return count, nil
	}
}

func BatchDeleteByUsing(table restdb.ResourceType, column string, parameters []string, tx restdb.Transaction) (int64, error) {
	if len(parameters) == 0 {
		return 0, nil
	}

	if len(parameters) > BatchLimit {
		var subParameters [][]string
		offset := len(parameters) / BatchLimit
		tail := len(parameters) % BatchLimit
		for i := 0; i < offset; i++ {
			subParameters = append(subParameters, parameters[i*BatchLimit:(i+1)*BatchLimit])
		}

		if tail > 0 {
			subParameters = append(subParameters, parameters[offset*BatchLimit:])
		}

		for _, subParameter := range subParameters {
			if _, err := executeBatchDelete(table, column, subParameter, tx); err != nil {
				return 0, err
			}
		}
	}

	return executeBatchDelete(table, column, parameters, tx)
}

func executeBatchDelete(table restdb.ResourceType, column string, parameters []string, tx restdb.Transaction) (int64, error) {
	var buffer strings.Builder
	buffer.WriteString("DELETE FROM gr_")
	buffer.WriteString(string(table))
	buffer.WriteString(" USING (VALUES('")
	buffer.WriteString(strings.Join(parameters, "'),('"))
	buffer.WriteString("'))")
	buffer.WriteString(" AS temp(")
	buffer.WriteString(column)
	buffer.WriteString(") WHERE ")
	buffer.WriteString("gr_")
	buffer.WriteString(string(table))
	buffer.WriteString(".")
	buffer.WriteString(column)
	buffer.WriteString(" = temp")
	buffer.WriteString(".")
	buffer.WriteString(column)

	if count, err := tx.Exec(buffer.String()); err != nil {
		return count, err
	} else {
		return count, nil
	}
}

func BatchUpdate(table restdb.ResourceType, column, value, condition string, conditions []interface{}, tx restdb.Transaction) (int64, error) {
	var buffer strings.Builder
	buffer.WriteString("UPDATE gr_")
	buffer.WriteString(string(table))
	buffer.WriteString(" SET ")
	buffer.WriteString(column)
	buffer.WriteString(" = '")
	buffer.WriteString(value)
	buffer.WriteString("' WHERE ")
	buffer.WriteString(condition)
	buffer.WriteString(" IN (")
	for i := range conditions {
		buffer.WriteString("$")
		buffer.WriteString(strconv.Itoa(i + 1))
		buffer.WriteString(",")
	}
	sql := strings.TrimSuffix(buffer.String(), ",") + ")"
	if count, err := tx.Exec(sql, conditions...); err != nil {
		return count, err
	} else {
		return count, nil
	}
}
