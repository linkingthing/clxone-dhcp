package api

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

type Pool6TemplateApi struct {
	Service *service.Pool6TemplateService
}

func NewPool6TemplateApi() *Pool6TemplateApi {
	return &Pool6TemplateApi{Service: service.NewPool6TemplateService()}
}

func (p *Pool6TemplateApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool6Template)
	if err := template.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pool template %s params invalid: %s",
				template.Name, err.Error()))
	}

	template.SetID(template.Name)
	if rettemplate, err := p.Service.Create(template); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool template %s failed: %s",
				template.Name, err.Error()))
	} else {
		return rettemplate, nil
	}
}

func (p *Pool6TemplateApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if templates, err := p.Service.List(ctx); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pool templates from db failed: %s",
				err.Error()))
	} else {
		return templates, nil
	}
}

func (p *Pool6TemplateApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	templateID := ctx.Resource.GetID()
	if templates, err := p.Service.Get(templateID); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool template %s from db failed: %s",
				templateID, err.Error()))
	} else {
		return templates, nil
	}
}

func (p *Pool6TemplateApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool6Template)
	if err := template.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update pool template %s params invalid: %s",
				template.Name, err.Error()))
	}
	rettemplate, err := p.Service.Update(template)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update pool template %s failed: %s",
				template.Name, err.Error()))
	}

	return rettemplate, nil
}

func (p *Pool6TemplateApi) Delete(ctx *restresource.Context) *resterror.APIError {
	templateID := ctx.Resource.GetID()
	if err := p.Service.Delete(templateID); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool template %s failed: %s",
				templateID, err.Error()))
	}
	return nil
}
