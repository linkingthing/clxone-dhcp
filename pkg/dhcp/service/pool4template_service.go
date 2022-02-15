package service

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Pool4TemplateService struct {
}

func NewPool4TemplateService() *Pool4TemplateService {
	return &Pool4TemplateService{}
}

func (p *Pool4TemplateService) Create(template *resource.Pool4Template) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(template)
		return err
	}); err != nil {
		return nil, err
	}

	return template, nil
}

func (p *Pool4TemplateService) List(ctx *restresource.Context) (interface{}, error) {
	conditions := make(map[string]interface{})
	if name, ok := util.GetFilterValueWithEqModifierFromFilters(util.FilterNameName,
		ctx.GetFilters()); ok {
		conditions[util.FilterNameName] = name
	} else {
		conditions[util.SqlOrderBy] = "name"
	}

	var templates []*resource.Pool4Template
	if err := db.GetResources(conditions, &templates); err != nil {
		return nil, err
	}

	return templates, nil
}

func (p *Pool4TemplateService) Get(templateID string) (restresource.Resource, error) {
	var templates []*resource.Pool4Template
	template, err := restdb.GetResourceWithID(db.GetDB(), templateID, &templates)
	if err != nil {
		return nil, err
	}

	return template.(*resource.Pool4Template), nil
}

func (p *Pool4TemplateService) Update(template *resource.Pool4Template) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(resource.TablePool4Template, map[string]interface{}{
			resource.SqlColumnBeginOffset: template.BeginOffset,
			resource.SqlColumnCapacity:    template.Capacity,
			util.SqlColumnsComment:        template.Comment,
		}, map[string]interface{}{restdb.IDField: template.GetID()})
		return err
	}); err != nil {
		return nil, err
	}

	return template, nil
}

func (p *Pool4TemplateService) Delete(templateID string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Delete(resource.TablePool4Template, map[string]interface{}{
			restdb.IDField: templateID})
		return err
	}); err != nil {
		return err
	}

	return nil
}
