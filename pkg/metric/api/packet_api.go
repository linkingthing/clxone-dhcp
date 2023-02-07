package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
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
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return stats, nil
}

func (h *PacketStatApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	packetStat, err := h.Service.Get(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return packetStat, nil
}

func (h *PacketStatApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.ActionExport(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameMetric, ctx.Resource.GetAction().Name))
	}
}

func (h *PacketStatApi) ActionExport(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := h.Service.Export(ctx); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return result, nil
	}
}
