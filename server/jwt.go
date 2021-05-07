package server

import (
	"context"
	"net"
	"strings"
	"time"

	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/dhcp/grpc_clients"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/grpc_clients/user_service/pb"
	"github.com/zdnscloud/gorest"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
)

func JWTMiddleWare() gorest.HandlerFunc {
	return func(c *restresource.Context) *resterror.APIError {
		token := c.Request.Header.Get("authorization")
		clientIP := getClientIP(c.Request.RemoteAddr)

		conn, err := grpcclient.NewClient("clxone-user-grpc")
		if err != nil {
			return resterror.NewAPIError(resterror.Unauthorized, err.Error())
		}
		defer conn.Close()
		cli := pb.NewUserServiceClient(conn)

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		user, err := cli.CheckToken(ctx, &pb.CheckTokenRequest{
			Token:    token,
			ClientIP: clientIP,
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
