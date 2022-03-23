package service

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type PingerService struct {
}

func NewPingerService() (*PingerService, error) {
	if err := CreateDefaultPinger(); err != nil {
		return nil, err
	}

	return &PingerService{}, nil
}

func CreateDefaultPinger() error {
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

func (p *PingerService) List() ([]*resource.Pinger, error) {
	var pingers []*resource.Pinger
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &pingers)
	}); err != nil {
		return nil, fmt.Errorf("list pinger failed:%s", err.Error())
	}

	return pingers, nil
}

func (p *PingerService) Get(id string) (*resource.Pinger, error) {
	var pingers []*resource.Pinger
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &pingers)
	}); err != nil {
		return nil, fmt.Errorf("get pinger %s failed:%s", id, err.Error())
	} else if len(pingers) == 0 {
		return nil, fmt.Errorf("no found pinger %s", id)
	}

	return pingers[0], nil
}

func (p *PingerService) Update(pinger *resource.Pinger) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TablePinger, map[string]interface{}{
			resource.SqlColumnEnabled: pinger.Enabled,
			resource.SqlColumnTimeout: pinger.Timeout,
		}, map[string]interface{}{restdb.IDField: pinger.GetID()}); err != nil {
			return err
		}

		return sendUpdatePingerCmdToDHCPAgent(pinger)
	}); err != nil {
		return fmt.Errorf("update pinger %s failed:%s", pinger.GetID(), err.Error())
	}

	return nil
}

func sendUpdatePingerCmdToDHCPAgent(pinger *resource.Pinger) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdatePinger,
		&pbdhcpagent.UpdatePingerRequest{
			Enabled: pinger.Enabled,
			Timeout: pinger.Timeout,
		})
}
