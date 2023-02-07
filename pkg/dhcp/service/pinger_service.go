package service

import (
	"fmt"

	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type PingerService struct {
}

func NewPingerService() (*PingerService, error) {
	if err := createDefaultPinger(); err != nil {
		return nil, err
	}

	return &PingerService{}, nil
}

func createDefaultPinger() error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(resource.TablePinger, nil); err != nil {
			return fmt.Errorf("check dhcp pinger failed: %s", pg.Error(err).Error())
		} else if !exists {
			if _, err := tx.Insert(resource.DefaultPinger); err != nil {
				return fmt.Errorf("insert default dhcp pinger failed: %s", pg.Error(err).Error())
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNamePinger), pg.Error(err).Error())
	}

	return pingers, nil
}

func (p *PingerService) Get(id string) (*resource.Pinger, error) {
	var pingers []*resource.Pinger
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &pingers)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(pingers) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNamePinger, id)
	}

	return pingers[0], nil
}

func (p *PingerService) Update(pinger *resource.Pinger) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePinger, map[string]interface{}{
			resource.SqlColumnEnabled: pinger.Enabled,
			resource.SqlColumnTimeout: pinger.Timeout,
		}, map[string]interface{}{restdb.IDField: pinger.GetID()}); err != nil {
			return pg.Error(err)
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, pinger.GetID(), pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNamePinger, pinger.GetID())
		}

		return sendUpdatePingerCmdToDHCPAgent(pinger)
	})
}

func sendUpdatePingerCmdToDHCPAgent(pinger *resource.Pinger) error {
	return kafka.SendDHCPCmd(kafka.UpdatePinger,
		&pbdhcpagent.UpdatePingerRequest{
			Enabled: pinger.Enabled,
			Timeout: pinger.Timeout,
		}, nil)
}
