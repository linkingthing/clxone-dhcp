package service

import (
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Pool4TemplateService struct {
}

func NewPool4TemplateService() *Pool4TemplateService {
	return &Pool4TemplateService{}
}

func (p *Pool4TemplateService) Create(template *resource.Pool4Template) error {
	if err := template.Validate(); err != nil {
		return err
	}

	template.SetID(template.Name)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Insert(template)
		return err
	}); err != nil {
		return util.FormatDbInsertError(errorno.ErrNameTemplate, template.Name, err)
	}

	return nil
}

func (p *Pool4TemplateService) List(conditions map[string]interface{}) ([]*resource.Pool4Template, error) {
	var templates []*resource.Pool4Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &templates)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameTemplate), pg.Error(err).Error())
	}

	return templates, nil
}

func (p *Pool4TemplateService) Get(id string) (*resource.Pool4Template, error) {
	var templates []*resource.Pool4Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &templates)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(templates) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameTemplate, id)
	}

	return templates[0], nil
}

func (p *Pool4TemplateService) Update(template *resource.Pool4Template) error {
	if err := template.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePool4Template, map[string]interface{}{
			resource.SqlColumnBeginOffset: template.BeginOffset,
			resource.SqlColumnCapacity:    template.Capacity,
			resource.SqlColumnComment:     template.Comment,
		}, map[string]interface{}{restdb.IDField: template.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, template.Name, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameTemplate, template.GetID())
		} else {
			return nil
		}
	})
}

func (p *Pool4TemplateService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TablePool4Template, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameTemplate, id)
		} else {
			return nil
		}
	})
}
