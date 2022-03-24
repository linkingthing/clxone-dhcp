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
	pool := ctx.Resource.(*resource.ReservedPool6)
	if err := p.Service.Create(ctx.Resource.GetParent().(*resource.Subnet6), pool); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pool, nil
}

func (p *ReservedPool6Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	pools, err := p.Service.List(ctx.Resource.GetParent().GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pools, nil
}

func (p *ReservedPool6Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool, err := p.Service.Get(ctx.Resource.GetParent().(*resource.Subnet6), ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pool, nil
}

func (p *ReservedPool6Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := p.Service.Delete(
		ctx.Resource.GetParent().(*resource.Subnet6),
		ctx.Resource.(*resource.ReservedPool6)); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
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
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("parse action valid template input invalid"))
	}

	ret, err := p.Service.ActionValidTemplate(
		ctx.Resource.GetParent().(*resource.Subnet6),
		ctx.Resource.(*resource.ReservedPool6),
		templateInfo)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return ret, nil
}

func (p *ReservedPool6Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.ReservedPool6)
	if err := p.Service.Update(ctx.Resource.GetParent().GetID(), pool); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pool, nil
}
