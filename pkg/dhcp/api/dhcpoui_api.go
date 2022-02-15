package api

import (
	"fmt"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

type DhcpOuiHandler struct {
	Service *service.DhcpOuiService
}

func NewDhcpOuiApi() *DhcpOuiHandler {
	return &DhcpOuiHandler{Service: service.NewDhcpOuiService()}
}

func (d *DhcpOuiHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpoui := ctx.Resource.(*resource.DhcpOui)
	if err := dhcpoui.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create dhcp oui %s failed: %s", dhcpoui.Oui, err.Error()))
	}
	dhcpoui.SetID(dhcpoui.Oui)
	retDhcpoui, err := d.Service.Create(dhcpoui)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create dhcp oui %s failed: %s", dhcpoui.GetID(), err.Error()))
	}
	return retDhcpoui, nil
}

func (d *DhcpOuiHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ouis, err := d.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp ouis from db failed: %s", err.Error()))
	}
	return ouis, nil
}

func (d *DhcpOuiHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpouiID := ctx.Resource.(*resource.DhcpOui).GetID()
	dhcpoui, err := d.Service.Get(dhcpouiID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get dhcp oui %s from db failed: %s", dhcpouiID, err.Error()))
	}

	return dhcpoui.(*resource.DhcpOui), nil
}

func (d *DhcpOuiHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpoui := ctx.Resource.(*resource.DhcpOui)
	if err := dhcpoui.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update dhcp oui %s failed: %s", dhcpoui.Oui, err.Error()))
	}
	retdhcpoui, err := d.Service.Update(dhcpoui)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update dhcp oui %s failed: %s", dhcpoui.GetID(), err.Error()))
	}

	return retdhcpoui, nil
}

func (d *DhcpOuiHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	dhcpouiId := ctx.Resource.GetID()
	err := d.Service.Delete(dhcpouiId)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete dhcp oui %s failed: %s", dhcpouiId, err.Error()))
	}

	return nil
}
