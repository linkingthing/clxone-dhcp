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

func RegisterApi(apiServer *gorest.Server, router gin.IRoutes) error {
	//server.InitConsulHandler()
	apiServer.Schemas.MustImport(&Version, resource.SharedNetwork4{}, api.NewSharedNetwork4Api())
	apiServer.Schemas.MustImport(&Version, resource.Subnet4{}, api.NewSubnet4Api())
	apiServer.Schemas.MustImport(&Version, resource.Pool4{}, api.NewPool4Api())
	apiServer.Schemas.MustImport(&Version, resource.ReservedPool4{}, api.NewReservedPool4Api())
	apiServer.Schemas.MustImport(&Version, resource.Reservation4{}, api.NewReservation4Api())
	apiServer.Schemas.MustImport(&Version, resource.ClientClass4{}, api.NewClientClass4Api())
	apiServer.Schemas.MustImport(&Version, resource.Pool4Template{}, api.NewPool4TemplateApi())
	apiServer.Schemas.MustImport(&Version, resource.SubnetLease4{}, api.NewSubnetLease4Api())
	apiServer.Schemas.MustImport(&Version, resource.Subnet6{}, api.NewSubnet6Api())
	apiServer.Schemas.MustImport(&Version, resource.PdPool{}, api.NewPdPoolApi())
	apiServer.Schemas.MustImport(&Version, resource.ReservedPdPool{}, api.NewReservedPdPoolApi())
	apiServer.Schemas.MustImport(&Version, resource.Pool6{}, api.NewPool6Api())
	apiServer.Schemas.MustImport(&Version, resource.ReservedPool6{}, api.NewReservedPool6Api())
	apiServer.Schemas.MustImport(&Version, resource.Reservation6{}, api.NewReservation6Api())
	apiServer.Schemas.MustImport(&Version, resource.ClientClass6{}, api.NewClientClass6Api())
	apiServer.Schemas.MustImport(&Version, resource.Pool6Template{}, api.NewPool6TemplateApi())
	apiServer.Schemas.MustImport(&Version, resource.SubnetLease6{}, api.NewSubnetLease6Api())

	apiServer.Schemas.MustImport(&Version, resource.Agent4{}, api.NewAgent4Api())
	apiServer.Schemas.MustImport(&Version, resource.Agent6{}, api.NewAgent6Api())

	apiServer.Schemas.MustImport(&Version, resource.DhcpFingerprint{}, api.NewDhcpFingerprintApi())

	if dhcpConfigApi, err := api.NewDhcpConfigApi(); err != nil {
		return err
	} else {
		apiServer.Schemas.MustImport(&Version, resource.DhcpConfig{}, dhcpConfigApi)
	}

	if pingerApi, err := api.NewPingerApi(); err != nil {
		return err
	} else {
		apiServer.Schemas.MustImport(&Version, resource.Pinger{}, pingerApi)
	}

	if admitApi, err := api.NewAdmitApi(); err != nil {
		return err
	} else {
		apiServer.Schemas.MustImport(&Version, resource.Admit{}, admitApi)
	}

	apiServer.Schemas.MustImport(&Version, resource.AdmitMac{}, api.NewAdmitMacApi())
	apiServer.Schemas.MustImport(&Version, resource.AdmitDuid{}, api.NewAdmitDuidApi())
	apiServer.Schemas.MustImport(&Version, resource.AdmitFingerprint{}, api.NewAdmitFingerprintApi())

	if rateLimitApi, err := api.NewRateLimitApi(); err != nil {
		return err
	} else {
		apiServer.Schemas.MustImport(&Version, resource.RateLimit{}, rateLimitApi)
	}

	apiServer.Schemas.MustImport(&Version, resource.RateLimitMac{}, api.NewRateLimitMacApi())
	apiServer.Schemas.MustImport(&Version, resource.RateLimitDuid{}, api.NewRateLimitDuidApi())

	apiServer.Schemas.MustImport(&Version, resource.DhcpOui{}, api.NewDhcpOuiApi())
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
		&resource.AdmitDuid{},
		&resource.AdmitFingerprint{},
		&resource.RateLimit{},
		&resource.RateLimitMac{},
		&resource.RateLimitDuid{},
	}
}
