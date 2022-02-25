package service

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type AdmitMacService struct{}

func NewAdmitMacService() *AdmitMacService {
	return &AdmitMacService{}
}

func (d *AdmitMacService) Create(admitMac *resource.AdmitMac) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitMac); err != nil {
			return err
		}
		return sendCreateAdmitMacCmdToDHCPAgent(admitMac)
	}); err != nil {
		return nil, err
	}
	return admitMac, nil
}

func sendCreateAdmitMacCmdToDHCPAgent(admitMac *resource.AdmitMac) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateAdmitMac,
		&pbdhcpagent.CreateAdmitMacRequest{
			HwAddress: admitMac.HwAddress,
		})
}

func (d *AdmitMacService) List(conditions map[string]interface{}) (interface{}, error) {
	var macs []*resource.AdmitMac
	if err := db.GetResources(conditions, &macs); err != nil {
		return nil, err
	}
	return macs, nil
}

func (d *AdmitMacService) Get(admitMacID string) (restresource.Resource, error) {
	var admitMacs []*resource.AdmitMac
	admitMac, err := restdb.GetResourceWithID(db.GetDB(), admitMacID, &admitMacs)
	if err != nil {
		return nil, err
	}
	return admitMac.(*resource.AdmitMac), nil
}

func (d *AdmitMacService) Delete(admitMacId string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitMac,
			map[string]interface{}{restdb.IDField: admitMacId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found admit mac %s", admitMacId)
		}

		return sendDeleteAdmitMacCmdToDHCPAgent(admitMacId)
	}); err != nil {
		return err
	}

	return nil
}

func sendDeleteAdmitMacCmdToDHCPAgent(admitMacId string) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteAdmitMac,
		&pbdhcpagent.DeleteAdmitMacRequest{
			HwAddress: admitMacId,
		})
}

func (d *AdmitMacService) Update(admitMac *resource.AdmitMac) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableAdmitMac,
			map[string]interface{}{util.SqlColumnsComment: admitMac.Comment},
			map[string]interface{}{restdb.IDField: admitMac.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found admit mac %s", admitMac.GetID())
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return admitMac, nil
}
