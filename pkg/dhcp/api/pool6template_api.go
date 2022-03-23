package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type Pool6TemplateApi struct {
	Service *service.Pool6TemplateService
}

func NewPool6TemplateApi() *Pool6TemplateApi {
	return &Pool6TemplateApi{Service: service.NewPool6TemplateService()}
}

func (p *Pool6TemplateApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool6Template)
	if err := p.Service.Create(template); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return template, nil
}

func (p *Pool6TemplateApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	templates, err := p.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return templates, nil
}

func (p *Pool6TemplateApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template, err := p.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return template, nil
}

func (p *Pool6TemplateApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool6Template)
	if err := p.Service.Update(template); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return template, nil
}

func (p *Pool6TemplateApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := p.Service.Delete(ctx.Resource.GetID()); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}
