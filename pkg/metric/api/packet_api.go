package api

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/config"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/service"
)

type PacketStatApi struct {
	Service *service.PacketStatService
}

func NewPacketStatApi(config *config.DHCPConfig) *PacketStatApi {
	return &PacketStatApi{Service: service.NewPacketStatService(config)}
}

func (h *PacketStatApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	stats, err := h.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list packet stats failed: %s", err.Error()))
	}
	return stats, nil
}

func (h *PacketStatApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	packetStat, err := h.Service.Get(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get packet stats failed: %s", err.Error()))
	}
	return packetStat, nil
}

func (h *PacketStatApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.ActionExport(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *PacketStatApi) ActionExport(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := h.Service.Export(ctx); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("packet stats export action failed: %s", err.Error()))
	} else {
		return result, nil
	}
}
