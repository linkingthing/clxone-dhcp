package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"

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
	pool := ctx.Resource.(*resource.Pool6)
	if err := p.Service.Create(ctx.Resource.GetParent().(*resource.Subnet6), pool); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pool, nil
}

func (p *Pool6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	pools, err := p.Service.List(ctx.Resource.GetParent().(*resource.Subnet6))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pools, nil
}

func (p *Pool6Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool, err := p.Service.Get(ctx.Resource.GetParent().(*resource.Subnet6), ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pool, nil
}

func (p *Pool6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := p.Service.Delete(
		ctx.Resource.GetParent().(*resource.Subnet6),
		ctx.Resource.(*resource.Pool6)); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (p *Pool6Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.Pool6)
	if err := p.Service.Update(ctx.Resource.GetParent().GetID(), pool); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pool, nil
}

func (p *Pool6Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameValidTemplate:
		return p.actionValidTemplate(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameDhcpPool, ctx.Resource.GetAction().Name))
	}
}

func (p *Pool6Api) actionValidTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameDhcpPool, resource.ActionNameValidTemplate))
	}

	if templatePool, err := p.Service.ActionValidTemplate(ctx.Resource.GetParent().(*resource.Subnet6),
		ctx.Resource.(*resource.Pool6), templateInfo); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return templatePool, nil
	}
}
