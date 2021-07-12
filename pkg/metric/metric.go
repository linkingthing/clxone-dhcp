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
	if err := api.InitScannedDHCPHandler(conf); err != nil {
		return err
	}

	lps, err := api.NewLPSHandler(conf)
	if err != nil {
		return err
	}

	apiServer.Schemas.MustImport(&Version, resource.DhcpSentry{}, api.NewDhcpSentryHandler())
	apiServer.Schemas.MustImport(&Version, resource.DhcpServer{}, api.NewDhcpServerHandler())
	apiServer.Schemas.MustImport(&Version, resource.Lps{}, lps)
	apiServer.Schemas.MustImport(&Version, resource.Lease{}, api.NewLeaseHandler(conf))
	apiServer.Schemas.MustImport(&Version, resource.LeaseTotal{}, api.NewLeaseTotalHandler(conf))
	apiServer.Schemas.MustImport(&Version, resource.PacketStat{}, api.NewPacketStatHandler(conf))
	apiServer.Schemas.MustImport(&Version, resource.SubnetUsedRatio{}, api.NewSubnetUsedRatioHandler(conf))
	return nil
}
