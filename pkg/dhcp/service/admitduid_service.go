package service

import (
	"fmt"

	pg "github.com/cuityhj/gohelper/postgresql"
	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type AdmitDuidService struct{}

func NewAdmitDuidService() *AdmitDuidService {
	return &AdmitDuidService{}
}

func (d *AdmitDuidService) Create(admitDuid *resource.AdmitDuid) error {
	admitDuid.SetID(admitDuid.Duid)
	if err := admitDuid.Validate(); err != nil {
		return fmt.Errorf("validate admit duid %s failed:%s",
			admitDuid.GetID(), err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitDuid); err != nil {
			return pg.Error(err)
		}

		return sendCreateAdmitDuidCmdToDHCPAgent(admitDuid)
	}); err != nil {
		return fmt.Errorf("create admit duid %s failed: %s",
			admitDuid.GetID(), err.Error())
	}

	return nil
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
		return nil, fmt.Errorf("list admit duid failed:%s", pg.Error(err).Error())
	}

	return duids, nil
}

func (d *AdmitDuidService) Get(id string) (*resource.AdmitDuid, error) {
	var admitDuids []*resource.AdmitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &admitDuids)
	}); err != nil {
		return nil, fmt.Errorf("get admit duid of %s failed:%s", id, pg.Error(err).Error())
	} else if len(admitDuids) != 1 {
		return nil, fmt.Errorf("no found admit duid %s", id)
	}

	return admitDuids[0], nil
}

func (d *AdmitDuidService) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitDuid,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found admit duid %s", id)
		}

		return sendDeleteAdmitDuidCmdToDHCPAgent(id)
	}); err != nil {
		return fmt.Errorf("delete admit duid %s failed:%s", id, err.Error())
	}

	return nil
}

func sendDeleteAdmitDuidCmdToDHCPAgent(admitDuidId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAdmitDuid,
		&pbdhcpagent.DeleteAdmitDuidRequest{Duid: admitDuidId}, nil)
}

func (d *AdmitDuidService) Update(admitDuid *resource.AdmitDuid) error {
	if err := admitDuid.Validate(); err != nil {
		return fmt.Errorf("validate admit duid %s failed: %s", admitDuid.GetID(), err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableAdmitDuid,
			map[string]interface{}{resource.SqlColumnComment: admitDuid.Comment},
			map[string]interface{}{restdb.IDField: admitDuid.GetID()}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found admit duid %s", admitDuid.GetID())
		}

		return nil
	}); err != nil {
		return fmt.Errorf("update admit duid %s failed:%s", admitDuid.GetID(), err.Error())
	}

	return nil
}
