package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type ReservedPdPoolApi struct {
	Service *service.ReservedPdPoolService
}

func NewReservedPdPoolApi() *ReservedPdPoolApi {
	return &ReservedPdPoolApi{Service: service.NewReservedPdPoolService()}
}

func (p *ReservedPdPoolApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pdpool := ctx.Resource.(*resource.ReservedPdPool)
	if err := p.Service.Create(ctx.Resource.GetParent().(*resource.Subnet6), pdpool); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pdpool, nil
}

func (p *ReservedPdPoolApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	pdpools, err := p.Service.List(ctx.Resource.GetParent().GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pdpools, nil
}

func (p *ReservedPdPoolApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pdpool, err := p.Service.Get(ctx.Resource.GetParent().(*resource.Subnet6), ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pdpool, nil
}

func (p *ReservedPdPoolApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := p.Service.Delete(
		ctx.Resource.GetParent().(*resource.Subnet6),
		ctx.Resource.(*resource.ReservedPdPool)); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (p *ReservedPdPoolApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.ReservedPdPool)
	if err := p.Service.Update(ctx.Resource.GetParent().GetID(), pool); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pool, nil
}
