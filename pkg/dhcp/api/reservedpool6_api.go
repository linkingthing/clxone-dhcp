package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type ReservedPool6Handler struct {
	Service *service.ReservedPool6Service
}

func NewReservedPool6Api() *ReservedPool6Handler {
	return &ReservedPool6Handler{Service: service.NewReservedPool6Service()}
}

func (p *ReservedPool6Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.ReservedPool6)
	if err := pool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pool params invalid: %s", err.Error()))
	}
	retPool, err := p.Service.Create(subnet, pool)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return retPool, nil
}

func (p *ReservedPool6Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pools, err := service.ListReservedPool6s(subnet)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pools with subnet %s from db failed: %s",
				subnet.GetID(), err.Error()))
	}

	return pools, nil
}

func (p *ReservedPool6Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	poolID := ctx.Resource.GetID()
	pool, err := p.Service.Get(subnetID, poolID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool %s with subnet %s failed: %s",
				poolID, subnetID, err.Error()))
	}
	return pool, nil
}

func (p *ReservedPool6Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.ReservedPool6)
	if err := p.Service.Delete(subnet, pool); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func (p *ReservedPool6Handler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameValidTemplate:
		return p.actionValidTemplate(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (p *ReservedPool6Handler) actionValidTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.ReservedPool6)
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("parse action refresh input invalid"))
	}
	ret, err := p.Service.ActionValidTemplate(subnet, pool, templateInfo)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("exec template %s failed: %s", pool.Template, err.Error()))
	}

	return ret, nil
}

func (p *ReservedPool6Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.ReservedPool6)
	retPool, err := p.Service.Update(pool)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update reserved pool6 %s with subnet %s failed: %s",
				pool.String(), ctx.Resource.GetParent().GetID(), err.Error()))
	}
	return retPool, nil
}
