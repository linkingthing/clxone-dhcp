package service

import (
	"fmt"

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

func (p *Pool4TemplateService) Create(template *resource.Pool4Template) error {
	if err := template.Validate(); err != nil {
		return fmt.Errorf("validate pool template %s params invalid: %s",
			template.Name, err.Error())
	}

	template.SetID(template.Name)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(template)
		return err
	}); err != nil {
		return fmt.Errorf("create pool template %s failed:%s",
			template.Name, err.Error())
	}

	return nil
}

func (p *Pool4TemplateService) List(ctx *restresource.Context) ([]*resource.Pool4Template, error) {
	conditions := make(map[string]interface{})
	if name, ok := util.GetFilterValueWithEqModifierFromFilters(util.FilterNameName,
		ctx.GetFilters()); ok {
		conditions[util.FilterNameName] = name
	} else {
		conditions[util.SqlOrderBy] = util.SqlColumnsName
	}

	var templates []*resource.Pool4Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &templates)
	}); err != nil {
		return nil, fmt.Errorf("list pool template failed:%s", err.Error())
	}

	return templates, nil
}

func (p *Pool4TemplateService) Get(id string) (*resource.Pool4Template, error) {
	var templates []*resource.Pool4Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &templates)
	}); err != nil {
		return nil, fmt.Errorf("get pool template %s failed:%s", id, err.Error())
	} else if len(templates) == 0 {
		return nil, fmt.Errorf("no found pool template %s", id)
	}

	return templates[0], nil
}

func (p *Pool4TemplateService) Update(template *resource.Pool4Template) error {
	if err := template.Validate(); err != nil {
		return fmt.Errorf("validate pool template %s params invalid: %s",
			template.Name, err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePool4Template, map[string]interface{}{
			resource.SqlColumnBeginOffset: template.BeginOffset,
			resource.SqlColumnCapacity:    template.Capacity,
			util.SqlColumnsComment:        template.Comment,
		}, map[string]interface{}{restdb.IDField: template.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found pool4 template %s", template.GetID())
		} else {
			return nil
		}
	}); err != nil {
		return fmt.Errorf("update pool template %s failed:%s",
			template.Name, err.Error())
	}

	return nil
}

func (p *Pool4TemplateService) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TablePool4Template, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found pool4 template %s", id)
		} else {
			return nil
		}
	}); err != nil {
		return fmt.Errorf("delete pool template %s failed:%s", id, err.Error())
	}

	return nil
}
