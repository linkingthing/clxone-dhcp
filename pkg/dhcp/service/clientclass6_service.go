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
	ClientClass6Option60 = "option vendor-class-identifier == '%s'"
)

type ClientClass6Service struct {
}

func NewClientClass6Service() *ClientClass6Service {
	return &ClientClass6Service{}
}

func (c *ClientClass6Service) Create(clientClass *resource.ClientClass6) error {
	clientClass.SetID(clientClass.Name)
	if err := clientClass.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(clientClass); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
		}

		return sendCreateClientClass6CmdToAgent(clientClass)
	})
}

func sendCreateClientClass6CmdToAgent(clientClass *resource.ClientClass6) error {
	return kafka.SendDHCPCmd(kafka.CreateClientClass6,
		&pbdhcpagent.CreateClientClass6Request{
			Name:   clientClass.Name,
			Code:   16,
			Regexp: fmt.Sprintf(ClientClass6Option60, clientClass.Regexp),
		}, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteClientClass6,
				&pbdhcpagent.DeleteClientClass6Request{Name: clientClass.Name}); err != nil {
				log.Errorf("add clientclass6 %s failed, rollback with nodes %v failed: %s",
					clientClass.Name, nodesForSucceed, err.Error())
			}
		})
}

func (c *ClientClass6Service) List() ([]*resource.ClientClass6, error) {
	var clientClasses []*resource.ClientClass6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{
			resource.SqlOrderBy: resource.SqlColumnName}, &clientClasses)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameClientClass), pg.Error(err).Error())
	}

	return clientClasses, nil
}

func (c *ClientClass6Service) Get(id string) (*resource.ClientClass6, error) {
	var clientClasses []*resource.ClientClass6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &clientClasses)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(clientClasses) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameClientClass, id)
	}

	return clientClasses[0], nil
}

func (c *ClientClass6Service) Update(clientClass *resource.ClientClass6) error {
	if err := clientClass.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableClientClass6, map[string]interface{}{
			resource.SqlColumnClassRegexp: clientClass.Regexp,
		}, map[string]interface{}{restdb.IDField: clientClass.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, clientClass.GetID(), pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameClientClass, clientClass.GetID())
		}

		return sendUpdateClientClass6CmdToDHCPAgent(clientClass)
	})
}

func sendUpdateClientClass6CmdToDHCPAgent(clientClass *resource.ClientClass6) error {
	return kafka.SendDHCPCmd(kafka.UpdateClientClass6,
		&pbdhcpagent.UpdateClientClass6Request{
			Name:   clientClass.Name,
			Code:   16,
			Regexp: fmt.Sprintf(ClientClass6Option60, clientClass.Regexp),
		}, nil)
}

func (c *ClientClass6Service) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exist, err := tx.Exists(resource.TableSubnet6,
			map[string]interface{}{resource.SqlColumnClientClass: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
		} else if exist {
			return errorno.ErrUsedBy(errorno.ErrNameClientClass, "", string(errorno.ErrNameNetworkV6))
		}

		if rows, err := tx.Delete(resource.TableClientClass6,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameClientClass, id)
		}

		return sendDeleteClientClass6CmdToDHCPAgent(id)
	})
}

func sendDeleteClientClass6CmdToDHCPAgent(clientClassID string) error {
	return kafka.SendDHCPCmd(kafka.DeleteClientClass6,
		&pbdhcpagent.DeleteClientClass6Request{Name: clientClassID}, nil)
}
