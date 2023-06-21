package util

import (
	"fmt"
	"net"
	"regexp"
	"strings"
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
		return fmt.Errorf("no found regexp type %s", typ)
	}

	for _, s := range ss {
		if s != "" {
			for _, reg := range regexps {
				if ret := reg.Regexp.MatchString(s); ret != reg.ExpectResult {
					return fmt.Errorf("%s %s", s, reg.ErrMsg)
				}
			}
		}
	}

	return nil
}

func NormalizeMac(mac string) (string, error) {
	if hw, err := net.ParseMAC(mac); err != nil {
		return "", err
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
