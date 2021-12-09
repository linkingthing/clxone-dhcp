package dhcp

import (
	"github.com/gin-gonic/gin"

	"github.com/linkingthing/gorest"
	restresource "github.com/linkingthing/gorest/resource"

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
	api.InitConsulHandler()
	apiServer.Schemas.MustImport(&Version, resource.SharedNetwork4{}, api.NewSharedNetwork4Handler())
	apiServer.Schemas.MustImport(&Version, resource.Subnet4{}, api.NewSubnet4Handler())
	apiServer.Schemas.MustImport(&Version, resource.Pool4{}, api.NewPool4Handler())
	apiServer.Schemas.MustImport(&Version, resource.ReservedPool4{}, api.NewReservedPool4Handler())
	apiServer.Schemas.MustImport(&Version, resource.Reservation4{}, api.NewReservation4Handler())
	apiServer.Schemas.MustImport(&Version, resource.ClientClass4{}, api.NewClientClass4Handler())
	apiServer.Schemas.MustImport(&Version, resource.Pool4Template{}, api.NewPool4TemplateHandler())
	apiServer.Schemas.MustImport(&Version, resource.SubnetLease4{}, api.NewSubnetLease4Handler())

	apiServer.Schemas.MustImport(&Version, resource.Subnet6{}, api.NewSubnet6Handler())
	apiServer.Schemas.MustImport(&Version, resource.PdPool{}, api.NewPdPoolHandler())
	apiServer.Schemas.MustImport(&Version, resource.ReservedPdPool{}, api.NewReservedPdPoolHandler())
	apiServer.Schemas.MustImport(&Version, resource.Pool6{}, api.NewPool6Handler())
	apiServer.Schemas.MustImport(&Version, resource.ReservedPool6{}, api.NewReservedPool6Handler())
	apiServer.Schemas.MustImport(&Version, resource.Reservation6{}, api.NewReservation6Handler())
	apiServer.Schemas.MustImport(&Version, resource.ClientClass6{}, api.NewClientClass6Handler())
	apiServer.Schemas.MustImport(&Version, resource.Pool6Template{}, api.NewPool6TemplateHandler())
	apiServer.Schemas.MustImport(&Version, resource.SubnetLease6{}, api.NewSubnetLease6Handler())

	apiServer.Schemas.MustImport(&Version, resource.Agent4{}, api.NewAgent4Handler())
	apiServer.Schemas.MustImport(&Version, resource.Agent6{}, api.NewAgent6Handler())

	apiServer.Schemas.MustImport(&Version, resource.DhcpFingerprint{}, api.NewDhcpFingerprintHandler())
	if dhcpConfigHandler, err := api.NewDhcpConfigHandler(); err != nil {
		return err
	} else {
		apiServer.Schemas.MustImport(&Version, resource.DhcpConfig{}, dhcpConfigHandler)
	}

	if pingerHandler, err := api.NewPingerHandler(); err != nil {
		return err
	} else {
		apiServer.Schemas.MustImport(&Version, resource.Pinger{}, pingerHandler)
	}

	if admitHandler, err := api.NewAdmitHandler(); err != nil {
		return err
	} else {
		apiServer.Schemas.MustImport(&Version, resource.Admit{}, admitHandler)
	}

	apiServer.Schemas.MustImport(&Version, resource.AdmitMac{}, api.NewAdmitMacHandler())
	apiServer.Schemas.MustImport(&Version, resource.AdmitFingerprint{}, api.NewAdmitFingerprintHandler())
	apiServer.Schemas.MustImport(&Version, resource.DhcpOui{}, api.NewDhcpOuiHandler())
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.SharedNetwork4{},
		&resource.Subnet4{},
		&resource.Pool4{},
		&resource.ReservedPool4{},
		&resource.Reservation4{},
		&resource.ClientClass4{},
		&resource.Pool4Template{},
		&resource.Subnet6{},
		&resource.Pool6{},
		&resource.ReservedPool6{},
		&resource.PdPool{},
		&resource.ReservedPdPool{},
		&resource.Reservation6{},
		&resource.ClientClass6{},
		&resource.Pool6Template{},
		&resource.DhcpConfig{},
		&resource.DhcpFingerprint{},
		&resource.SubnetLease4{},
		&resource.SubnetLease6{},
		&resource.Pinger{},
		&resource.DhcpOui{},
		&resource.Admit{},
		&resource.AdmitMac{},
		&resource.AdmitFingerprint{},
	}
}
