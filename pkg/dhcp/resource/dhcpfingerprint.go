package resource

import (
	"strconv"
	"strings"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
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
		return errorno.ErrEmpty(string(errorno.ErrNameFingerprint))
	}

	for _, v := range strings.Split(f.Fingerprint, ",") {
		if i, err := strconv.Atoi(v); err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameFingerprint, f.Fingerprint)
		} else if i <= 0 || i >= 255 {
			return errorno.ErrNotInScope(errorno.ErrNameFingerprint, 1, 254)
		}
	}

	if err := util.ValidateStrings(util.RegexpTypeCommon, f.VendorId, f.OperatingSystem, f.ClientType); err != nil {
		return err
	}

	f.MatchPattern = MatchPatternEqual
	f.IsReadOnly = false
	return nil
}
