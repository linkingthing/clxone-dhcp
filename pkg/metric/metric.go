package metric

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/handler"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

var Version = restresource.APIVersion{
	Version: "v1",
	Group:   "linkingthing.com/dhcp/metric",
}

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	conf := config.GetConfig()
	apiServer.Schemas.MustImport(&Version, resource.Dhcp{}, handler.NewDhcpHandler(conf))
	apiServer.Schemas.MustImport(&Version, resource.Node{}, handler.NewNodeHandler())
	_, err := handler.NewScannedSubnetHandler()
	if err != nil {
		return err
	}
	handler.NewLPSHandler(conf)
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.Node{},
	}
}
