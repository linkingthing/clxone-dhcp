package api

import (
	"fmt"

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
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := pdpool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pdpool params invalid: %s", err.Error()))
	}
	retPdpool, err := p.Service.Create(subnet, pdpool)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pdpool %s with subnet %s failed: %s",
				pdpool.String(), subnet.GetID(), err.Error()))
	}

	return retPdpool, nil
}

func (p *PdPoolApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pdpools, err := service.ListPdPools(subnetID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pdpools with subnet %s from db failed: %s",
				subnetID, err.Error()))
	}

	return pdpools, nil
}

func (p *PdPoolApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pdpoolID := ctx.Resource.GetID()
	pdpool, err := p.Service.Get(subnetID, pdpoolID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pdpool %s with subnet %s from db failed: %s",
				pdpoolID, subnetID, err.Error()))
	}

	return pdpool.(*resource.PdPool), nil
}

func (p *PdPoolApi) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pdpool := ctx.Resource.(*resource.PdPool)
	err := p.Service.Delete(subnet, pdpool)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pdpool %s with subnet %s failed: %s",
				pdpool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func (p *PdPoolApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := pdpool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pdpool params invalid: %s", err.Error()))
	}
	retpdpool, err := p.Service.Update(pdpool)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update pdpool %s with subnet %s failed: %s",
				pdpool.String(), ctx.Resource.GetParent().GetID(), err.Error()))
	}

	return retpdpool, nil
}
