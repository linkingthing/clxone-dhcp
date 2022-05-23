package resource

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/linkingthing/clxone-utils/validator"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
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
	MatchPattern              MatchPattern `json:"matchPattern"`
	IsReadOnly                bool         `json:"isReadOnly"`
}

func (f *DhcpFingerprint) Validate() error {
	if len(f.Fingerprint) == 0 {
		return fmt.Errorf("fingerprint is required")
	}

	for _, v := range strings.Split(f.Fingerprint, ",") {
		if i, err := strconv.Atoi(v); err != nil {
			return fmt.Errorf("fingerprint must consist of numbers and commas, but get %s", f.Fingerprint)
		} else if i <= 0 || i >= 255 {
			return fmt.Errorf("fingerprint number %s not in [1,254]", v)
		}
	}

	if err := validator.ValidateStrings(f.VendorId, f.OperatingSystem, f.ClientType); err != nil {
		return err
	}

	f.MatchPattern = MatchPatternEqual
	f.IsReadOnly = false
	return nil
}
