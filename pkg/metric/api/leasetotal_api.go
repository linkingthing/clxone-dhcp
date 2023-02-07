package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/service"
)

type LeaseTotalApi struct {
	Service *service.LeaseTotalService
}

func NewLeaseTotalApi(config *config.DHCPConfig) *LeaseTotalApi {
	return &LeaseTotalApi{Service: service.NewLeaseTotalService(config)}
}

func (h *LeaseTotalApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	leases, err := h.Service.List(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return leases, nil
}

func (h *LeaseTotalApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	lease, err := h.Service.Get(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return lease, nil
}

func (h *LeaseTotalApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.ActionExport(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameLease, ctx.Resource.GetAction().Name))
	}
}

func (h *LeaseTotalApi) ActionExport(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := h.Service.Export(ctx); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return result, nil
	}
}
