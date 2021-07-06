package util

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
)

func ParseIP(ipstr string) (net.IP, bool, error) {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil, false, fmt.Errorf("invalid ip %s", ipstr)
	} else {
		return ip, ip.To4() != nil, nil
	}
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

func Ipv4FromUint32(val uint32) net.IP {
	addr := make([]byte, 4)
	binary.BigEndian.PutUint32(addr, val)
	return net.IP(addr)
}

func Ipv4StringToUint32(ipv4 string) (uint32, bool) {
	return Ipv4ToUint32(net.ParseIP(ipv4))
}

func Ipv4ToUint32(ipv4 net.IP) (uint32, bool) {
	if ipv4_ := ipv4.To4(); ipv4_ == nil {
		return 0, false
	} else {
		return binary.BigEndian.Uint32(ipv4_), true
	}
}

func Ipv6StringToBigInt(ipv6 string) (*big.Int, bool) {
	return Ipv6ToBigInt(net.ParseIP(ipv6))
}

func Ipv6ToBigInt(ipv6 net.IP) (*big.Int, bool) {
	if ipv6.To4() != nil {
		return nil, false
	}

	ipv6Int := big.NewInt(0)
	ipv6Int.SetBytes(ipv6.To16())
	return ipv6Int, true
}

func OneIpLessThanAnother(one, another string) bool {
	oneIP := net.ParseIP(one)
	anotherIP := net.ParseIP(another)
	if oneIP.To4() != nil && anotherIP.To4() == nil {
		return true
	}

	if oneIP.To4() == nil && anotherIP.To4() != nil {
		return false
	}

	if oneIP.To4() != nil {
		oneUint32, _ := Ipv4ToUint32(oneIP)
		anotherUint32, _ := Ipv4ToUint32(anotherIP)
		return oneUint32 < anotherUint32
	} else {
		oneBigInt, _ := Ipv6ToBigInt(oneIP)
		anotherBigInt, _ := Ipv6ToBigInt(anotherIP)
		return oneBigInt.Cmp(anotherBigInt) == -1
	}
}

func ClientIP(request *http.Request) string {
	clientIP := request.Header.Get("X-Forwarded-For")
	clientIPs := strings.Split(clientIP, ",")
	for _, ip := range clientIPs {
		if strings.TrimSpace(ip) == "127.0.0.1" {
			continue
		}

		clientIP = strings.TrimSpace(ip)
	}

	if len(clientIPs) == 0 {
		clientIP = strings.TrimSpace(request.Header.Get("X-Real-Ip"))
	}

	if clientIP != "" {
		return clientIP
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(request.RemoteAddr)); err == nil {
		return ip
	}

	return ""
}
