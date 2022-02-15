package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type RateLimitMacApi struct {
	Service *service.RateLimitMacService
}

func NewRateLimitMacApi() *RateLimitMacApi {
	return &RateLimitMacApi{Service: service.NewRateLimitMacService()}
}

func (d *RateLimitMacApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitMac := ctx.Resource.(*resource.RateLimitMac)
	ratelimitMac.SetID(ratelimitMac.HwAddress)
	if err := ratelimitMac.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create ratelimit mac %s failed: %s", ratelimitMac.GetID(), err.Error()))
	}
	retratelimitMac, err := d.Service.Create(ratelimitMac)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create ratelimit mac %s failed: %s", ratelimitMac.GetID(), err.Error()))
	}

	return retratelimitMac, nil
}

func (d *RateLimitMacApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	macs, err := d.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list ratelimit macs failed: %s", err.Error()))
	}

	return macs, nil
}

func (d *RateLimitMacApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitMacID := ctx.Resource.(*resource.RateLimitMac).GetID()
	ratelimitMac, err := d.Service.Get(ratelimitMacID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get ratelimit mac %s failed: %s", ratelimitMacID, err.Error()))
	}

	return ratelimitMac.(*resource.RateLimitMac), nil
}

func (d *RateLimitMacApi) Delete(ctx *restresource.Context) *resterror.APIError {
	ratelimitMacId := ctx.Resource.GetID()
	if err := d.Service.Delete(ratelimitMacId); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete ratelimit mac %s failed: %s", ratelimitMacId, err.Error()))
	}

	return nil
}

func (d *RateLimitMacApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitMac := ctx.Resource.(*resource.RateLimitMac)
	retratelimitMac, err := d.Service.Update(ratelimitMac)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update ratelimit mac %s failed: %s", ratelimitMac.GetID(), err.Error()))
	}

	return retratelimitMac, nil
}
