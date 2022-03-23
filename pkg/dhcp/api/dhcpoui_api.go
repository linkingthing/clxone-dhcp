package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type DhcpOuiApi struct {
	Service *service.DhcpOuiService
}

func NewDhcpOuiApi() *DhcpOuiApi {
	return &DhcpOuiApi{Service: service.NewDhcpOuiService()}
}

func (d *DhcpOuiApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpOui := ctx.Resource.(*resource.DhcpOui)
	if err := d.Service.Create(dhcpOui); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return dhcpOui, nil
}

func (d *DhcpOuiApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ouis, err := d.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return ouis, nil
}

func (d *DhcpOuiApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpOui, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return dhcpOui, nil
}

func (d *DhcpOuiApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpOui := ctx.Resource.(*resource.DhcpOui)
	if err := d.Service.Update(dhcpOui); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return dhcpOui, nil
}

func (d *DhcpOuiApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := d.Service.Delete(ctx.Resource.GetID()); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}
