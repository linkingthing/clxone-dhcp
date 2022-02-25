package service

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

const (
	FieldDuid = "duid"
)

type AdmitDuIdService struct{}

func NewAdmitDuIdService() *AdmitDuIdService {
	return &AdmitDuIdService{}
}

func (d *AdmitDuIdService) Create(admitDuid *resource.AdmitDuid) (restresource.Resource, error) {
	admitDuid.SetID(admitDuid.Duid)
	if err := admitDuid.Validate(); err != nil {
		return nil, fmt.Errorf("create admit duid Validate err : %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitDuid); err != nil {
			return err
		}
		return sendCreateAdmitDuidCmdToDHCPAgent(admitDuid)
	}); err != nil {
		return nil, fmt.Errorf("create admit duid insertdb failed: %s", err.Error())
	}

	return admitDuid, nil
}

func sendCreateAdmitDuidCmdToDHCPAgent(admitDuid *resource.AdmitDuid) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateAdmitDuid,
		&pbdhcpagent.CreateAdmitDuidRequest{
			Duid: admitDuid.Duid,
		})
}

func (d *AdmitDuIdService) List(conditions map[string]interface{}) (interface{}, error) {
	var duids []*resource.AdmitDuid
	if err := db.GetResources(conditions, &duids); err != nil {
		return nil, err
	}
	return duids, nil
}

func (d *AdmitDuIdService) Get(admitDuidID string) (restresource.Resource, error) {
	var admitDuids []*resource.AdmitDuid
	admitDuid, err := restdb.GetResourceWithID(db.GetDB(), admitDuidID, &admitDuids)
	if err != nil {
		return nil, err
	}
	return admitDuid.(*resource.AdmitDuid), nil
}

func (d *AdmitDuIdService) Delete(admitDuidId string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitDuid,
			map[string]interface{}{restdb.IDField: admitDuidId}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found admit duid %s", admitDuidId)
		}
		return sendDeleteAdmitDuidCmdToDHCPAgent(admitDuidId)
	}); err != nil {
		return err
	}

	return nil
}

func sendDeleteAdmitDuidCmdToDHCPAgent(admitDuidId string) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteAdmitDuid,
		&pbdhcpagent.DeleteAdmitDuidRequest{
			Duid: admitDuidId,
		})
}

func (d *AdmitDuIdService) Update(admitDuid *resource.AdmitDuid) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableAdmitDuid,
			map[string]interface{}{"comment": admitDuid.Comment},
			map[string]interface{}{restdb.IDField: admitDuid.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found admit duid %s", admitDuid.GetID())
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return admitDuid, nil
}
