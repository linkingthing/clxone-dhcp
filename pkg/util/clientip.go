package util

import (
	"net"
	"net/http"
	"strings"
)

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
