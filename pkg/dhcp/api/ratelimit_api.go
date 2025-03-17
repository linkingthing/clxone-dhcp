package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type RateLimitApi struct {
	Service *service.RateLimitService
}

func NewRateLimitApi() (*RateLimitApi, error) {
	s, err := service.NewRateLimitService()
	if err != nil {
		return nil, err
	}

	return &RateLimitApi{Service: s}, nil
}

func (r *RateLimitApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	rateLimits, err := r.Service.List()
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimits, nil
}

func (r *RateLimitApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rateLimit, err := r.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimit, nil
}

func (r *RateLimitApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rateLimit := ctx.Resource.(*resource.RateLimit)
	if err := r.Service.Update(rateLimit); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimit, nil
}
