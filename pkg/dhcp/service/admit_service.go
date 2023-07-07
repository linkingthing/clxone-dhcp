package service

import (
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type AdmitService struct{}

func NewAdmitService() (*AdmitService, error) {
	if err := createDefaultAdmit(); err != nil {
		return nil, err
	}

	return &AdmitService{}, nil
}

func createDefaultAdmit() error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(resource.TableAdmit, nil); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAdmit), pg.Error(err).Error())
		} else if !exists {
			if _, err := tx.Insert(resource.DefaultAdmit); err != nil {
				return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameAdmit), pg.Error(err).Error())
			}
		}

		return nil
	})
}

func (d *AdmitService) List() ([]*resource.Admit, error) {
	var admits []*resource.Admit
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &admits)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAdmit), pg.Error(err).Error())
	}

	return admits, nil
}

func (d *AdmitService) Get(id string) (*resource.Admit, error) {
	var admits []*resource.Admit
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &admits)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(admits) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameAdmit, id)
	}

	return admits[0], nil
}

func (d *AdmitService) Update(admit *resource.Admit) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableAdmit,
			map[string]interface{}{resource.SqlColumnEnabled: admit.Enabled},
			map[string]interface{}{restdb.IDField: admit.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, admit.GetID(), pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAdmit, admit.GetID())
		}

		return sendUpdateAdmitCmdToDHCPAgent(admit)
	})
}

func sendUpdateAdmitCmdToDHCPAgent(admit *resource.Admit) error {
	return kafka.SendDHCPCmd(kafka.UpdateAdmit,
		&pbdhcpagent.UpdateAdmitRequest{Enabled: admit.Enabled}, nil)
}
