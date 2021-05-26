package metric

import (
	"github.com/gin-gonic/gin"
	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/api"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

var Version = restresource.APIVersion{
	Version: "v1",
	Group:   "linkingthing.com/dhcp/metric",
}

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	conf := config.GetConfig()
	apiServer.Schemas.MustImport(&Version, resource.Node{}, api.NewNodeHandler())
	apiServer.Schemas.MustImport(&Version, resource.Dhcp{}, api.NewDhcpHandler(conf))
	_, err := api.NewScannedSubnetHandler()
	if err != nil {
		panic(err)
	}
	api.NewLPSHandler(conf)
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.Node{},
	}
}
