package api

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type AdmitDuIdApi struct {
	Service *service.AdmitDuIdService
}

func NewAdmitDuIdApi() *AdmitDuIdApi {
	return &AdmitDuIdApi{Service: service.NewAdmitDuIdService()}
}

func (d *AdmitDuIdApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuId := ctx.Resource.(*resource.AdmitDuid)
	admitDuid, err := d.Service.Create(admitDuId)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("create admit duid %s failed: %s", admitDuid.GetID(), err.Error()))
	}
	return admitDuid, nil
}

func (d *AdmitDuIdApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	conditions := util.GenStrConditionsFromFilters(ctx.GetFilters(), resource.FieldDuid, resource.FieldDuid)
	duids, err := d.Service.List(conditions)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list admit duids from db failed: %s", err.Error()))
	}
	return duids, nil
}

func (d *AdmitDuIdApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuidID := ctx.Resource.(*resource.AdmitDuid).GetID()
	admitDuid, err := d.Service.Get(admitDuidID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get admit duid %s from db failed: %s", admitDuidID, err.Error()))
	}
	return admitDuid.(*resource.AdmitDuid), nil
}

func (d *AdmitDuIdApi) Delete(ctx *restresource.Context) *resterror.APIError {
	admitDuidId := ctx.Resource.GetID()
	err := d.Service.Delete(admitDuidId)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete admit duid %s failed: %s", admitDuidId, err.Error()))
	}

	return nil
}

func (d *AdmitDuIdApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuid := ctx.Resource.(*resource.AdmitDuid)
	if err := admitDuid.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update admit duid %s failed: %s", admitDuid.GetID(), err.Error()))
	}
	ret, err := d.Service.Update(admitDuid)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update admit duid %s failed: %s", admitDuid.GetID(), err.Error()))
	}
	return ret, nil
}
