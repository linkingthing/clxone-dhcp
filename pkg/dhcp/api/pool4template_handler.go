package api

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Pool4TemplateHandler struct {
}

func NewPool4TemplateHandler() *Pool4TemplateHandler {
	return &Pool4TemplateHandler{}
}

func (p *Pool4TemplateHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool4Template)
	if err := template.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pool template %s params invalid: %s", template.Name, err.Error()))
	}

	template.SetID(template.Name)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(template)
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool template %s failed: %s", template.Name, err.Error()))
	}

	return template, nil
}

func (p *Pool4TemplateHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	conditions := make(map[string]interface{})
	if name, ok := util.GetFilterValueWithEqModifierFromFilters(util.FilterNameName, ctx.GetFilters()); ok {
		conditions[util.FilterNameName] = name
	} else {
		conditions["orderby"] = "name"
	}

	var templates []*resource.Pool4Template
	if err := db.GetResources(conditions, &templates); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pool templates from db failed: %s", err.Error()))
	}

	return templates, nil
}

func (p *Pool4TemplateHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	templateID := ctx.Resource.GetID()
	var templates []*resource.Pool4Template
	template, err := restdb.GetResourceWithID(db.GetDB(), templateID, &templates)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool template %s from db failed: %s", templateID, err.Error()))
	}

	return template.(*resource.Pool4Template), nil
}

func (p *Pool4TemplateHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool4Template)
	if err := template.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update pool template %s params invalid: %s", template.Name, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(resource.TablePool4Template, map[string]interface{}{
			"begin_offset": template.BeginOffset,
			"capacity":     template.Capacity,
			"comment":      template.Comment,
		}, map[string]interface{}{restdb.IDField: template.GetID()})
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update pool template %s failed: %s", template.Name, err.Error()))
	}

	return template, nil
}

func (p *Pool4TemplateHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	templateID := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Delete(resource.TablePool4Template, map[string]interface{}{restdb.IDField: templateID})
		return err
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool template %s failed: %s", templateID, err.Error()))
	}

	return nil
}
