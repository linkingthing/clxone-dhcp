package api

import (
	"fmt"

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

func (d *AdmitMacApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMac := ctx.Resource.(*resource.AdmitMac)
	admitMac.SetID(admitMac.HwAddress)
	if err := admitMac.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create admit mac %s failed: %s", admitMac.GetID(), err.Error()))
	}
	newAdmitMac, err := d.Service.Create(admitMac)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create admit mac %s failed: %s", admitMac.GetID(), err.Error()))
	}
	return newAdmitMac, nil
}

func (d *AdmitMacApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	conditions := util.GenStrConditionsFromFilters(ctx.GetFilters(), resource.SqlColumnHwAddress, resource.SqlColumnHwAddress)
	macs, err := d.Service.List(conditions)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list admit macs from db failed: %s", err.Error()))
	}
	return macs, nil
}

func (d *AdmitMacApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMacID := ctx.Resource.(*resource.AdmitMac).GetID()
	admitMac, err := d.Service.Get(admitMacID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get admit mac %s from db failed: %s", admitMacID, err.Error()))
	}
	return admitMac.(*resource.AdmitMac), nil
}

func (d *AdmitMacApi) Delete(ctx *restresource.Context) *resterror.APIError {
	admitMacId := ctx.Resource.GetID()
	err := d.Service.Delete(admitMacId)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete admit mac %s failed: %s", admitMacId, err.Error()))
	}
	return nil
}

func (d *AdmitMacApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMac := ctx.Resource.(*resource.AdmitMac)
	if err := admitMac.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update admit mac %s failed: %s", admitMac.GetID(), err.Error()))
	}
	retAdmitMac, err := d.Service.Update(admitMac)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create admit mac %s failed: %s", admitMac.GetID(), err.Error()))
	}
	return retAdmitMac, nil
}
