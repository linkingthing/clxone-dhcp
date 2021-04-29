package handler

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	resterror "github.com/zdnscloud/gorest/error"

	"github.com/linkingthing/clxone-dhcp/pkg/auth/authentification"
	"github.com/linkingthing/clxone-dhcp/pkg/auth/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

func GetUserToken(ctx *gin.Context) {
	login, err := authentification.ParseLoginBody(ctx)
	if err != nil {
		ctx.JSON(resterror.NotFound.Status,
			resource.LoginResponse{Code: resterror.NotFound.Status, Message: err.Error()})
		return
	}

	if login.Username == "" || login.Password == "" {
		ctx.JSON(resterror.NotFound.Status,
			resource.LoginResponse{Code: resterror.NotFound.Status, Message: "empty parameter"})
		return
	}

	passwordEncode, err := util.RSAEncrypt(resource.RsaPublicKey, login.Password)
	if err != nil {
		ctx.JSON(resterror.NotFound.Status,
			resource.LoginResponse{Code: resterror.NotFound.Status, Message: err.Error()})
		return
	}

	login.Password = base64.StdEncoding.EncodeToString(passwordEncode)
	if errorCode, err := authentification.CheckLogin(ctx, login); err != nil {
		ctx.JSON(errorCode.Status,
			resource.LoginResponse{Code: errorCode.Status, Message: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, resource.LoginResponse{Code: http.StatusOK})
}
