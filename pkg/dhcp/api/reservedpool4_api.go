package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type ReservedPool4Api struct {
	Service *service.ReservedPool4Service
}

func NewReservedPool4Api() *ReservedPool4Api {
	return &ReservedPool4Api{Service: service.NewReservedPool4Service()}
}

func (p *ReservedPool4Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.ReservedPool4)
	if err := pool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create reserved pool params invalid: %s", err.Error()))
	}
	retpool, err := p.Service.Create(subnet, pool)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return retpool, nil
}

func (p *ReservedPool4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pools, err := service.ListReservedPool4s(subnetID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list reserved pools with subnet %s failed: %s",
				subnetID, err.Error()))
	}

	return pools, nil
}

func (p *ReservedPool4Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
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

func (p *ReservedPool4Api) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.ReservedPool4)
	if err := p.Service.Delete(subnet, pool); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func (p *ReservedPool4Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameValidTemplate:
		return p.actionValidTemplate(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (p *ReservedPool4Api) actionValidTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.ReservedPool4)
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("parse action valid pool template input invalid"))
	}
	ret, err := p.Service.ActionValidTemplate(subnet, pool, templateInfo)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("exec actionvalidtemplate failed :%s", err.Error()))
	}
	return ret, nil
}

func (p *ReservedPool4Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.ReservedPool4)
	retPool, err := p.Service.Update(pool)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update reserved pool4 %s with subnet %s failed: %s",
				pool.String(), ctx.Resource.GetParent().GetID(), err.Error()))
	}

	return retPool, nil
}
