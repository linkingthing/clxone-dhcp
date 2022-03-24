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
		return fmt.Errorf("validate clientclass4 %s failed: %s",
			clientClass.GetID(), err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(clientClass); err != nil {
			return err
		}

		return sendCreateClientClass4CmdToAgent(clientClass)
	}); err != nil {
		return fmt.Errorf("create clientclass4 %s failed:%s",
			clientClass.GetID(), err.Error())
	}

	return nil
}

func sendCreateClientClass4CmdToAgent(clientClass4 *resource.ClientClass4) error {
	err := kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateClientClass4,
		&pbdhcpagent.CreateClientClass4Request{
			Name:   clientClass4.Name,
			Code:   60,
			Regexp: fmt.Sprintf(ClientClass4Option60, clientClass4.Regexp),
		})
	if err != nil {
		if err := sendDeleteClientClass4CmdToDHCPAgent(clientClass4.Name); err != nil {
			log.Errorf("add clientclass4 %s failed, rollback it failed: %s",
				clientClass4.Name, err.Error())
		}
	}

	return err
}

func (c *ClientClass4Service) List() ([]*resource.ClientClass4, error) {
	var clientClasses []*resource.ClientClass4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{
			resource.SqlOrderBy: resource.SqlColumnName},
			&clientClasses)
	}); err != nil {
		return nil, fmt.Errorf("list clientclass4 failed:%s", err.Error())
	}

	return clientClasses, nil
}

func (c *ClientClass4Service) Get(id string) (*resource.ClientClass4, error) {
	var clientClasses []*resource.ClientClass4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &clientClasses)
	}); err != nil {
		return nil, fmt.Errorf("get clientclass4 of %s failed:%s", id, err.Error())
	} else if len(clientClasses) == 0 {
		return nil, fmt.Errorf("no found clientclass4 %s", id)
	}

	return clientClasses[0], nil
}

func (c *ClientClass4Service) Update(clientClass *resource.ClientClass4) error {
	if err := clientClass.Validate(); err != nil {
		return fmt.Errorf("validate clientclass4 %s failed: %s",
			clientClass.GetID(), err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableClientClass4,
			map[string]interface{}{resource.SqlColumnClassRegexp: clientClass.Regexp},
			map[string]interface{}{restdb.IDField: clientClass.GetID()}); err != nil {
			return err
		}

		return sendUpdateClientClass4CmdToDHCPAgent(clientClass)
	}); err != nil {
		return fmt.Errorf("update clientclass4 %s failed:%s",
			clientClass.GetID(), err.Error())
	}

	return nil
}

func sendUpdateClientClass4CmdToDHCPAgent(clientClass *resource.ClientClass4) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdateClientClass4,
		&pbdhcpagent.UpdateClientClass4Request{
			Name:   clientClass.Name,
			Code:   60,
			Regexp: fmt.Sprintf(ClientClass4Option60, clientClass.Regexp),
		})
}

func (c *ClientClass4Service) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exist, err := tx.Exists(resource.TableSubnet4,
			map[string]interface{}{resource.SqlColumnClientClass: id}); err != nil {
			return err
		} else if exist {
			return fmt.Errorf("client class %s used by subnet4", id)
		}

		if rows, err := tx.Delete(resource.TableClientClass4,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found clientclass4 %s", id)
		}

		return sendDeleteClientClass4CmdToDHCPAgent(id)
	}); err != nil {
		return fmt.Errorf("delete clientclass4 %s failed:%s", id, err.Error())
	}

	return nil
}

func sendDeleteClientClass4CmdToDHCPAgent(clientClassID string) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteClientClass4,
		&pbdhcpagent.DeleteClientClass4Request{
			Name: clientClassID,
		})
}
