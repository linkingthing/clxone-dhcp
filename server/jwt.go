package server

import (
	"context"
	"time"

	"github.com/linkingthing/gorest"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	pb "github.com/linkingthing/clxone-dhcp/pkg/proto"
	pbuser "github.com/linkingthing/clxone-dhcp/pkg/proto/user"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

func JWTMiddleWare() gorest.HandlerFunc {
	return func(c *restresource.Context) *resterror.APIError {
		conn, err := pb.NewConn(config.GetConfig().CallServices.User)
		if err != nil {
			return resterror.NewAPIError(resterror.ServerError, err.Error())
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		user, err := pbuser.NewUserServiceClient(conn).CheckToken(ctx, &pbuser.CheckTokenRequest{
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
