package dhcp

import (
	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/handler"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "linkingthing.com/dhcp",
	}
)

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	apiServer.Schemas.MustImport(&Version, resource.Subnet{}, handler.NewSubnetHandler())
	apiServer.Schemas.MustImport(&Version, resource.Pool{}, handler.NewPoolHandler())
	apiServer.Schemas.MustImport(&Version, resource.PdPool{}, handler.NewPdPoolHandler())
	apiServer.Schemas.MustImport(&Version, resource.Reservation{}, handler.NewReservationHandler())
	apiServer.Schemas.MustImport(&Version, resource.DhcpConfig{}, handler.NewDhcpConfigHandler())
	apiServer.Schemas.MustImport(&Version, resource.ClientClass{}, handler.NewClientClassHandler())
	apiServer.Schemas.MustImport(&Version, resource.StaticAddress{}, handler.NewStaticAddressHandler())
	apiServer.Schemas.MustImport(&Version, resource.PoolTemplate{}, handler.NewPoolTemplateHandler())
	apiServer.Schemas.MustImport(&Version, resource.Node{}, handler.NewNodeHandler())

	_, err := handler.NewScannedSubnetHandler()
	if err != nil {
		return err
	}
	conf := config.GetConfig()
	apiServer.Schemas.MustImport(&Version, resource.Dhcp{}, handler.NewDhcpHandler(conf))

	handler.NewLPSHandler(conf)
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.Subnet{},
		&resource.Pool{},
		&resource.PdPool{},
		&resource.Reservation{},
		&resource.DhcpConfig{},
		&resource.ClientClass{},
		&resource.StaticAddress{},
		&resource.PoolTemplate{},
	}
}
