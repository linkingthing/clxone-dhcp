package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type AdmitDuidApi struct {
	Service *service.AdmitDuidService
}

func NewAdmitDuidApi() *AdmitDuidApi {
	return &AdmitDuidApi{Service: service.NewAdmitDuidService()}
}

func (a *AdmitDuidApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuid := ctx.Resource.(*resource.AdmitDuid)
	if err := a.Service.Create(admitDuid); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitDuid, nil
}

func (a *AdmitDuidApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	duids, err := a.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		resource.SqlColumnDuid, resource.SqlColumnDuid))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return duids, nil
}

func (a *AdmitDuidApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuid, err := a.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitDuid, nil
}

func (a *AdmitDuidApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := a.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (a *AdmitDuidApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuid := ctx.Resource.(*resource.AdmitDuid)
	if err := a.Service.Update(admitDuid); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitDuid, nil
}
