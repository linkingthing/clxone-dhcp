package server

import (
	"context"
	"time"

	"github.com/zdnscloud/gorest"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/proto"
	"github.com/linkingthing/clxone-dhcp/pkg/proto/user"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

func JWTMiddleWare() gorest.HandlerFunc {
	return func(c *restresource.Context) *resterror.APIError {
		conn, err := pb.NewConn(config.GetConfig().CallServices.User)
		if err != nil {
			return resterror.NewAPIError(resterror.ServerError, err.Error())
		}

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		user, err := user.NewUserServiceClient(conn).CheckToken(ctx, &user.CheckTokenRequest{
			Token:    c.Request.Header.Get("authorization"),
			ClientIp: util.ClientIP(c.Request),
		})

		if err != nil {
			return resterror.NewAPIError(resterror.Unauthorized, err.Error())
		}

		c.Set("AuthedUser", user)
		return nil
	}
}
