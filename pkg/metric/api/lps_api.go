package api

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/config"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

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
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list lpses failed: %s"+err.Error()))
	}
	return lpses, nil
}

func (h *LPSApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	lps, err := h.Service.Get(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("get lps failed: %s", err.Error()))
	}
	return lps, nil
}

func (h *LPSApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportCSV:
		return h.ActionExport(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *LPSApi) ActionExport(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := h.Service.Export(ctx); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("lps export action failed: %s", err.Error()))
	} else {
		return result, nil
	}
}
