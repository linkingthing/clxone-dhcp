package service

import (
	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type AdmitDuidService struct{}

func NewAdmitDuidService() *AdmitDuidService {
	return &AdmitDuidService{}
}

func (d *AdmitDuidService) Create(admitDuid *resource.AdmitDuid) error {
	admitDuid.SetID(admitDuid.Duid)
	if err := admitDuid.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitDuid); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameDuid, admitDuid.Duid, err)
		}

		return sendCreateAdmitDuidCmdToDHCPAgent(admitDuid)
	})
}

func sendCreateAdmitDuidCmdToDHCPAgent(admitDuid *resource.AdmitDuid) error {
	return kafka.SendDHCPCmd(kafka.CreateAdmitDuid,
		&pbdhcpagent.CreateAdmitDuidRequest{Duid: admitDuid.Duid},
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteAdmitDuid,
				&pbdhcpagent.DeleteAdmitDuidRequest{Duid: admitDuid.Duid}); err != nil {
				log.Errorf("create admit duid %s failed, rollback with nodes %v failed: %s",
					admitDuid.Duid, nodesForSucceed, err.Error())
			}
		})
}

func (d *AdmitDuidService) List(conditions map[string]interface{}) ([]*resource.AdmitDuid, error) {
	var duids []*resource.AdmitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &duids)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAdmit), pg.Error(err).Error())
	}

	return duids, nil
}

func (d *AdmitDuidService) Get(id string) (*resource.AdmitDuid, error) {
	var admitDuids []*resource.AdmitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &admitDuids)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(admitDuids) != 1 {
		return nil, errorno.ErrNotFound(errorno.ErrNameAdmit, id)
	}

	return admitDuids[0], nil
}

func (d *AdmitDuidService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitDuid,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAdmit, id)
		}

		return sendDeleteAdmitDuidCmdToDHCPAgent(id)
	})
}

func sendDeleteAdmitDuidCmdToDHCPAgent(admitDuidId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAdmitDuid,
		&pbdhcpagent.DeleteAdmitDuidRequest{Duid: admitDuidId}, nil)
}

func (d *AdmitDuidService) Update(admitDuid *resource.AdmitDuid) error {
	if err := admitDuid.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableAdmitDuid,
			map[string]interface{}{resource.SqlColumnComment: admitDuid.Comment},
			map[string]interface{}{restdb.IDField: admitDuid.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, admitDuid.GetID(), pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAdmit, admitDuid.GetID())
		}

		return nil
	})
}
