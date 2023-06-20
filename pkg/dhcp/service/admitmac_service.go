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

type AdmitMacService struct{}

func NewAdmitMacService() *AdmitMacService {
	return &AdmitMacService{}
}

func (d *AdmitMacService) Create(admitMac *resource.AdmitMac) error {
	if err := admitMac.Validate(); err != nil {
		return err
	}

	admitMac.SetID(admitMac.HwAddress)
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitMac); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameMac, admitMac.HwAddress, err)
		}

		return sendCreateAdmitMacCmdToDHCPAgent(admitMac)
	})
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
	if mac, ok := conditions[resource.SqlColumnHwAddress].(string); ok {
		if mac, _ = util.NormalizeMac(mac); mac != "" {
			conditions[resource.SqlColumnHwAddress] = mac
		}
	}

	var macs []*resource.AdmitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &macs)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAdmit), pg.Error(err).Error())
	}

	return macs, nil
}

func (d *AdmitMacService) Get(id string) (*resource.AdmitMac, error) {
	var admitMacs []*resource.AdmitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &admitMacs)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(admitMacs) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameAdmit, id)
	}

	return admitMacs[0], nil
}

func (d *AdmitMacService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitMac,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAdmit, id)
		}

		return sendDeleteAdmitMacCmdToDHCPAgent(id)
	})
}

func sendDeleteAdmitMacCmdToDHCPAgent(admitMacId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAdmitMac,
		&pbdhcpagent.DeleteAdmitMacRequest{HwAddress: admitMacId}, nil)
}

func (d *AdmitMacService) Update(admitMac *resource.AdmitMac) error {
	if err := admitMac.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableAdmitMac,
			map[string]interface{}{resource.SqlColumnComment: admitMac.Comment},
			map[string]interface{}{restdb.IDField: admitMac.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, admitMac.GetID(), pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAdmit, admitMac.GetID())
		}

		return nil
	})
}
