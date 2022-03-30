package service

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
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
			return fmt.Errorf("check dhcp admit failed: %s", err.Error())
		} else if exists == false {
			if _, err := tx.Insert(resource.DefaultAdmit); err != nil {
				return fmt.Errorf("insert default dhcp admit failed: %s", err.Error())
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
		return nil, fmt.Errorf("list admit failed: %s", err.Error())
	}

	return admits, nil
}

func (d *AdmitService) Get(id string) (*resource.Admit, error) {
	var admits []*resource.Admit
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &admits)
	}); err != nil {
		return nil, fmt.Errorf("get admit %s failed:%s", id, err.Error())
	} else if len(admits) == 0 {
		return nil, fmt.Errorf("no found admit %s", id)
	}

	return admits[0], nil
}

func (d *AdmitService) Update(admit *resource.Admit) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableAdmit,
			map[string]interface{}{resource.SqlColumnEnabled: admit.Enabled},
			map[string]interface{}{restdb.IDField: admit.GetID()}); err != nil {
			return err
		}

		return sendUpdateAdmitCmdToDHCPAgent(admit)
	}); err != nil {
		return fmt.Errorf("update admit failed: %s", err.Error())
	}

	return nil
}

func sendUpdateAdmitCmdToDHCPAgent(admit *resource.Admit) error {
	return kafka.SendDHCPCmd(kafka.UpdateAdmit,
		&pbdhcpagent.UpdateAdmitRequest{Enabled: admit.Enabled}, nil)
}
