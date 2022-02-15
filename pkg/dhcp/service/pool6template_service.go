package service

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Pool6TemplateService struct {
}

func NewPool6TemplateService() *Pool6TemplateService {
	return &Pool6TemplateService{}
}

func (p *Pool6TemplateService) Create(template *resource.Pool6Template) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(template)
		return err
	}); err != nil {
		return nil, err
	}

	return template, nil
}

func (p *Pool6TemplateService) List(ctx *restresource.Context) (interface{}, error) {
	conditions := make(map[string]interface{})
	if name, ok := util.GetFilterValueWithEqModifierFromFilters(
		util.FilterNameName, ctx.GetFilters()); ok {
		conditions[util.FilterNameName] = name
	} else {
		conditions[util.SqlOrderBy] = util.SqlColumnsName
	}
	var templates []*resource.Pool6Template
	if err := db.GetResources(conditions, &templates); err != nil {
		return nil, err
	}

	return templates, nil
}

func (p *Pool6TemplateService) Get(templateID string) (restresource.Resource, error) {
	var templates []*resource.Pool6Template
	template, err := restdb.GetResourceWithID(db.GetDB(), templateID, &templates)
	if err != nil {
		return nil, err
	}

	return template.(*resource.Pool6Template), nil
}

func (p *Pool6TemplateService) Update(template *resource.Pool6Template) (restresource.Resource, error) {

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Update(resource.TablePool6Template, map[string]interface{}{
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

func (p *Pool6TemplateService) Delete(templateID string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Delete(resource.TablePool6Template,
			map[string]interface{}{restdb.IDField: templateID})
		return err
	}); err != nil {
		return err
	}

	return nil
}
