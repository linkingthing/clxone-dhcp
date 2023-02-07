package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type RateLimitDuidHandler struct {
	Service *service.RateLimitDuidService
}

func NewRateLimitDuidApi() *RateLimitDuidHandler {
	return &RateLimitDuidHandler{Service: service.NewRateLimitDuidService()}
}

func (d *RateLimitDuidHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rateLimitDuid := ctx.Resource.(*resource.RateLimitDuid)
	if err := d.Service.Create(rateLimitDuid); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimitDuid, nil
}

func (d *RateLimitDuidHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	rateLimitDuids, err := d.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		resource.SqlColumnDuid, resource.SqlColumnDuid))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimitDuids, nil
}

func (d *RateLimitDuidHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rateLimitDuid, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimitDuid, nil
}

func (d *RateLimitDuidHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := d.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (d *RateLimitDuidHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rateLimitDuid := ctx.Resource.(*resource.RateLimitDuid)
	if err := d.Service.Update(rateLimitDuid); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimitDuid, nil
}
