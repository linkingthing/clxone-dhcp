package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type PdPoolApi struct {
	Service *service.PdPoolService
}

func NewPdPoolApi() *PdPoolApi {
	return &PdPoolApi{Service: service.NewPdPoolService()}
}

func (p *PdPoolApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pdPool := ctx.Resource.(*resource.PdPool)
	if err := p.Service.Create(ctx.Resource.GetParent().(*resource.Subnet6), pdPool); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pdPool, nil
}

func (p *PdPoolApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	pdPools, err := p.Service.List(ctx.Resource.GetParent().(*resource.Subnet6))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pdPools, nil
}

func (p *PdPoolApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pdPool, err := p.Service.Get(
		ctx.Resource.GetParent().(*resource.Subnet6),
		ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pdPool, nil
}

func (p *PdPoolApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := p.Service.Delete(
		ctx.Resource.GetParent().(*resource.Subnet6),
		ctx.Resource.(*resource.PdPool)); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (p *PdPoolApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pdPool := ctx.Resource.(*resource.PdPool)
	if err := p.Service.Update(ctx.Resource.GetParent().GetID(), pdPool); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pdPool, nil
}
