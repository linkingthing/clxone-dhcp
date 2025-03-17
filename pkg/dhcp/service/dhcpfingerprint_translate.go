package service

import (
	"bytes"
	"time"

	"github.com/linkingthing/cement/uuid"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

const (
	DhcpFingerprintTemplateFileName     = "dhcp-fingerprint-template"
	DhcpFingerprintFileNamePrefix       = "dhcp-fingerprint-"
	DhcpFingerprintImportFileNamePrefix = "dhcp-fingerprint-import"

	FieldNameFingerprint = "指纹编码*"
	FieldNameVendorId    = "厂商标识"
	FieldNameClientType  = "客户端类型"
	FieldNameDataSource  = "数据来源"
)

var (
	TableHeaderDhcpFingerprint = []string{FieldNameFingerprint, FieldNameVendorId,
		FieldNameOperatingSystem, FieldNameClientType}
	TableHeaderDhcpFingerprintFail      = append(TableHeaderDhcpFingerprint, FailReasonLocalization)
	TableHeaderDhcpFingerprintFailLen   = len(TableHeaderDhcpFingerprintFail)
	DhcpFingerprintMandatoryFields      = []string{FieldNameFingerprint}
	TableHeaderDhcpFingerprintForExport = []string{FieldNameFingerprint, FieldNameVendorId,
		FieldNameOperatingSystem, FieldNameClientType, FieldNameDataSource}

	TemplateDhcpFingerprint = [][]string{
		[]string{"1,3,6,15,31,33", "MSFT5.0", "Windows10", "Windows"},
	}
)

func localizationDhcpFingerprintToStrSlice(fingerprint *resource.DhcpFingerprint) []string {
	return []string{
		fingerprint.Fingerprint, fingerprint.VendorId,
		fingerprint.OperatingSystem, fingerprint.ClientType,
	}
}

func localizationDhcpFingerprintForExport(fingerprint *resource.DhcpFingerprint) []string {
	return []string{
		fingerprint.Fingerprint, fingerprint.VendorId,
		fingerprint.OperatingSystem, fingerprint.ClientType,
		localizationDataSource(fingerprint.DataSource),
	}
}

func localizationDataSource(source resource.DataSource) string {
	switch source {
	case resource.DataSourceManual:
		return "手动添加"
	case resource.DataSourceAuto:
		return "自动采集"
	default:
		return "系统预置"

	}
}

func dhcpFingerprintToInsertDBSqlString(fingerprint *resource.DhcpFingerprint) string {
	id, _ := uuid.Gen()
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(id)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(fingerprint.Fingerprint)
	buf.WriteString("','")
	buf.WriteString(fingerprint.VendorId)
	buf.WriteString("','")
	buf.WriteString(fingerprint.OperatingSystem)
	buf.WriteString("','")
	buf.WriteString(fingerprint.ClientType)
	buf.WriteString("','")
	buf.WriteString(string(fingerprint.MatchPattern))
	buf.WriteString("','")
	buf.WriteString(string(fingerprint.DataSource))
	buf.WriteString("'),")
	return buf.String()
}
