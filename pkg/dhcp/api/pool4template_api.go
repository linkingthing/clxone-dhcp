package api

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

type Pool4TemplateApi struct {
	Service *service.Pool4TemplateService
}

func NewPool4TemplateApi() *Pool4TemplateApi {
	return &Pool4TemplateApi{Service: service.NewPool4TemplateService()}
}

func (p *Pool4TemplateApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool4Template)
	if err := template.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pool template %s params invalid: %s",
				template.Name, err.Error()))
	}
	template.SetID(template.Name)
	retTemplate, err := p.Service.Create(template)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool template %s failed: %s",
				template.Name, err.Error()))
	}

	return retTemplate, nil
}

func (p *Pool4TemplateApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {

	templates, err := p.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pool templates from db failed: %s", err.Error()))
	}

	return templates, nil
}

func (p *Pool4TemplateApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	templateID := ctx.Resource.GetID()
	template, err := p.Service.Get(templateID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool template %s from db failed: %s",
				templateID, err.Error()))
	}

	return template.(*resource.Pool4Template), nil
}

func (p *Pool4TemplateApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool4Template)
	if err := template.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update pool template %s params invalid: %s",
				template.Name, err.Error()))
	}
	retTemplate, err := p.Service.Update(template)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update pool template %s failed: %s",
				template.Name, err.Error()))
	}

	return retTemplate, nil
}

func (p *Pool4TemplateApi) Delete(ctx *restresource.Context) *resterror.APIError {
	templateID := ctx.Resource.GetID()
	if err := p.Service.Delete(templateID); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool template %s failed: %s",
				templateID, err.Error()))
	}

	return nil
}
