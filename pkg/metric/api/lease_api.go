package api

import (
	"github.com/linkingthing/cement/log"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/service"
)

type LeaseApi struct {
	Service *service.LeaseService
}

func NewLeaseApi(config *config.DHCPConfig) *LeaseApi {
	return &LeaseApi{Service: service.NewLeaseService(config)}
}

func (h *LeaseApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	leases, err := h.Service.List(ctx)
	if err != nil {
		log.Warnf("list lease failed: %s", err.Error())
	}

	return leases, nil
}

func (h *LeaseApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	lease, err := h.Service.Get(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return lease, nil
}

func (h *LeaseApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportExcel:
		return h.ActionExport(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameMetric, ctx.Resource.GetAction().Name))
	}
}

func (h *LeaseApi) ActionExport(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := h.Service.Export(ctx); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return result, nil
	}
}
