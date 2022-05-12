package service

import (
	"fmt"

	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

type Pool6TemplateService struct {
}

func NewPool6TemplateService() *Pool6TemplateService {
	return &Pool6TemplateService{}
}

func (p *Pool6TemplateService) Create(template *resource.Pool6Template) error {
	if err := template.Validate(); err != nil {
		return fmt.Errorf("validate pool6 template %s params invalid: %s",
			template.Name, err.Error())
	}

	template.SetID(template.Name)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(template)
		return err
	}); err != nil {
		return fmt.Errorf("create pool6 template %s failed:%s",
			template.Name, pg.Error(err).Error())
	}

	return nil
}

func (p *Pool6TemplateService) List(conditions map[string]interface{}) ([]*resource.Pool6Template, error) {
	var templates []*resource.Pool6Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &templates)
	}); err != nil {
		return nil, fmt.Errorf("list pool6 template failed:%s", pg.Error(err).Error())
	}

	return templates, nil
}

func (p *Pool6TemplateService) Get(id string) (*resource.Pool6Template, error) {
	var templates []*resource.Pool6Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &templates)
	}); err != nil {
		return nil, fmt.Errorf("get pool6 template %s failed:%s", id, pg.Error(err).Error())
	} else if len(templates) == 0 {
		return nil, fmt.Errorf("no found pool6 template %s", id)
	}

	return templates[0], nil
}

func (p *Pool6TemplateService) Update(template *resource.Pool6Template) error {
	if err := template.Validate(); err != nil {
		return fmt.Errorf("validate pool6 template %s params invalid: %s",
			template.Name, err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePool6Template, map[string]interface{}{
			resource.SqlColumnBeginOffset: template.BeginOffset,
			resource.SqlColumnCapacity:    template.Capacity,
			resource.SqlColumnComment:     template.Comment,
		}, map[string]interface{}{restdb.IDField: template.GetID()}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found pool6 template %s", template.GetID())
		} else {
			return nil
		}
	}); err != nil {
		return fmt.Errorf("update pool6 template %s failed:%s",
			template.Name, err.Error())
	}

	return nil
}

func (p *Pool6TemplateService) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TablePool6Template,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found pool6 template %s", id)
		} else {
			return nil
		}
	}); err != nil {
		return fmt.Errorf("delete pool6 template %s failed:%s", id, err.Error())
	}

	return nil
}
