package service

import (
	"fmt"

	pg "github.com/linkingthing/clxone-utils/postgresql"
	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type AdmitMacService struct{}

func NewAdmitMacService() *AdmitMacService {
	return &AdmitMacService{}
}

func (d *AdmitMacService) Create(admitMac *resource.AdmitMac) error {
	admitMac.SetID(admitMac.HwAddress)
	if err := admitMac.Validate(); err != nil {
		return fmt.Errorf("validate admit mac %s failed: %s", admitMac.GetID(), err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitMac); err != nil {
			return pg.Error(err)
		}

		return sendCreateAdmitMacCmdToDHCPAgent(admitMac)
	}); err != nil {
		return fmt.Errorf("create admit mac failed:%s", err.Error())
	}

	return nil
}

func sendCreateAdmitMacCmdToDHCPAgent(admitMac *resource.AdmitMac) error {
	return kafka.SendDHCPCmd(kafka.CreateAdmitMac,
		&pbdhcpagent.CreateAdmitMacRequest{HwAddress: admitMac.HwAddress},
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteAdmitMac,
				&pbdhcpagent.DeleteAdmitMacRequest{HwAddress: admitMac.HwAddress},
			); err != nil {
				log.Errorf("create admit mac %s failed, rollback with nodes %v failed: %s",
					admitMac.HwAddress, nodesForSucceed, err.Error())
			}
		})
}

func (d *AdmitMacService) List(conditions map[string]interface{}) ([]*resource.AdmitMac, error) {
	var macs []*resource.AdmitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &macs)
	}); err != nil {
		return nil, fmt.Errorf("list admit mac failed:%s", pg.Error(err).Error())
	}

	return macs, nil
}

func (d *AdmitMacService) Get(id string) (*resource.AdmitMac, error) {
	var admitMacs []*resource.AdmitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &admitMacs)
	}); err != nil {
		return nil, fmt.Errorf("get admit mac %s failed:%s", id, pg.Error(err).Error())
	} else if len(admitMacs) == 0 {
		return nil, fmt.Errorf("no found admit mac %s", id)
	}

	return admitMacs[0], nil
}

func (d *AdmitMacService) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitMac,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found admit mac %s", id)
		}

		return sendDeleteAdmitMacCmdToDHCPAgent(id)
	}); err != nil {
		return fmt.Errorf("delete admit mac %s failed:%s", id, err.Error())
	}

	return nil
}

func sendDeleteAdmitMacCmdToDHCPAgent(admitMacId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAdmitMac,
		&pbdhcpagent.DeleteAdmitMacRequest{HwAddress: admitMacId}, nil)
}

func (d *AdmitMacService) Update(admitMac *resource.AdmitMac) error {
	if err := admitMac.Validate(); err != nil {
		return fmt.Errorf("update admit mac %s failed: %s",
			admitMac.GetID(), err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableAdmitMac,
			map[string]interface{}{resource.SqlColumnComment: admitMac.Comment},
			map[string]interface{}{restdb.IDField: admitMac.GetID()}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found admit mac %s", admitMac.GetID())
		}

		return nil
	}); err != nil {
		return fmt.Errorf("update admit mac of %s failed:%s",
			admitMac.GetID(), err.Error())
	}

	return nil
}
