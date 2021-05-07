package server

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/linkingthing/clxone-dhcp/pkg/proto"
	"github.com/zdnscloud/gorest"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
)

func JWTMiddleWare() gorest.HandlerFunc {
	return func(c *restresource.Context) *resterror.APIError {
		token := c.Request.Header.Get("authorization")
		clientIP := getClientIP(c.Request.RemoteAddr)

		conn, err := proto.NewClient("clxone-user-grpc")
		if err != nil {
			return resterror.NewAPIError(resterror.Unauthorized, err.Error())
		}
		defer conn.Close()
		cli := proto.NewUserServiceClient(conn)

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		user, err := cli.CheckToken(ctx, &proto.CheckTokenRequest{
			Token:    token,
			ClientIp: clientIP,
		})

		if err != nil {
			return resterror.NewAPIError(resterror.Unauthorized, err.Error())
		}

		c.Set("AuthedUser", user)
		return nil
	}
}

func getClientIP(remoteAddr string) string {
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr)); err == nil {
		return ip
	}

	return ""
}
