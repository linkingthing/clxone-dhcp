package api

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Pool6TemplateHandler struct {
}

func NewPool6TemplateHandler() *Pool6TemplateHandler {
	return &Pool6TemplateHandler{}
}

func (p *Pool6TemplateHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool6Template)
	if err := template.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pool6 template %s params invalid: %s",
				template.Name, err.Error()))
	}

	template.SetID(template.Name)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(template)
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool6 template %s failed: %s",
				template.Name, err.Error()))
	}

	return template, nil
}

func (p *Pool6TemplateHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	conditions := make(map[string]interface{})
	if name, ok := util.GetFilterValueWithEqModifierFromFilters(
		util.FilterNameName, ctx.GetFilters()); ok {
		conditions[util.FilterNameName] = name
	} else {
		conditions["orderby"] = "name"
	}

	var templates []*resource.Pool6Template
	if err := db.GetResources(conditions, &templates); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pool6 templates from db failed: %s",
				err.Error()))
	}

	return templates, nil
}

func (p *Pool6TemplateHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	templateID := ctx.Resource.GetID()
	var templates []*resource.Pool6Template
	template, err := restdb.GetResourceWithID(db.GetDB(), templateID, &templates)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool6 template %s from db failed: %s",
				templateID, err.Error()))
	}

	return template.(*resource.Pool6Template), nil
}

func (p *Pool6TemplateHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.Pool6Template)
	if err := template.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update pool6 template %s params invalid: %s",
				template.Name, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePool6Template, map[string]interface{}{
			"begin_offset": template.BeginOffset,
			"capacity":     template.Capacity,
			"comment":      template.Comment,
		}, map[string]interface{}{restdb.IDField: template.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found pool6 template %s", template.GetID())
		} else {
			return nil
		}
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update pool6 template %s failed: %s",
				template.Name, err.Error()))
	}

	return template, nil
}

func (p *Pool6TemplateHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	templateID := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TablePool6Template,
			map[string]interface{}{restdb.IDField: templateID}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found pool6 template %s", templateID)
		} else {
			return nil
		}
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool6 template %s failed: %s",
				templateID, err.Error()))
	}

	return nil
}
