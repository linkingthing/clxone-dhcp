package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type PdPoolApi struct {
	Service *service.PdPoolService
}

func NewPdPoolApi() *PdPoolApi {
	return &PdPoolApi{Service: service.NewPdPoolService()}
}

func (p *PdPoolApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pdPool := ctx.Resource.(*resource.PdPool)

	if err := p.Service.Create(subnet, pdPool); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pdPool, nil
}

func (p *PdPoolApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	pdPools, err := service.ListPdPools(ctx.Resource.GetParent().GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pdPools, nil
}

func (p *PdPoolApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pdPool, err := p.Service.Get(ctx.Resource.GetParent().GetID(), ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pdPool, nil
}

func (p *PdPoolApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := p.Service.Delete(
		ctx.Resource.GetParent().(*resource.Subnet6),
		ctx.Resource.(*resource.PdPool)); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}

func (p *PdPoolApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pdPool := ctx.Resource.(*resource.PdPool)
	if err := p.Service.Update(ctx.Resource.GetParent().GetID(), pdPool); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pdPool, nil
}
