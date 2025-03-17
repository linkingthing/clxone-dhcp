package service

import (
	"bytes"
	"time"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

const (
	DhcpOuiTemplateFileName     = "dhcp-oui-template"
	DhcpOuiFileNamePrefix       = "dhcp-oui-"
	DhcpOuiImportFileNamePrefix = "dhcp-oui-import"

	FieldNameOui          = "OUI*"
	FieldNameOrganization = "厂商标识*"
)

var (
	TableHeaderDhcpOui          = []string{FieldNameOui, FieldNameOperatingSystem}
	TableHeaderDhcpOuiFail      = append(TableHeaderDhcpOui, FailReasonLocalization)
	TableHeaderDhcpOuiFailLen   = len(TableHeaderDhcpOuiFail)
	DhcpOuiMandatoryFields      = []string{FieldNameOui, FieldNameOrganization}
	TableHeaderDhcpOuiForExport = []string{FieldNameOui, FieldNameOperatingSystem, FieldNameDataSource}

	TemplateDhcpOui = [][]string{
		[]string{"01:02:03", "LINKINGTHING.COM"},
	}
)

func localizationDhcpOuiToStrSlice(oui *resource.DhcpOui) []string {
	return []string{oui.Oui, oui.Organization}
}

func localizationDhcpOuiForExport(oui *resource.DhcpOui) []string {
	return []string{oui.Oui, oui.Organization, localizationDataSource(oui.DataSource)}
}

func dhcpOuiToInsertDBSqlString(oui *resource.DhcpOui) string {
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(oui.Oui)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(oui.Oui)
	buf.WriteString("','")
	buf.WriteString(oui.Organization)
	buf.WriteString("','")
	buf.WriteString(string(oui.DataSource))
	buf.WriteString("'),")
	return buf.String()
}
