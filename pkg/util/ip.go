package util

import (
	"fmt"
	"net"
	"strconv"

	restresource "github.com/zdnscloud/gorest/resource"
)

type IPVersion uint32

const (
	IPVersion4 IPVersion = 4
	IPVersion6 IPVersion = 6
)

func (v IPVersion) Validate() bool {
	return v == IPVersion4 || v == IPVersion6
}

func (v IPVersion) IsEmpty() bool {
	return uint32(v) == 0
}

func IPVersionFromFilter(filters []restresource.Filter) (IPVersion, bool) {
	if versionStr, ok := GetFilterValueWithEqModifierFromFilters(FilterNameVersion, filters); ok {
		if versionInt, err := strconv.Atoi(versionStr); err == nil && IPVersion(versionInt).Validate() {
			return IPVersion(versionInt), true
		}
	}

	return IPVersion(0), false
}

func ParseIP(ipstr string) (net.IP, bool, error) {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil, false, fmt.Errorf("invalid ip %s", ipstr)
	} else {
		return ip, ip.To4() != nil, nil
	}
}

func ParsePrefixVersion(prefix string, version IPVersion) error {
	ip, ipnet, err := net.ParseCIDR(prefix)
	if err != nil {
		return err
	}

	if version == IPVersion4 && ip.To4() == nil {
		return fmt.Errorf("%s is not ipv4 prefix", prefix)
	}

	if version == IPVersion6 {
		if ip.To4() != nil {
			return fmt.Errorf("%s is not ipv6 prefix", prefix)
		} else if ones, _ := ipnet.Mask.Size(); ones > 64 {
			return fmt.Errorf("ip mask size %d is bigger than 64", ones)
		}
	}

	return nil
}

func CheckIPsValidWithVersion(isv4 bool, ips ...string) error {
	for _, ip := range ips {
		_, isv4_, err := ParseIP(ip)
		if err != nil {
			return err
		}

		if isv4 != isv4_ {
			return fmt.Errorf("ip %s is diff from expect version", ip)
		}
	}

	return nil
}

func CheckIPsValid(ips ...string) error {
	for _, ip := range ips {
		if net.ParseIP(ip) == nil {
			return fmt.Errorf("invalid ip %s", ip)
		}
	}

	return nil
}

func CheckAddressValid(addresses ...string) error {
	for _, address := range addresses {
		if net.ParseIP(address) == nil {
			if _, err := net.ResolveTCPAddr("tcp", address); err != nil {
				return err
			}
		}
	}

	return nil
}

var (
	PrivateSubnetA  = &net.IPNet{IP: net.IP{10, 0, 0, 0}, Mask: net.IPMask{255, 0, 0, 0}}
	PrivateSubnetB  = &net.IPNet{IP: net.IP{172, 16, 0, 0}, Mask: net.IPMask{255, 16, 0, 0}}
	PrivateSubnetC  = &net.IPNet{IP: net.IP{192, 168, 0, 0}, Mask: net.IPMask{255, 255, 0, 0}}
	PrivateSubnetLo = &net.IPNet{IP: net.IP{127, 0, 0, 0}, Mask: net.IPMask{255, 0, 0, 0}}
)

/*
ref rfc1918:Private Address Space:
10.0.0.0        -   10.255.255.255  (10/8 prefix)
172.16.0.0      -   172.31.255.255  (172.16/12 prefix)
192.168.0.0     -   192.168.255.255 (192.168/16 prefix)
127.0.0.0     -   192.168.255.255 (127/8 prefix)
*/
func CheckSubnetPrivate(subNet string) (bool, error) {
	if ip, ipnet, err := net.ParseCIDR(subNet); err != nil {
		return false, err
	} else {
		ones, _ := ipnet.Mask.Size()
		onesA, _ := PrivateSubnetA.Mask.Size()
		onesB, _ := PrivateSubnetB.Mask.Size()
		onesC, _ := PrivateSubnetC.Mask.Size()
		onesLo, _ := PrivateSubnetLo.Mask.Size()
		switch {
		case PrivateSubnetA.Contains(ip) && onesA < ones:
			return true, nil
		case PrivateSubnetB.Contains(ip) && onesB < ones:
			return true, nil
		case PrivateSubnetC.Contains(ip) && onesC < ones:
			return true, nil
		case PrivateSubnetLo.Contains(ip) && onesLo < ones:
			return true, nil
		default:
			return false, nil
		}
	}
}

func CheckPrefixsContainEachOther(parentPrefix string, subNet *net.IPNet) bool {
	_, parentNet, err := net.ParseCIDR(parentPrefix)
	if err != nil {
		return false
	}

	parentOnes, _ := parentNet.Mask.Size()
	subOnes, _ := subNet.Mask.Size()
	if parentNet.Contains(subNet.IP) && parentOnes <= subOnes {
		return true
	}

	if subNet.Contains(parentNet.IP) && subOnes <= parentOnes {
		return true
	}

	return false
}

func PrefixsContainEachOther(parentPrefix, subPrefix string) (bool, error) {
	_, parentNet, err := net.ParseCIDR(parentPrefix)
	if err != nil {
		return false, err
	}
	_, subNet, err := net.ParseCIDR(subPrefix)
	if err != nil {
		return false, err
	}

	parentOnes, _ := parentNet.Mask.Size()
	subOnes, _ := subNet.Mask.Size()
	if parentNet.Contains(subNet.IP) && parentOnes <= subOnes {
		return true, nil
	}

	if subNet.Contains(parentNet.IP) && subOnes <= parentOnes {
		return true, nil
	}

	return false, nil
}

func PrefixsContainsIpNet(parentPrefix string, subNet net.IPNet) bool {
	_, parentNet, err := net.ParseCIDR(parentPrefix)
	if err != nil {
		return false
	}

	parentOnes, _ := parentNet.Mask.Size()
	subOnes, _ := subNet.Mask.Size()
	if parentNet.Contains(subNet.IP) && parentOnes <= subOnes {
		return true
	}

	return false
}

func PrefixsContains(parentPrefix string, subPrefix string) bool {
	_, parentNet, err := net.ParseCIDR(parentPrefix)
	if err != nil {
		return false
	}
	_, subNet, err := net.ParseCIDR(subPrefix)
	if err != nil {
		return false
	}

	parentOnes, _ := parentNet.Mask.Size()
	subOnes, _ := subNet.Mask.Size()
	if parentNet.Contains(subNet.IP) && parentOnes <= subOnes {
		return true
	}

	return false
}

func PrefixsNotEqualContains(parentPrefix string, subPrefix string) bool {
	_, parentNet, err := net.ParseCIDR(parentPrefix)
	if err != nil {
		return false
	}
	_, subNet, err := net.ParseCIDR(subPrefix)
	if err != nil {
		return false
	}

	parentOnes, _ := parentNet.Mask.Size()
	subOnes, _ := subNet.Mask.Size()
	if parentNet.Contains(subNet.IP) && parentOnes < subOnes {
		return true
	}

	return false
}

func PrefixListContains(parentPrefixs []string, prefix string) (bool, string, string, error) {
	for _, parentPrefix := range parentPrefixs {
		if PrefixsContains(parentPrefix, prefix) {
			return true, parentPrefix, prefix, nil
		}
	}

	return false, "", "", nil
}

func CheckSubnetListContainSubnet(parentPrefixs []*net.IPNet, subNet *net.IPNet) (bool, string) {
	for _, parentPrefix := range parentPrefixs {
		if CheckPrefixsContainEachOther(parentPrefix.String(), subNet) {
			return true, parentPrefix.String()
		}
	}

	return false, ""
}

func PrefixsEqual(prevPrefix, prefix string) bool {
	_, prevNet, err := net.ParseCIDR(prevPrefix)
	if err != nil {
		return false
	}

	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		return false
	}

	parentOnes, _ := prevNet.Mask.Size()
	subOnes, _ := ipNet.Mask.Size()
	return prevNet.IP.Equal(ipNet.IP) && parentOnes == subOnes
}
