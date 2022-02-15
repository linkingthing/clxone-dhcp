package api

import (
	"fmt"

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
	if err := pdpool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pdpool params invalid: %s", err.Error()))
	}
	retpdpool, err := p.Service.Create(subnet, pdpool)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pdpool %s-%d with subnet %s failed: %s",
				pdpool.String(), pdpool.DelegatedLen, subnet.GetID(), err.Error()))
	}

	return retpdpool, nil
}

func (p *ReservedPdPoolApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pdpools, err := p.Service.List(subnetID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pdpools with subnet %s failed: %s",
				subnetID, err.Error()))
	}

	return pdpools, nil
}

func (p *ReservedPdPoolApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pdpoolID := ctx.Resource.GetID()
	pdpool, err := p.Service.Get(subnetID, pdpoolID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pdpool %s with subnet %s failed: %s",
				pdpoolID, subnetID, err.Error()))
	}

	return pdpool, nil
}

func (p *ReservedPdPoolApi) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pdpool := ctx.Resource.(*resource.ReservedPdPool)
	if err := p.Service.Delete(subnet, pdpool); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pdpool %s with subnet %s failed: %s",
				pdpool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func (p *ReservedPdPoolApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.ReservedPdPool)
	retPool, err := p.Service.Update(pool)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update reserved pdpool %s with subnet %s failed: %s",
				pool.String(), ctx.Resource.GetParent().GetID(), err.Error()))
	}

	return retPool, nil
}
