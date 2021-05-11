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

type PoolTemplateHandler struct {
}

func NewPoolTemplateHandler() *PoolTemplateHandler {
	return &PoolTemplateHandler{}
}

func (p *PoolTemplateHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.PoolTemplate)
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

func (p *PoolTemplateHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	conditions := map[string]interface{}{"orderby": "name"}
	if version, ok := util.IPVersionFromFilter(ctx.GetFilters()); ok == false {
		return nil, nil
	} else {
		conditions[util.FilterNameVersion] = version
	}

	if name, ok := util.GetFilterValueWithEqModifierFromFilters(util.FilterNameName, ctx.GetFilters()); ok {
		conditions[util.FilterNameName] = name
	}

	var templates []*resource.PoolTemplate
	if err := db.GetResources(conditions, &templates); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pool templates from db failed: %s", err.Error()))
	}

	return templates, nil
}

func (p *PoolTemplateHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	templateID := ctx.Resource.GetID()
	var templates []*resource.PoolTemplate
	template, err := restdb.GetResourceWithID(db.GetDB(), templateID, &templates)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool template %s from db failed: %s", templateID, err.Error()))
	}

	return template.(*resource.PoolTemplate), nil
}

func (p *PoolTemplateHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	template := ctx.Resource.(*resource.PoolTemplate)
	if err := template.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update pool template %s params invalid: %s", template.Name, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(resource.TablePoolTemplate, map[string]interface{}{
			"begin_offset":   template.BeginOffset,
			"capacity":       template.Capacity,
			"domain_servers": template.DomainServers,
			"routers":        template.Routers,
			"client_class":   template.ClientClass,
			"comment":        template.Comment,
		}, map[string]interface{}{restdb.IDField: template.GetID()})
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update pool template %s failed: %s", template.Name, err.Error()))
	}

	return template, nil
}

func (p *PoolTemplateHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	templateID := ctx.Resource.GetID()
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Delete(resource.TablePoolTemplate, map[string]interface{}{restdb.IDField: templateID})
		return err
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool template %s failed: %s", templateID, err.Error()))
	}

	return nil
}
