package util

import (
	"fmt"
	"strings"
	"testing"

	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/zdnscloud/g53"
)

func TestCompare(t *testing.T) {
	addr := "10.0.0.111:58081"
	fmt.Println(strings.Contains(addr, "10.0.0.111"))
}

func TestIsContainsSubnetOrIP(t *testing.T) {
	tests := []struct {
		ipEntireNet, ipSubnet string
		expected              bool
	}{
		{ipEntireNet: "2008::/32", ipSubnet: "2008::/32", expected: true},
		{ipEntireNet: "2008::/32", ipSubnet: "2007::/32", expected: false},
		{ipEntireNet: "2008::/32", ipSubnet: "2008:0:0:10::/60", expected: true},
		{ipEntireNet: "2008::/32", ipSubnet: "2008:0:0:12::/64", expected: true},
		{ipEntireNet: "2008::/32", ipSubnet: "2007:0:0:13::/64", expected: false},
	}

	for _, tt := range tests {
		result, err := PrefixContainsSubnetOrIP(tt.ipEntireNet, tt.ipSubnet)
		if err != nil {
			t.Errorf("get err:%s", err.Error())
		}
		if result != tt.expected {
			t.Errorf("PrefixContainsSubnetOrIP failed:expected:%t but get %t", tt.expected, result)
		}
	}
}

func TestCheckNameValid(t *testing.T) {
	names := []string{
		"a", "-", "123", "abc12", "any", "你好", "270沟", "眸", "北京a2", "圎圏圎圐圑园圆圔圕图圗团圙圚圛圜", "尐尒尕尗尛尜尞尟尠",
		"1.2.3", "1_b", "a-c", "_", "111", "]", "你.1", "bihaoa是重～点点",
		"a-", "-a", "_a", "b_", "b-", "-b", "asd-", "-123-"}

	for _, name := range names {
		if err := CheckNameValid(name); err == nil {
			t.Logf("valid name:%s", name)
		}
	}
}

func TestRRName(t *testing.T) {
	names := []string{"__", "_", "a-b", "-", "abc", "b_a", "123", "a@d", "@@", "a.b", "a.b.c.", ".", "..",
		"b-", "-n", "--fn", "a*b", "@", "*", "_asd_", "-123-", "_fd", "__rd", "_g_f", "12-45"}
	for _, name := range names {
		if err := CheckRRNameValid(name); err == nil {
			t.Logf("valid name:%s", name)
		}
	}
}

func TestDomainNames(t *testing.T) {
	names := []string{"www.baidu.com", "*.baidu.com", "a_b.baidu.com", "bbc.com", "bbc.com_", "bbc.com-", "a8b.cc.afa.si5.com",
		"abc.*", "*.com", ".", "a.c", "__", "_", "a-b.cn", "-", "abc", "b_a", "123.cn", "@.con", "@@", "@*",
		"a.b", "a.b.c.", ".com", "..", "@a.com", "b-", "f.a-n", "--fn", "a*b", "@.cc.com", "*", "asd_", "a*.b.com",
		"_fd", "__rd", "_g_f", "12-45", ".cn", "abc"}

	var count = 0
	for _, name := range names {
		if err := CheckDomainNamesValid(name); err == nil {
			t.Logf("valid name:%s", name)
			count++
		}
	}
	t.Log("total:", count)
}

func TestCheckZoneNameValid(t *testing.T) {
	names := []string{"__", "_", "a-b", "-", "abc", "b_a", "123", "a@d", "@@", "a.b", "a.b.c.", ".", "..",
		"b-", "-n", "--fn", "a*b", "@", "*", "_asd_", "-123-", "_fd", "__rd", "_g_f"}
	for _, name := range names {
		if err := CheckZoneNameValid(name); err == nil {
			t.Logf("valid name:%s", name)
		}
	}
}

func TestG53(t *testing.T) {
	names := []string{"@", "."}
	for _, name := range names {
		name, err := g53.NameFromString(name)
		if err != nil {
			t.Errorf(err.Error())
		}

		t.Log(name.String(true))
	}
}

func TestAddress(t *testing.T) {
	address := []string{"10.0.0.201:53", "20.10.203.51", "10.0.300.2:6666", "2108::1", "[2009::ff1]:58082"}
	if err := CheckAddressValid(address...); err != nil {
		t.Errorf(err.Error())
	}
}

func TestContext(t *testing.T) {
	ss := GetCurrentUser(&restresource.Context{})
	t.Log("ss", ss)
}

func GetCurrentUser(ctx *restresource.Context) string {
	if user, ok := ctx.Get("admin"); ok {
		return user.(string)
	} else {
		return ""
	}
}
