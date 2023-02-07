package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/service"
)

type LPSApi struct {
	Service *service.LPSService
}

func NewLPSApi(config *config.DHCPConfig) *LPSApi {
	return &LPSApi{Service: service.NewLPSService(config)}
}

func (h *LPSApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	lpses, err := h.Service.List(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return lpses, nil
}

func (h *LPSApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	lps, err := h.Service.Get(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return lps, nil
}

func (h *LPSApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.ActionExport(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameLPS, ctx.Resource.GetAction().Name))
	}
}

func (h *LPSApi) ActionExport(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := h.Service.Export(ctx); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return result, nil
	}
}
