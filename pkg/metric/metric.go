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
	Group:   "linkingthing.com/metric",
}

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	conf := config.GetConfig()
	apiServer.Schemas.MustImport(&Version, resource.Node{}, handler.NewNodeHandler(conf))
	apiServer.Schemas.MustImport(&Version, resource.Dns{}, handler.NewDnsHandler(conf))
	apiServer.Schemas.MustImport(&Version, resource.Dhcp{}, handler.NewDhcpHandler(conf))
	apiServer.Schemas.MustImport(&Version, resource.AssetSearch{}, handler.NewAssetSearchHandler())
	apiServer.Schemas.MustImport(&Version, resource.RegionPortrait{}, handler.NewRegionPortraitHandler())
	apiServer.Schemas.MustImport(&Version, resource.AppTrend{}, handler.NewAppTrendHandler())
	apiServer.Schemas.MustImport(&Version, resource.AssetPortrait{}, handler.NewAssetPortraitHandler())
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.Node{},
	}
}

// for gen api doc
func RegistHandler(schemas restresource.SchemaManager) {
	schemas.MustImport(&Version, resource.Node{}, &handler.NodeHandler{})
	schemas.MustImport(&Version, resource.Dns{}, &handler.DnsHandler{})
	schemas.MustImport(&Version, resource.Dhcp{}, &handler.DhcpHandler{})
	schemas.MustImport(&Version, resource.AssetSearch{}, &handler.AssetSearchHandler{})
	schemas.MustImport(&Version, resource.RegionPortrait{}, &handler.RegionPortraitHandler{})
	schemas.MustImport(&Version, resource.AppTrend{}, &handler.AppTrendHandler{})
	schemas.MustImport(&Version, resource.AssetPortrait{}, &handler.AssetPortraitHandler{})
}
