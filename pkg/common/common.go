package common

import (
	"github.com/gin-gonic/gin"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/common/handler"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "linkingthing.com/common",
	}
)

func RegisterHandler(group gin.RouterGroup) {
	commonGroup := group.Group(Version.GetUrl())
	commonGroup.POST("/getdispatchtoken", handler.GetUserToken)
	commonGroup.POST("/getsystemapitoken", handler.GetUserToken)
}
