package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type ReservedPool4Api struct {
	Service *service.ReservedPool4Service
}

func NewReservedPool4Api() *ReservedPool4Api {
	return &ReservedPool4Api{Service: service.NewReservedPool4Service()}
}

func (p *ReservedPool4Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.ReservedPool4)
	if err := p.Service.Create(ctx.Resource.GetParent().(*resource.Subnet4), pool); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pool, nil
}

func (p *ReservedPool4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	pools, err := p.Service.List(ctx.Resource.GetParent().GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pools, nil
}

func (p *ReservedPool4Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool, err := p.Service.Get(ctx.Resource.GetParent().(*resource.Subnet4), ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pool, nil
}

func (p *ReservedPool4Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := p.Service.Delete(
		ctx.Resource.GetParent().(*resource.Subnet4),
		ctx.Resource.(*resource.ReservedPool4)); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (p *ReservedPool4Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.ReservedPool4)
	if err := p.Service.Update(ctx.Resource.GetParent().GetID(), pool); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pool, nil
}

func (p *ReservedPool4Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameValidTemplate:
		return p.actionValidTemplate(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameDhcpReservedPool, ctx.Resource.GetAction().Name))
	}
}

func (p *ReservedPool4Api) actionValidTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameDhcpReservedPool, resource.ActionNameValidTemplate))
	}

	ret, err := p.Service.ActionValidTemplate(ctx.Resource.GetParent().(*resource.Subnet4),
		ctx.Resource.(*resource.ReservedPool4), templateInfo)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return ret, nil
}
