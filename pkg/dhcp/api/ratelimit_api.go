package api

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

type RateLimitApi struct {
	Service *service.RateLimitService
}

func NewRateLimitApi() *RateLimitApi {
	return &RateLimitApi{Service: service.NewRateLimitService()}
}

func (d *RateLimitApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if ratelimits, err := d.Service.List(); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp ratelimit failed: %s", err.Error()))
	} else {
		return ratelimits, nil
	}
}

func (d *RateLimitApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitID := ctx.Resource.(*resource.RateLimit).GetID()
	ratelimit, err := d.Service.Get(ratelimitID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get ratelimit failed: %s", err.Error()))
	}

	return ratelimit.(*resource.RateLimit), nil
}

func (d *RateLimitApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimit := ctx.Resource.(*resource.RateLimit)
	retratelimit, err := d.Service.Update(ratelimit)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update dhcp ratelimit failed: %s", err.Error()))
	}

	return retratelimit, nil
}
