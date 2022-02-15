package service

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type PingerService struct {
}

func NewPingerService() *PingerService {
	return &PingerService{}
}

func (p *PingerService) CreateDefaultPinger() error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(resource.TablePinger, nil); err != nil {
			return fmt.Errorf("check dhcp pinger failed: %s", err.Error())
		} else if exists == false {
			if _, err := tx.Insert(resource.DefaultPinger); err != nil {
				return fmt.Errorf("insert default dhcp pinger failed: %s", err.Error())
			}
		}

		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (p *PingerService) List() (interface{}, error) {
	var pingers []*resource.Pinger
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &pingers)
	}); err != nil {
		return nil, err
	}

	return pingers, nil
}

func (p *PingerService) Get(pingerID string) (restresource.Resource, error) {
	var pingers []*resource.Pinger
	pinger, err := restdb.GetResourceWithID(db.GetDB(), pingerID, &pingers)
	if err != nil {
		return nil, err
	}
	return pinger.(*resource.Pinger), nil
}

func (p *PingerService) Update(pinger *resource.Pinger) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TablePinger, map[string]interface{}{
			resource.SqlColumnEnabled: pinger.Enabled,
			resource.SqlColumnTimeout: pinger.Timeout,
		}, map[string]interface{}{restdb.IDField: pinger.GetID()}); err != nil {
			return err
		}

		return sendUpdatePingerCmdToDHCPAgent(pinger)
	}); err != nil {
		return nil, err
	}

	return pinger, nil
}

func sendUpdatePingerCmdToDHCPAgent(pinger *resource.Pinger) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdatePinger,
		&pbdhcpagent.UpdatePingerRequest{
			Enabled: pinger.Enabled,
			Timeout: pinger.Timeout,
		})
}
