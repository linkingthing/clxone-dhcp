package server

import (
	"context"
	"time"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/pb"
	"github.com/linkingthing/clxone-dhcp/pkg/pb/user"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/zdnscloud/gorest"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"
)

func JWTMiddleWare() gorest.HandlerFunc {
	return func(c *restresource.Context) *resterror.APIError {
		token := c.Request.Header.Get("authorization")
		clientIP := util.ClientIP(c.Request)

		conn, err := pb.NewConn(config.GetConfig().CallServices.User)
		if err != nil {
			logrus.Error(err)
			return resterror.NewAPIError(resterror.ServerError, err.Error())
		}
		cli := user.NewUserServiceClient(conn)

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		user, err := cli.CheckToken(ctx, &user.CheckTokenRequest{
			Token:    token,
			ClientIp: clientIP,
		})

		if err != nil {
			logrus.Error(err)
			return resterror.NewAPIError(resterror.ServerError, err.Error())
		}

		c.Set("AuthedUser", user)
		return nil
	}
}
