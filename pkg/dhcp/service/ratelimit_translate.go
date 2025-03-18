package service

import (
	"bytes"
	"time"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

const (
	RateLimitDuidTemplateFileName     = "rate-limit-duid-template"
	RateLimitDuidFileNamePrefix       = "rate-limit-duid-"
	RateLimitDuidImportFileNamePrefix = "rate-limit-duid-import"

	RateLimitMacTemplateFileName     = "rate-limit-mac-template"
	RateLimitMacFileNamePrefix       = "rate-limit-mac-"
	RateLimitMacImportFileNamePrefix = "rate-limit-mac-import"

	FieldNameRateLimitDuid = "DUID*"
	FieldNameRateLimit     = "限速指标"
	FieldNameRateLimitMac  = "MAC地址*"
)

var (
	TableHeaderRateLimitDuid        = []string{FieldNameRateLimitDuid, FieldNameRateLimit, FieldNameComment}
	TableHeaderRateLimitDuidFail    = append(TableHeaderRateLimitDuid, FailReasonLocalization)
	TableHeaderRateLimitDuidFailLen = len(TableHeaderRateLimitDuidFail)
	RateLimitDuidMandatoryFields    = []string{FieldNameRateLimitDuid}

	TableHeaderRateLimitMac        = []string{FieldNameRateLimitMac, FieldNameRateLimit, FieldNameComment}
	TableHeaderRateLimitMacFail    = append(TableHeaderRateLimitMac, FailReasonLocalization)
	TableHeaderRateLimitMacFailLen = len(TableHeaderRateLimitMacFail)
	RateLimitMacMandatoryFields    = []string{FieldNameRateLimitMac}

	TemplateRateLimitDuid = [][]string{[]string{"0102", "10", "备注1"}}
	TemplateRateLimitMac  = [][]string{[]string{"01:02:03:04:05:06", "20", "备注1"}}
)

func localizationRateLimitDuidToStrSlice(duid *resource.RateLimitDuid) []string {
	return []string{duid.Duid, uint32ToString(duid.RateLimit), duid.Comment}
}

func localizationRateLimitMacToStrSlice(mac *resource.RateLimitMac) []string {
	return []string{mac.HwAddress, uint32ToString(mac.RateLimit), mac.Comment}
}

func rateLimitDuidToInsertDBSqlString(duid *resource.RateLimitDuid) string {
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(duid.Duid)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(duid.Duid)
	buf.WriteString("','")
	buf.WriteString(uint32ToString(duid.RateLimit))
	buf.WriteString("','")
	buf.WriteString(duid.Comment)
	buf.WriteString("'),")
	return buf.String()
}

func rateLimitMacToInsertDBSqlString(mac *resource.RateLimitMac) string {
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(mac.HwAddress)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(mac.HwAddress)
	buf.WriteString("','")
	buf.WriteString(uint32ToString(mac.RateLimit))
	buf.WriteString("','")
	buf.WriteString(mac.Comment)
	buf.WriteString("'),")
	return buf.String()
}
