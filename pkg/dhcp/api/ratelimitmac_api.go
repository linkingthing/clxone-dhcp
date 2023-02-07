package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type RateLimitMacApi struct {
	Service *service.RateLimitMacService
}

func NewRateLimitMacApi() *RateLimitMacApi {
	return &RateLimitMacApi{Service: service.NewRateLimitMacService()}
}

func (d *RateLimitMacApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rateLimitMac := ctx.Resource.(*resource.RateLimitMac)
	if err := d.Service.Create(rateLimitMac); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimitMac, nil
}

func (d *RateLimitMacApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	macs, err := d.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		resource.SqlColumnHwAddress, resource.SqlColumnHwAddress))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return macs, nil
}

func (d *RateLimitMacApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rateLimitMac, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimitMac, nil
}

func (d *RateLimitMacApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := d.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (d *RateLimitMacApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rateLimitMac := ctx.Resource.(*resource.RateLimitMac)
	if err := d.Service.Update(rateLimitMac); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimitMac, nil
}
