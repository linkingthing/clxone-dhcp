package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
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
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return template, nil
}

func (p *Pool4TemplateApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	conditions := make(map[string]interface{})
	if name, ok := util.GetFilterValueWithEqModifierFromFilters(
		util.FilterNameName, ctx.GetFilters()); ok {
		conditions[util.FilterNameName] = name
	} else {
		conditions[resource.SqlOrderBy] = resource.SqlColumnName
	}

	templates, err := p.Service.List(conditions)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return templates, nil
}

func (p *Pool4TemplateApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template, err := p.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return template, nil
}

func (p *Pool4TemplateApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool4Template)
	if err := p.Service.Update(template); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return template, nil
}

func (p *Pool4TemplateApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := p.Service.Delete(ctx.Resource.GetID()); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}
