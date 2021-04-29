package util

import (
	"fmt"
	"regexp"
	"strings"
)

type NameRegexp struct {
	Regexp       *regexp.Regexp
	ErrMsg       string
	ExpectResult bool
}

type CheckName interface {
	GetNameRegexps() []*NameRegexp
}

var NameRegs = []*NameRegexp{
	{
		Regexp:       regexp.MustCompile(`^[0-9a-zA-Z-_\p{Han}]+$`),
		ErrMsg:       "name is not legal",
		ExpectResult: true,
	},
	{
		Regexp:       regexp.MustCompile(`(?i)^cmcc$|^ctcc$|^cucc$|^any$|^default$|^none$`),
		ErrMsg:       "name is not legal",
		ExpectResult: false,
	},
	{
		Regexp:       regexp.MustCompile(`(_$|-$)|(^-)|(^\.)|(\.{2})|(_{2})|(-{2})`),
		ErrMsg:       "name is not legal",
		ExpectResult: false,
	},
}

var DomainNameRegs = []*NameRegexp{
	{
		Regexp:       regexp.MustCompile(`(^[\w.-]{1,63}$)|(^[*@]$)`),
		ErrMsg:       "name is not legal",
		ExpectResult: true,
	},
	{
		Regexp:       regexp.MustCompile(`(^-)|(^\.)|(\.{2})|(_{2})|(-{2})`),
		ErrMsg:       "name is not legal",
		ExpectResult: false,
	},
}

var ZoneNameRegs = []*NameRegexp{
	{
		Regexp:       regexp.MustCompile(`^[\w@.-]+$`),
		ErrMsg:       "name is not legal",
		ExpectResult: true,
	}, {
		Regexp:       regexp.MustCompile(`(_$|-$)|(\.{2})|(@{2})|(_{2})|(-{2})`),
		ErrMsg:       "name is not legal",
		ExpectResult: false,
	},
}

var DomainNamesRegs = []*NameRegexp{
	{
		Regexp:       regexp.MustCompile(`(^([\w-_.]+|[*@])\.[a-zA-Z]+$)|(^[\w]+$)`),
		ErrMsg:       "name is not legal",
		ExpectResult: true,
	},
	{
		Regexp:       regexp.MustCompile(`(^-)|(^\.)|(\*$)|(-$)|(_$)`),
		ErrMsg:       "name is not legal",
		ExpectResult: false,
	},
}

func CheckNameValid(name string) error {
	for _, reg := range NameRegs {
		if ret := reg.Regexp.MatchString(name); ret != reg.ExpectResult {
			return fmt.Errorf(reg.ErrMsg)
		}
	}
	return nil
}

func CheckDomainNamesValid(name string) error {
	if len(name) > 255 {
		return fmt.Errorf("max length is 255")
	}

	for _, reg := range DomainNamesRegs {
		if ret := reg.Regexp.MatchString(name); ret != reg.ExpectResult {
			return fmt.Errorf(reg.ErrMsg)
		}
	}
	return nil
}

func CheckRRNameValid(name string) error {
	for _, reg := range DomainNameRegs {
		if ret := reg.Regexp.MatchString(name); ret != reg.ExpectResult {
			return fmt.Errorf(reg.ErrMsg)
		}
	}
	return nil
}

func CheckZoneNameValid(name string) error {
	for _, reg := range ZoneNameRegs {
		if ret := reg.Regexp.MatchString(name); ret != reg.ExpectResult {
			return fmt.Errorf(reg.ErrMsg)
		}
	}
	return nil
}

func IsBrokenPipeErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "broken pipe") ||
		strings.Contains(strings.ToLower(err.Error()), "connection reset by peer")
}
