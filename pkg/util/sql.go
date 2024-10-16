package util

import (
	"bytes"
	"strings"

	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

func BatchUpdateById(tx restdb.Transaction, table string, columns []string, values map[string]string) error {
	return BatchUpdateByColumn(tx, table, restdb.IDField, columns, values)
}

func BatchUpdateByColumn(tx restdb.Transaction, table string, conditionColumn string, updateColumns []string,
	values map[string]string) error {

	if len(updateColumns) == 0 || len(values) == 0 {
		return nil
	}

	var buf bytes.Buffer
	buf.WriteString("UPDATE gr_")
	buf.WriteString(table)
	buf.WriteString(" SET ")
	for i, column := range updateColumns {
		buf.WriteString(column)
		buf.WriteString(" = tmp.")
		buf.WriteString(column)
		if i < len(updateColumns)-1 {
			buf.WriteString(",")
		}
	}
	buf.WriteString(" FROM (VALUES ")
	for _, value := range values {
		buf.WriteString(value)
		buf.WriteString(",")
	}
	buf.Truncate(buf.Len() - 1)

	updateColumns = append([]string{conditionColumn}, updateColumns...)
	buf.WriteString(")")
	buf.WriteString(" AS tmp (")
	buf.WriteString(strings.Join(updateColumns, ","))
	buf.WriteString(")")
	buf.WriteString(" WHERE gr_")
	buf.WriteString(table)
	buf.WriteString("." + conditionColumn + " = tmp." + conditionColumn)

	_, err := tx.Exec(buf.String())
	if err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameUpdate, table, pg.Error(err).Error())
	}
	return nil
}
