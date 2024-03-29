package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Pool4TemplateApi struct {
	Service *service.Pool4TemplateService
}

func NewPool4TemplateApi() *Pool4TemplateApi {
	return &Pool4TemplateApi{Service: service.NewPool4TemplateService()}
}

func (p *Pool4TemplateApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool4Template)
	if err := p.Service.Create(template); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return template, nil
}

func (p *Pool4TemplateApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	templates, err := p.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		resource.SqlColumnName, resource.SqlColumnName))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return templates, nil
}

func (p *Pool4TemplateApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template, err := p.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return template, nil
}

func (p *Pool4TemplateApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool4Template)
	if err := p.Service.Update(template); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return template, nil
}

func (p *Pool4TemplateApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := p.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}
