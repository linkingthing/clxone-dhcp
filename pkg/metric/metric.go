package metric

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/linkingthing/gorest"
	restresource "github.com/linkingthing/gorest/resource"

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
	if err := api.NewAlarmService(conf); err != nil {
		return fmt.Errorf("register alarm failed:%s", err.Error())
	}

	if err := api.InitScannedDHCPHandler(conf); err != nil {
		return err
	}

	apiServer.Schemas.MustImport(&Version, resource.DhcpSentry{}, api.NewDhcpSentryHandler())
	apiServer.Schemas.MustImport(&Version, resource.DhcpServer{}, api.NewDhcpServerHandler())
	apiServer.Schemas.MustImport(&Version, resource.Lps{}, api.NewLPSHandler(conf))
	apiServer.Schemas.MustImport(&Version, resource.Lease{}, api.NewLeaseHandler(conf))
	apiServer.Schemas.MustImport(&Version, resource.LeaseTotal{}, api.NewLeaseTotalHandler(conf))
	apiServer.Schemas.MustImport(&Version, resource.PacketStat{}, api.NewPacketStatHandler(conf))
	apiServer.Schemas.MustImport(&Version, resource.SubnetUsedRatio{}, api.NewSubnetUsedRatioHandler(conf))
	return nil
}
