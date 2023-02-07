package util

import (
	"regexp"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type StringRegexp struct {
	Regexp       *regexp.Regexp
	ErrMsg       string
	ExpectResult bool
}

var StringRegexps = []*StringRegexp{
	{
		Regexp:       regexp.MustCompile(`^[0-9a-zA-Z-\.:/_\p{Han}]+$`),
		ErrMsg:       "is illegal",
		ExpectResult: true,
	},
	{
		Regexp:       regexp.MustCompile(`(^-)|(^_)|(^:)|(^/)|(^\.)`),
		ErrMsg:       "is illegal",
		ExpectResult: false,
	},
	{
		Regexp:       regexp.MustCompile(`-$|_$|:$|/$|\.$`),
		ErrMsg:       "is illegal",
		ExpectResult: false,
	},
}

func ValidateStrings(ss ...string) error {
	for _, s := range ss {
		if s != "" {
			for _, reg := range StringRegexps {
				if ret := reg.Regexp.MatchString(s); ret != reg.ExpectResult {
					return errorno.ErrInvalidParams("", s)
				}
			}
		}
	}

	return nil
}
