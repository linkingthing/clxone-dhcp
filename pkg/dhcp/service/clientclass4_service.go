package service

import (
	"fmt"

	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

const (
	ClientClass4Option60 = "option vendor-class-identifier == '%s'"
)

type ClientClass4Service struct {
}

func NewClientClass4Service() *ClientClass4Service {
	return &ClientClass4Service{}
}

func (c *ClientClass4Service) Create(clientClass *resource.ClientClass4) error {
	clientClass.SetID(clientClass.Name)
	if err := clientClass.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(clientClass); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, clientClass.Name, pg.Error(err).Error())
		}

		return sendCreateClientClass4CmdToAgent(clientClass)
	})
}

func sendCreateClientClass4CmdToAgent(clientClass4 *resource.ClientClass4) error {
	return kafka.SendDHCPCmd(kafka.CreateClientClass4,
		&pbdhcpagent.CreateClientClass4Request{
			Name:   clientClass4.Name,
			Code:   60,
			Regexp: fmt.Sprintf(ClientClass4Option60, clientClass4.Regexp),
		}, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteClientClass4,
				&pbdhcpagent.DeleteClientClass4Request{Name: clientClass4.Name}); err != nil {
				log.Errorf("add clientclass4 %s failed, rollback with nodes %v failed: %s",
					clientClass4.Name, nodesForSucceed, err.Error())
			}
		})
}

func (c *ClientClass4Service) List() ([]*resource.ClientClass4, error) {
	var clientClasses []*resource.ClientClass4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{resource.SqlOrderBy: resource.SqlColumnName}, &clientClasses)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameClientClass), pg.Error(err).Error())
	}

	return clientClasses, nil
}

func (c *ClientClass4Service) Get(id string) (*resource.ClientClass4, error) {
	var clientClasses []*resource.ClientClass4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &clientClasses)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(clientClasses) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameClientClass, id)
	}

	return clientClasses[0], nil
}

func (c *ClientClass4Service) Update(clientClass *resource.ClientClass4) error {
	if err := clientClass.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableClientClass4,
			map[string]interface{}{resource.SqlColumnClassRegexp: clientClass.Regexp},
			map[string]interface{}{restdb.IDField: clientClass.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, clientClass.GetID(), pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameClientClass, clientClass.GetID())
		}

		return sendUpdateClientClass4CmdToDHCPAgent(clientClass)
	})
}

func sendUpdateClientClass4CmdToDHCPAgent(clientClass *resource.ClientClass4) error {
	return kafka.SendDHCPCmd(kafka.UpdateClientClass4,
		&pbdhcpagent.UpdateClientClass4Request{
			Name:   clientClass.Name,
			Code:   60,
			Regexp: fmt.Sprintf(ClientClass4Option60, clientClass.Regexp),
		}, nil)
}

func (c *ClientClass4Service) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exist, err := tx.Exists(resource.TableSubnet4,
			map[string]interface{}{resource.SqlColumnClientClass: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
		} else if exist {
			return errorno.ErrBeenUsed(errorno.ErrNameClientClass, id)
		}

		if rows, err := tx.Delete(resource.TableClientClass4,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameClientClass, id)
		}

		return sendDeleteClientClass4CmdToDHCPAgent(id)
	})
}

func sendDeleteClientClass4CmdToDHCPAgent(clientClassID string) error {
	return kafka.SendDHCPCmd(kafka.DeleteClientClass4,
		&pbdhcpagent.DeleteClientClass4Request{Name: clientClassID}, nil)
}
