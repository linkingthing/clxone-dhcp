package util

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

const (
	FilterNameName       = "name"
	FilterNameComment    = "comment"
	FilterNameIp         = "ip"
	FilterNameCreateTime = "create_time"
	FilterNameVersion    = "version"
	FileNameSubnet       = "subnet"
)

func GenStrConditionsFromFilters(filters []restresource.Filter, filterNames ...string) map[string]interface{} {
	if len(filters) == 0 {
		return nil
	}

	conditions := make(map[string]interface{})
	for _, filterName := range filterNames {
		if value, ok := GetFilterValueWithEqModifierFromFilters(filterName, filters); ok {
			conditions[filterName] = value
		}
	}

	return conditions
}

func GetFilterValueWithEqModifierFromFilters(filterName string, filters []restresource.Filter) (string, bool) {
	for _, filter := range filters {
		if filter.Name == filterName && filter.Modifier == restresource.Eq {
			if len(filter.Values) == 1 && strings.TrimSpace(filter.Values[0]) != "" {
				return filter.Values[0], true
			}
			break
		}
	}

	return "", false
}

func GetFilterValueWithEqModifierFromFilter(filter restresource.Filter) (string, bool) {
	if filter.Modifier == restresource.Eq && len(filter.Values) == 1 {
		return filter.Values[0], true
	}

	return "", false
}

func GenSqlFromFilters(filters []restresource.Filter, table restdb.ResourceType, columnNames, arrayColumnNames []string, orderbyNames ...string) string {
	var buf bytes.Buffer
	buf.WriteString("SELECT * FROM gr_")
	buf.WriteString(string(table))
	buf.WriteString(" WHERE ")
	for _, filter := range filters {
		found := false
		for _, columnName := range columnNames {
			if value, validValue, foundColumn := getValueFromFilter(columnName, filter); foundColumn {
				if validValue {
					buf.WriteString(" ")
					buf.WriteString(columnName)
					buf.WriteString(" = '")
					buf.WriteString(value)
					buf.WriteString("' and ")
				}
				found = true
				break
			}
		}

		if found == false {
			for _, arrayColumnName := range arrayColumnNames {
				if value, validValue, foundColumn := getValueFromFilter(arrayColumnName, filter); foundColumn {
					if validValue {
						buf.WriteString(" '")
						buf.WriteString(value)
						buf.WriteString("' = any(")
						buf.WriteString(arrayColumnName)
						buf.WriteString(") and ")
					}
					break
				}
			}
		}
	}

	sql := strings.TrimSuffix(buf.String(), "and ")
	sql = strings.TrimSuffix(sql, "WHERE ")
	buf.Reset()
	for i, orderbyName := range orderbyNames {
		if i == 0 {
			buf.WriteString(" order by ")
		}
		buf.WriteString(orderbyName)
		buf.WriteString(",")
	}

	return sql + strings.TrimSuffix(buf.String(), ",")
}

func getValueFromFilter(filterName string, filter restresource.Filter) (string, bool, bool) {
	if filter.Name == filterName && filter.Modifier == restresource.Eq {
		if len(filter.Values) == 1 && len(filter.Values[0]) != 0 {
			return filter.Values[0], true, true
		} else {
			return "", false, true
		}
	}

	return "", false, false
}

func PrefixContainsSubnetOrIP(prefix, subnetOrIp string) (bool, error) {
	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return false, fmt.Errorf("parser prefix:%s failed:%s", prefix, err.Error())
	}

	if strings.Contains(subnetOrIp, "/") {
		if ip, subIpNet, err := net.ParseCIDR(subnetOrIp); err != nil {
			return false, fmt.Errorf("parser subnet:%s failed:%s", subnetOrIp, err.Error())
		} else {
			subSize, _ := subIpNet.Mask.Size()
			size, _ := ipNet.Mask.Size()
			return ipNet.Contains(ip) && (subSize == 0 || size <= subSize), nil
		}
	} else {
		if ip := net.ParseIP(subnetOrIp); ip == nil {
			return false, fmt.Errorf("bad ip:%s", subnetOrIp)
		} else {
			return ipNet.Contains(ip), nil
		}
	}
}

func RecombineSlices(slice1, slice2 []string, deleteSlice2 bool) []string {
	roleMap := make(map[string]bool)
	var out []string
	for _, s := range slice1 {
		roleMap[s] = true
	}

	for _, s := range slice2 {
		if deleteSlice2 {
			delete(roleMap, s)
		} else {
			roleMap[s] = true
		}
	}

	for k := range roleMap {
		out = append(out, k)
	}

	return out
}

func RemoveDuplicateOfSlices(slice1 []string) []string {
	roleMap := make(map[string]bool)
	var out []string
	for _, s := range slice1 {
		roleMap[s] = true
	}

	for k := range roleMap {
		out = append(out, k)
	}

	return out
}

func ExclusiveSlices(slice1, slice2 []string) []string {
	var out []string
	roleMap := make(map[string]bool)
	for _, s1 := range slice1 {
		roleMap[s1] = true
	}

	for _, s2 := range slice2 {
		if _, ok := roleMap[s2]; ok {
			delete(roleMap, s2)
		}
	}

	for k := range roleMap {
		out = append(out, k)
	}

	return out
}

func IsIpv6SubnetOrIps(ipv6SubnetOrIps ...string) bool {
	for _, subnetOrIp := range ipv6SubnetOrIps {
		if ip := checkSubnetOrIp(subnetOrIp); ip == nil {
			return false
		} else {
			return ip.To4() == nil
		}
	}

	return false
}

func IsIpv4SubnetOrIps(ipv4SubnetOrIps ...string) bool {
	for _, subnetOrIp := range ipv4SubnetOrIps {
		if ip := checkSubnetOrIp(subnetOrIp); ip == nil {
			return false
		} else {
			return ip.To4() != nil
		}
	}

	return false
}

func checkSubnetOrIp(subnetOrIp string) net.IP {
	if strings.Contains(subnetOrIp, "/") {
		if ip, _, err := net.ParseCIDR(subnetOrIp); err != nil {
			return nil
		} else {
			return ip
		}
	}

	return net.ParseIP(subnetOrIp)
}
