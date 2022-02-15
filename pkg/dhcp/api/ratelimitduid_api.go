package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type RateLimitDuidHandler struct {
	Service *service.RateLimitDuidService
}

func NewRateLimitDuidApi() *RateLimitDuidHandler {
	return &RateLimitDuidHandler{Service: service.NewRateLimitDuidService()}
}

func (d *RateLimitDuidHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitDuid := ctx.Resource.(*resource.RateLimitDuid)
	if err := ratelimitDuid.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create ratelimit duid %s failed: %s", ratelimitDuid.GetID(), err.Error()))
	}
	ratelimitDuid.SetID(ratelimitDuid.Duid)
	retratelimitDuid, err := d.Service.Create(ratelimitDuid)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create ratelimit duid %s failed: %s", ratelimitDuid.GetID(), err.Error()))
	}
	return retratelimitDuid, nil
}

func (d *RateLimitDuidHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if retduids, err := d.Service.List(ctx); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list ratelimit duids from db failed: %s", err.Error()))
	} else {
		return retduids, nil
	}
}

func (d *RateLimitDuidHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitDuidID := ctx.Resource.(*resource.RateLimitDuid).GetID()
	retratelimitDuid, err := d.Service.Get(ratelimitDuidID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get ratelimit duid %s from db failed: %s", ratelimitDuidID, err.Error()))
	}
	return retratelimitDuid, nil
}

func (d *RateLimitDuidHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	ratelimitDuidId := ctx.Resource.GetID()
	if err := d.Service.Delete(ratelimitDuidId); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete ratelimit duid %s failed: %s", ratelimitDuidId, err.Error()))
	}

	return nil
}

func (d *RateLimitDuidHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	ratelimitDuid := ctx.Resource.(*resource.RateLimitDuid)
	if err := ratelimitDuid.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update ratelimit duid %s failed: %s", ratelimitDuid.GetID(), err.Error()))
	}
	retratelimitDuid, err := d.Service.Update(ratelimitDuid)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update ratelimit duid %s failed: %s", ratelimitDuid.GetID(), err.Error()))
	}

	return retratelimitDuid, nil
}
