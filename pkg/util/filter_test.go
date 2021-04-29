package util

import (
	"encoding/base64"
	"testing"
	"time"

	restresource "github.com/zdnscloud/gorest/resource"
)

func TestPrefixContainsSubnetOrIP(t *testing.T) {
	datas := []struct {
		prefix, subnetOrIp string
		expected           bool
	}{
		{prefix: "2600:1600:100::/40", subnetOrIp: "2600:1600:103:500::/56", expected: true},
		{prefix: "2600:1600:500::/40", subnetOrIp: "2600:1600:502:2::/64", expected: true},
		{prefix: "2600:1600:600::/40", subnetOrIp: "2600:1600:502:2::/64", expected: false},
		{prefix: "2600:1600:400::/40", subnetOrIp: "2600:1600:103:500::/56", expected: false},
		{prefix: "2600:1600:400::1/32", subnetOrIp: "2600:1600:400::4", expected: true},
		{prefix: "192.168.0.0/24", subnetOrIp: "192.168.0.3", expected: true},
		{prefix: "192.178.0.1", subnetOrIp: "192.178.0.1", expected: true},
		{prefix: "2600:1600:400::/40", subnetOrIp: "", expected: false},
	}

	for _, data := range datas {
		result, err := PrefixContainsSubnetOrIP(data.prefix, data.subnetOrIp)
		if err != nil {
			t.Errorf("get err:%s", err.Error())
			continue
		}
		if result != data.expected {
			t.Errorf("<%s contains %s> tests failed expected:%t but get %t", data.prefix, data.subnetOrIp,
				data.expected, result)
		}
	}
}

func TestDiffFromSlices(t *testing.T) {
	slice1 := []string{"1"}
	slice2 := []string{""}

	t.Log(ExclusiveSlices(slice1, slice2))
}

func TestCheckSubnetPrivate(t *testing.T) {
	//11.0.0.0/24、10.0.0.0/24、111.0.0.0/24
	subnet := "111.0.0.0/24"
	if ok, err := CheckSubnetPrivate(subnet); err != nil {
		t.Error(err)
	} else {
		t.Log(ok)
	}
}

func TestRSAEncrypt(t *testing.T) {
	//GenerateRSAKey(512)

	message := "I am spider man"
	var pubPEMData = []byte(`-----BEGIN RSA Public Key-----
MFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBAK+W1jWdJh9S0WvOmv19ET6TRG2IdR5G
Vw5rKhcIZ4DQTRbsDXJ8/B5FNDrGIK5viPi7KZhi88lDAUwIDfrLzl8CAwEAAQ==
-----END RSA Public Key-----
`)

	var privateKey = []byte(`-----BEGIN RSA Private Key-----
MIIBOwIBAAJBAK+W1jWdJh9S0WvOmv19ET6TRG2IdR5GVw5rKhcIZ4DQTRbsDXJ8
/B5FNDrGIK5viPi7KZhi88lDAUwIDfrLzl8CAwEAAQJAFd+QZ7Vf3l8Ov4NJQ3Kl
B0qJJ6vsCw1wIteuspfVbPJQ4HyMQZJQxytSt3KJiNyJx0gyblAtlw2bpFITBs4K
YQIhAMIPqBeR+fxllvovl2YJhlguuZSTsOUK48JUsdVYPik9AiEA56Hf2FW0+Kad
nGsYn3Hr/h8cPqUJfAQitG6w7TmJN8sCIQCyTV5dYbN1swW4E7ggeYnlRfEfUV/L
4miH6feHFU/v5QIhAMmE+pNjFXxSsMLaJeTqHw/Kjy8tNFAx5OOnfcQVj3z7AiAa
5KcgxuRX5VChUiXL8bl0kDgeuipk+48OUFqplN13hg==
-----END RSA Private Key-----
`)
	if code, err := RSAEncrypt(pubPEMData, message); err != nil {
		t.Errorf(err.Error())
	} else {
		encoded := base64.StdEncoding.EncodeToString(code)
		t.Log(encoded, len(encoded))
		ss, _ := base64.StdEncoding.DecodeString(encoded)
		if out, err := RSADecrypt(privateKey, ss); err != nil {
			t.Errorf(err.Error())
		} else {
			t.Log(string(out))
		}
	}
}

func TestRecombineSlices(t *testing.T) {
	slice1 := []string{"1", "2"}
	slice2 := []string{"2"}
	t.Log(RecombineSlices(slice1, slice2, true))
}

func TestGenSqlFromFilters(t *testing.T) {
	filters := []restresource.Filter{
		{Name: "name", Values: []string{"a1"}},
		{Name: "ips", Values: []string{"10.0.0.0/24"}},
		{Name: "comment", Values: []string{"hello"}},
		{Name: "aaa", Values: []string{"hello"}},
		{Name: "acls", Values: []string{"any"}},
	}

	t.Log(GenSqlFromFilters(filters, "acl", []string{"name", "comment", "cd"}, []string{"ips", "acls", "ppp"}, "create_time"))
}

func TestFormatTime(t *testing.T) {
	timeStr := "2021-02-02T21:56:43.321Z"
	time_, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		t.Error(err)
	}
	t.Log(time_.Format(TimeFormat))
}
