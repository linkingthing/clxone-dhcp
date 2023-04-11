package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
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
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list leases total count failed: %s", err.Error()))
	}
	return leases, nil
}

func (h *LeaseTotalApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	lease, err := h.Service.Get(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("get leases count with failed: %s", err.Error()))
	}
	return lease, nil
}

func (h *LeaseTotalApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportExcel:
		return h.ActionExport(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *LeaseTotalApi) ActionExport(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := h.Service.Export(ctx); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("leases count %s export action failed: %s", ctx.Resource.GetID(), err.Error()))
	} else {
		return result, nil
	}
}
