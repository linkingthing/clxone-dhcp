package service

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"

	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type AdmitService struct{}

func NewAdmitService() *AdmitService {
	return &AdmitService{}
}

func (d *AdmitService) CreateDefaultAdmit() error {
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

func (d *AdmitService) List() (interface{}, error) {
	var admits []*resource.Admit
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &admits)
	}); err != nil {
		return nil, fmt.Errorf("list dhcp admit from db failed: %s", err.Error())
	}
	return admits, nil
}

func (d *AdmitService) Get(admitID string) (restresource.Resource, *resterror.APIError) {
	var admits []*resource.Admit
	admit, err := restdb.GetResourceWithID(db.GetDB(), admitID, &admits)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get admit from db failed: %s", err.Error()))
	}
	return admit.(*resource.Admit), nil
}

func (d *AdmitService) Update(admit *resource.Admit) (restresource.Resource, error) {
	cond := map[string]interface{}{restdb.IDField: admit.GetID()}
	newValue := map[string]interface{}{resource.SqlColumnEnabled: admit.Enabled}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableAdmit, newValue, cond); err != nil {
			return err
		}
		return sendUpdateAdmitCmdToDHCPAgent(admit)
	}); err != nil {
		return nil, fmt.Errorf("update dhcp admit failed: %s", err.Error())
	}

	return admit, nil
}

func sendUpdateAdmitCmdToDHCPAgent(admit *resource.Admit) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdateAdmit,
		&pbdhcpagent.UpdateAdmitRequest{
			Enabled: admit.Enabled,
		})
}
