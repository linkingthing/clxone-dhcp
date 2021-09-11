package resource

import (
	"fmt"
	"strconv"
	"strings"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

type MatchPattern string

const (
	MatchPatternEqual   MatchPattern = "equal"
	MatchPatternPrefix  MatchPattern = "prefix"
	MatchPatternSuffix  MatchPattern = "suffix"
	MatchPatternKeyword MatchPattern = "keyword"
	MatchPatternRegexp  MatchPattern = "regexp"
)

func (m MatchPattern) Validate() bool {
	return m == MatchPatternEqual || m == MatchPatternPrefix ||
		m == MatchPatternSuffix || m == MatchPatternKeyword ||
		m == MatchPatternRegexp
}

var TableDhcpFingerprint = restdb.ResourceDBType(&DhcpFingerprint{})

type DhcpFingerprint struct {
	restresource.ResourceBase `json:",inline"`
	Fingerprint               string       `json:"fingerprint" rest:"required=true" db:"uk"`
	VendorId                  string       `json:"vendorId" db:"uk"`
	OperatingSystem           string       `json:"operatingSystem" db:"uk"`
	ClientType                string       `json:"clientType" db:"uk"`
	MatchPattern              MatchPattern `json:"matchPattern" rest:"required=true"`
	IsReadOnly                bool         `json:"isReadOnly"`
}

func (f *DhcpFingerprint) Validate() error {
	for _, v := range strings.Split(f.Fingerprint, ",") {
		if _, err := strconv.Atoi(v); err != nil {
			return fmt.Errorf("fingerprint must consist of numbers and commas, but get %s", f.Fingerprint)
		}
	}

	if f.MatchPattern.Validate() == false {
		return fmt.Errorf("match pattern %s not in [equal, prefix, suffix, keyword, regexp]", f.MatchPattern)
	}

	f.IsReadOnly = false
	return nil
}
