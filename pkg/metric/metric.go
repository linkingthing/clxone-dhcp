package metric

import (
	"github.com/gin-gonic/gin"
	"github.com/linkingthing/gorest"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/api"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/service"
)

var Version = restresource.APIVersion{
	Version: "v1",
	Group:   "linkingthing.com/dhcp/metric",
}

func RegisterApi(apiServer *gorest.Server, router gin.IRoutes) error {
	conf := config.GetConfig()
	if err := service.NewPrometheusClient(conf); err != nil {
		return err
	}

	if err := service.InitScannedDHCPService(conf); err != nil {
		return err
	}

	apiServer.Schemas.MustImport(&Version, resource.DhcpSentry{}, api.NewDhcpSentryApi())
	apiServer.Schemas.MustImport(&Version, resource.DhcpServer{}, api.NewDhcpServerApi())
	apiServer.Schemas.MustImport(&Version, resource.Lps{}, api.NewLPSApi(conf))
	apiServer.Schemas.MustImport(&Version, resource.Lease{}, api.NewLeaseApi(conf))
	apiServer.Schemas.MustImport(&Version, resource.LeaseTotal{}, api.NewLeaseTotalApi(conf))
	apiServer.Schemas.MustImport(&Version, resource.PacketStat{}, api.NewPacketStatApi(conf))
	apiServer.Schemas.MustImport(&Version, resource.SubnetUsedRatio{}, api.NewSubnetUsedRatioApi(conf))
	return nil
}
