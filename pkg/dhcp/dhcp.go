package dhcp

import (
	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/api"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "linkingthing.com/dhcp",
	}
)

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	apiServer.Schemas.MustImport(&Version, resource.Subnet{}, api.NewSubnetHandler())
	apiServer.Schemas.MustImport(&Version, resource.Pool{}, api.NewPoolHandler())
	apiServer.Schemas.MustImport(&Version, resource.PdPool{}, api.NewPdPoolHandler())
	apiServer.Schemas.MustImport(&Version, resource.Reservation{}, api.NewReservationHandler())
	apiServer.Schemas.MustImport(&Version, resource.DhcpConfig{}, api.NewDhcpConfigHandler())
	apiServer.Schemas.MustImport(&Version, resource.ClientClass{}, api.NewClientClassHandler())
	apiServer.Schemas.MustImport(&Version, resource.StaticAddress{}, api.NewStaticAddressHandler())
	apiServer.Schemas.MustImport(&Version, resource.PoolTemplate{}, api.NewPoolTemplateHandler())

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
