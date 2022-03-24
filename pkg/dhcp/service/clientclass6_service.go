package service

import (
	"fmt"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
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
		return fmt.Errorf("validate clientclass6 %s failed: %s",
			clientClass.GetID(), err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(clientClass); err != nil {
			return err
		}

		return sendCreateClientClass6CmdToAgent(clientClass)
	}); err != nil {
		return fmt.Errorf("create clientclass6 %s failed:%s",
			clientClass.GetID(), err.Error())
	}

	return nil
}

func sendCreateClientClass6CmdToAgent(clientClass *resource.ClientClass6) error {
	err := kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateClientClass6,
		&pbdhcpagent.CreateClientClass6Request{
			Name:   clientClass.Name,
			Code:   16,
			Regexp: fmt.Sprintf(ClientClass6Option60, clientClass.Regexp),
		})
	if err != nil {
		if err := sendDeleteClientClass6CmdToDHCPAgent(clientClass.Name); err != nil {
			log.Errorf("add clientclass6 %s failed, rollback it failed: %s",
				clientClass.Name, err.Error())
		}
	}

	return err
}

func (c *ClientClass6Service) List() ([]*resource.ClientClass6, error) {
	var clientClasses []*resource.ClientClass6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{
			resource.SqlOrderBy: resource.SqlColumnName}, &clientClasses)
	}); err != nil {
		return nil, fmt.Errorf("list clientclass6 failed:%s", err.Error())
	}

	return clientClasses, nil
}

func (c *ClientClass6Service) Get(id string) (*resource.ClientClass6, error) {
	var clientClasses []*resource.ClientClass6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &clientClasses)
	}); err != nil {
		return nil, fmt.Errorf("get clientclass6 %s failed:%s", id, err.Error())
	} else if len(clientClasses) == 0 {
		return nil, fmt.Errorf("no found clientclass6 %s", id)
	}

	return clientClasses[0], nil
}

func (c *ClientClass6Service) Update(clientClass *resource.ClientClass6) error {
	if err := clientClass.Validate(); err != nil {
		return fmt.Errorf("validate clientclass6 %s failed: %s",
			clientClass.GetID(), err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableClientClass6, map[string]interface{}{
			resource.SqlColumnClassRegexp: clientClass.Regexp,
		}, map[string]interface{}{restdb.IDField: clientClass.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found clientclass6 %s", clientClass.GetID())
		}

		return sendUpdateClientClass6CmdToDHCPAgent(clientClass)
	}); err != nil {
		return fmt.Errorf("update clientclass6 %s failed:%s",
			clientClass.GetID(), err.Error())
	}

	return nil
}

func sendUpdateClientClass6CmdToDHCPAgent(clientClass *resource.ClientClass6) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdateClientClass6,
		&pbdhcpagent.UpdateClientClass6Request{
			Name:   clientClass.Name,
			Code:   16,
			Regexp: fmt.Sprintf(ClientClass6Option60, clientClass.Regexp),
		})
}

func (c *ClientClass6Service) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exist, err := tx.Exists(resource.TableSubnet6,
			map[string]interface{}{resource.SqlColumnClientClass: id}); err != nil {
			return err
		} else if exist {
			return fmt.Errorf("client class %s used by subnet6", id)
		}

		if rows, err := tx.Delete(resource.TableClientClass6,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found clientclass6 %s", id)
		}

		return sendDeleteClientClass6CmdToDHCPAgent(id)
	}); err != nil {
		return fmt.Errorf("delete clientclass6 %s failed:%s", id, err.Error())
	}

	return nil
}

func sendDeleteClientClass6CmdToDHCPAgent(clientClassID string) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteClientClass6,
		&pbdhcpagent.DeleteClientClass6Request{
			Name: clientClassID,
		})
}
