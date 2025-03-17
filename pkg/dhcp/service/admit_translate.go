package service

import (
	"bytes"
	"time"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

const (
	AdmitDuidTemplateFileName     = "admit-duid-template"
	AdmitDuidFileNamePrefix       = "admit-duid-"
	AdmitDuidImportFileNamePrefix = "admit-duid-import"

	AdmitMacTemplateFileName     = "admit-mac-template"
	AdmitMacFileNamePrefix       = "admit-mac-"
	AdmitMacImportFileNamePrefix = "admit-mac-import"

	AdmitFingerprintTemplateFileName     = "admit-fingerprint-template"
	AdmitFingerprintFileNamePrefix       = "admit-fingerprint-"
	AdmitFingerprintImportFileNamePrefix = "admit-fingerprint-import"

	FieldNameAdmitDuid       = "DUID*"
	FieldNameIsAdmitted      = "是否允许*"
	FieldNameAdmitMac        = "MAC地址*"
	FieldNameAdmitClientType = "客户端类型*"
)

var (
	TableHeaderAdmitDuid        = []string{FieldNameAdmitDuid, FieldNameIsAdmitted, FieldNameComment}
	TableHeaderAdmitDuidFail    = append(TableHeaderAdmitDuid, FailReasonLocalization)
	TableHeaderAdmitDuidFailLen = len(TableHeaderAdmitDuidFail)
	AdmitDuidMandatoryFields    = []string{FieldNameAdmitDuid, FieldNameIsAdmitted}

	TableHeaderAdmitMac        = []string{FieldNameAdmitMac, FieldNameIsAdmitted, FieldNameComment}
	TableHeaderAdmitMacFail    = append(TableHeaderAdmitMac, FailReasonLocalization)
	TableHeaderAdmitMacFailLen = len(TableHeaderAdmitMacFail)
	AdmitMacMandatoryFields    = []string{FieldNameAdmitMac, FieldNameIsAdmitted}

	TableHeaderAdmitFingerprint        = []string{FieldNameAdmitClientType, FieldNameIsAdmitted, FieldNameComment}
	TableHeaderAdmitFingerprintFail    = append(TableHeaderAdmitFingerprint, FailReasonLocalization)
	TableHeaderAdmitFingerprintFailLen = len(TableHeaderAdmitFingerprintFail)
	AdmitFingerprintMandatoryFields    = []string{FieldNameAdmitClientType, FieldNameIsAdmitted}

	TemplateAdmitDuid        = [][]string{[]string{"0102", "是", "备注1"}}
	TemplateAdmitMac         = [][]string{[]string{"01:02:03:04:05:06", "是", "备注1"}}
	TemplateAdmitFingerprint = [][]string{[]string{"Windows", "是", "备注1"}}
)

func localizationAdmitDuidToStrSlice(duid *resource.AdmitDuid) []string {
	return []string{duid.Duid, localizationBool(duid.IsAdmitted), duid.Comment}
}

func localizationAdmitMacToStrSlice(mac *resource.AdmitMac) []string {
	return []string{mac.HwAddress, localizationBool(mac.IsAdmitted), mac.Comment}
}

func localizationAdmitFingerprintToStrSlice(fingerprint *resource.AdmitFingerprint) []string {
	return []string{fingerprint.ClientType, localizationBool(fingerprint.IsAdmitted), fingerprint.Comment}
}

func localizationBool(b bool) string {
	if b {
		return "是"
	} else {
		return "否"
	}
}

func internationalizationBool(b string) bool {
	if b == "是" {
		return true
	} else {
		return false
	}
}

func admitDuidToInsertDBSqlString(duid *resource.AdmitDuid) string {
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(duid.Duid)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(duid.Duid)
	buf.WriteString("','")
	buf.WriteString(boolToString(duid.IsAdmitted))
	buf.WriteString("','")
	buf.WriteString(duid.Comment)
	buf.WriteString("'),")
	return buf.String()
}

func admitMacToInsertDBSqlString(mac *resource.AdmitMac) string {
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(mac.HwAddress)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(mac.HwAddress)
	buf.WriteString("','")
	buf.WriteString(boolToString(mac.IsAdmitted))
	buf.WriteString("','")
	buf.WriteString(mac.Comment)
	buf.WriteString("'),")
	return buf.String()
}

func admitFingerprintToInsertDBSqlString(fingerprint *resource.AdmitFingerprint) string {
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(fingerprint.ClientType)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(fingerprint.ClientType)
	buf.WriteString("','")
	buf.WriteString(boolToString(fingerprint.IsAdmitted))
	buf.WriteString("','")
	buf.WriteString(fingerprint.Comment)
	buf.WriteString("'),")
	return buf.String()
}
