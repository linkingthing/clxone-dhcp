package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/service"
)

type SubnetUsedRatioApi struct {
	Service *service.SubnetUsedRatioService
}

func NewSubnetUsedRatioApi(config *config.DHCPConfig) *SubnetUsedRatioApi {
	return &SubnetUsedRatioApi{Service: service.NewSubnetUsedRatioService(config)}
}

func (h *SubnetUsedRatioApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetUsedRatios, err := h.Service.List(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return subnetUsedRatios, nil
}

func (h *SubnetUsedRatioApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetUsedRatio, err := h.Service.Get(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return subnetUsedRatio, nil
}

func (h *SubnetUsedRatioApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameExportExcel:
		return h.ActionExport(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameUsedRatio, ctx.Resource.GetAction().Name))
	}
}

func (h *SubnetUsedRatioApi) ActionExport(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if result, err := h.Service.Export(ctx); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return result, nil
	}
}
