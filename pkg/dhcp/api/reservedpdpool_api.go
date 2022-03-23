package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type ReservedPdPoolApi struct {
	Service *service.ReservedPdPoolService
}

func NewReservedPdPoolApi() *ReservedPdPoolApi {
	return &ReservedPdPoolApi{Service: service.NewReservedPdPoolService()}
}

func (p *ReservedPdPoolApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pdpool := ctx.Resource.(*resource.ReservedPdPool)

	if err := p.Service.Create(subnet, pdpool); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pdpool, nil
}

func (p *ReservedPdPoolApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	pdpools, err := p.Service.List(ctx.Resource.GetParent().GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pdpools, nil
}

func (p *ReservedPdPoolApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pdpool, err := p.Service.Get(ctx.Resource.GetParent().GetID(), ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pdpool, nil
}

func (p *ReservedPdPoolApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := p.Service.Delete(
		ctx.Resource.GetParent().(*resource.Subnet6),
		ctx.Resource.(*resource.ReservedPdPool)); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}

func (p *ReservedPdPoolApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.ReservedPdPool)
	if err := p.Service.Update(ctx.Resource.GetParent().GetID(), pool); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pool, nil
}
