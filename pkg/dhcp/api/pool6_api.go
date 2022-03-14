package api

import (
	"fmt"

	"github.com/linkingthing/cement/log"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type Pool6Api struct {
	Service *service.Pool6Service
}

func NewPool6Api() *Pool6Api {
	return &Pool6Api{Service: service.NewPool6Service()}
}

func (p *Pool6Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.Pool6)
	if err := pool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pool params invalid: %s", err.Error()))
	}
	retpool, err := p.Service.Create(subnet, pool)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return retpool, nil
}

func (p *Pool6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pools, err := service.ListPool6s(subnet)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pools with subnet %s from db failed: %s",
				subnet.GetID(), err.Error()))
	}
	return pools, nil
}

func (p *Pool6Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	poolID := ctx.Resource.GetID()
	pools, err := p.Service.Get(subnetID, poolID)
	if err != nil {
		log.Warnf("get pool %s with subnet %s failed: %s",
			poolID, subnetID, err.Error())
	}
	return pools, nil
}

func (p *Pool6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.Pool6)
	err := p.Service.Delete(subnet, pool)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func (p *Pool6Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.Pool6)
	retPool, err := p.Service.Update(pool)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update pool6 %s with subnet %s failed: %s",
				pool.String(), ctx.Resource.GetParent().GetID(), err.Error()))
	}

	return retPool, nil
}

func (p *Pool6Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameValidTemplate:
		return p.actionCalidTemplate(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (p *Pool6Api) actionCalidTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	templatePool, err := p.Service.ActionValidTemplate(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("template %s invalid: %s", ctx.Resource.(*resource.Pool6).Template, err.Error()))
	}

	return templatePool, nil
}
