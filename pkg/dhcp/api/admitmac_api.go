package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type AdmitMacApi struct {
	Service *service.AdmitMacService
}

func NewAdmitMacApi() *AdmitMacApi {
	return &AdmitMacApi{Service: service.NewAdmitMacService()}
}

func (a *AdmitMacApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMac := ctx.Resource.(*resource.AdmitMac)
	if err := a.Service.Create(admitMac); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return admitMac, nil
}

func (a *AdmitMacApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	macs, err := a.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		resource.SqlColumnHwAddress, resource.SqlColumnHwAddress))
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return macs, nil
}

func (a *AdmitMacApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMac, err := a.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return admitMac, nil
}

func (a *AdmitMacApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := a.Service.Delete(ctx.Resource.GetID()); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}

func (a *AdmitMacApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMac := ctx.Resource.(*resource.AdmitMac)
	if err := a.Service.Update(admitMac); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return admitMac, nil
}
