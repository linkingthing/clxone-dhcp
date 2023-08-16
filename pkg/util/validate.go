package util

import (
	"net"
	"regexp"
	"strings"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type StringRegexp struct {
	Regexp       *regexp.Regexp
	ErrMsg       string
	ExpectResult bool
}

var (
	StringRegexpsCommon = []*StringRegexp{
		{
			Regexp:       regexp.MustCompile(`^[0-9a-zA-Z-\.:_\p{Han}]+$`),
			ErrMsg:       "is illegal",
			ExpectResult: true,
		},
		{
			Regexp:       regexp.MustCompile(`(^-)|(^_)|(^:)|(^\.)`),
			ErrMsg:       "is illegal",
			ExpectResult: false,
		},
		{
			Regexp:       regexp.MustCompile(`-$|_$|:$|\.$`),
			ErrMsg:       "is illegal",
			ExpectResult: false,
		},
	}

	StringRegexpsWithSlash = []*StringRegexp{
		{
			Regexp:       regexp.MustCompile(`^[0-9a-zA-Z-\.:\\/_\p{Han}]+$`),
			ErrMsg:       "is illegal",
			ExpectResult: true,
		},
		{
			Regexp:       regexp.MustCompile(`(^-)|(^_)|(^:)|(^\.)|(^\\)|(^/)`),
			ErrMsg:       "is illegal",
			ExpectResult: false,
		},
		{
			Regexp:       regexp.MustCompile(`-$|_$|:$|\.$|\\$|/$`),
			ErrMsg:       "is illegal",
			ExpectResult: false,
		},
	}

	StringRegexpsWithSpace = []*StringRegexp{
		{
			Regexp:       regexp.MustCompile(`^[0-9a-zA-Z-\s\.:\\/_\p{Han}]+$`),
			ErrMsg:       "is illegal",
			ExpectResult: true,
		},
		{
			Regexp:       regexp.MustCompile(`(^-)|(^_)|(^:)|(^\.)|(^\\)|(^/)|(^\s)`),
			ErrMsg:       "is illegal",
			ExpectResult: false,
		},
		{
			Regexp:       regexp.MustCompile(`-$|_$|:$|\.$|\\$|/$|\s$`),
			ErrMsg:       "is illegal",
			ExpectResult: false,
		},
	}

	StringRegexpsWithComma = []*StringRegexp{
		{
			Regexp:       regexp.MustCompile(`^[0-9a-zA-Z-,，\s\.:\\/_\p{Han}]+$`),
			ErrMsg:       "is illegal",
			ExpectResult: true,
		},
		{
			Regexp:       regexp.MustCompile(`(^-)|(^_)|(^:)|(^\.)|(^\\)|(^/)|(^\s)|(^,)|(^，)`),
			ErrMsg:       "is illegal",
			ExpectResult: false,
		},
		{
			Regexp:       regexp.MustCompile(`-$|_$|:$|\.$|\\$|/$|\s$|,$|，$`),
			ErrMsg:       "is illegal",
			ExpectResult: false,
		},
	}
)

type RegexpType string

const (
	RegexpTypeCommon RegexpType = "common"
	RegexpTypeSlash  RegexpType = "slash"
	RegexpTypeSpace  RegexpType = "space"
	RegexpTypeComma  RegexpType = "comma"
)

func ValidateStrings(typ RegexpType, ss ...string) error {
	var regexps []*StringRegexp
	switch typ {
	case RegexpTypeCommon:
		regexps = StringRegexpsCommon
	case RegexpTypeSlash:
		regexps = StringRegexpsWithSlash
	case RegexpTypeSpace:
		regexps = StringRegexpsWithSpace
	case RegexpTypeComma:
		regexps = StringRegexpsWithComma
	default:
		return errorno.ErrInvalidParams(errorno.ErrNameRegexp, string(typ))
	}

	for _, s := range ss {
		if s != "" {
			for _, reg := range regexps {
				if ret := reg.Regexp.MatchString(s); ret != reg.ExpectResult {
					return errorno.ErrInvalidParams("", s)
				}
			}
		}
	}

	return nil
}

func NormalizeMac(mac string) (string, error) {
	if hw, err := net.ParseMAC(mac); err != nil {
		return "", errorno.ErrInvalidParams(errorno.ErrNameMac, mac)
	} else {
		return strings.ToUpper(hw.String()), nil
	}
}

func ToLower(ss []string) []string {
	rs := make([]string, len(ss))
	for i, s := range ss {
		rs[i] = strings.ToLower(s)
	}
	return rs
}

func IsSubnetMask(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	if tip := ip.To4(); tip != nil {
		ip = tip
	}
	var foundZero bool
	for _, bt := range ip {
		if foundZero {
			if bt != 0 {
				return false
			}
			continue
		}

		for i := 7; i >= 0; i-- {
			if (bt>>i)&1 == 0 {
				foundZero = true
			} else if foundZero {
				return false
			}
		}
	}
	return true
}
