package server

import (
	"net"
	"strings"

	client "github.com/linkingthing/clxone-dhcp/pkg/dhcp/clients/user_service_client"
	"github.com/zdnscloud/gorest"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
)

func JWTMiddleWare() gorest.HandlerFunc {
	return func(c *restresource.Context) *resterror.APIError {
		if c.Request.RequestURI == "/health" {
			return nil
		}
		err := client.ValidateToken(c.Request.Header.Get("authorization"),
			getClientIP(c.Request.RemoteAddr))
		if err != nil {
			return resterror.NewAPIError(resterror.Unauthorized, err.Error())
		}
		return nil
	}
}

func getClientIP(remoteAddr string) string {
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr)); err == nil {
		return ip
	}

	return ""
}
