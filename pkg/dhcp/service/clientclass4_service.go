package service

import (
	"fmt"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	ClientClass4Option60 = "option vendor-class-identifier == '%s'"
)

type ClientClass4Service struct {
}

func NewClientClass4Service() *ClientClass4Service {
	return &ClientClass4Service{}
}

func (c *ClientClass4Service) Create(clientClass *resource.ClientClass4) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(clientClass); err != nil {
			return err
		}
		return sendCreateClientClass4CmdToAgent(clientClass)
	}); err != nil {
		return nil, err
	}
	return clientClass, nil
}

func sendCreateClientClass4CmdToAgent(clientclass *resource.ClientClass4) error {
	err := kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateClientClass4,
		&pbdhcpagent.CreateClientClass4Request{
			Name:   clientclass.Name,
			Code:   60,
			Regexp: fmt.Sprintf(ClientClass4Option60, clientclass.Regexp),
		})
	if err != nil {
		if err := sendDeleteClientClass4CmdToDHCPAgent(clientclass.Name); err != nil {
			log.Errorf("add clientclass4 %s failed, rollback it failed: %s",
				clientclass.Name, err.Error())
		}
	}

	return err
}

func (c *ClientClass4Service) List() (interface{}, error) {
	var clientClasses []*resource.ClientClass4
	if err := db.GetResources(map[string]interface{}{util.SqlOrderBy: util.SqlColumnsName},
		&clientClasses); err != nil {
		return nil, err
	}

	return clientClasses, nil
}

func (c *ClientClass4Service) Get(clientclassID string) (restresource.Resource, error) {
	var clientclasses []*resource.ClientClass4
	clientclass, err := restdb.GetResourceWithID(db.GetDB(), clientclassID, &clientclasses)
	if err != nil {
		return nil, err
	}
	return clientclass.(*resource.ClientClass4), nil
}

func (c *ClientClass4Service) Update(clientclass *resource.ClientClass4) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableClientClass4,
			map[string]interface{}{resource.SqlColumnClassRegexp: clientclass.Regexp},
			map[string]interface{}{restdb.IDField: clientclass.GetID()}); err != nil {
			return err
		}
		return sendUpdateClientClass4CmdToDHCPAgent(clientclass)
	}); err != nil {
		return nil, err
	}
	return clientclass, nil
}

func sendUpdateClientClass4CmdToDHCPAgent(clientclass *resource.ClientClass4) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdateClientClass4,
		&pbdhcpagent.UpdateClientClass4Request{
			Name:   clientclass.Name,
			Code:   60,
			Regexp: fmt.Sprintf(ClientClass4Option60, clientclass.Regexp),
		})
}

func (c *ClientClass4Service) Delete(clientclassID string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exist, err := tx.Exists(resource.TableSubnet4,
			map[string]interface{}{resource.SqlColumnClassID: clientclassID}); err != nil {
			return err
		} else if exist {
			return fmt.Errorf("client class %s used by subnet4", clientclassID)
		}
		if _, err := tx.Delete(resource.TableClientClass4,
			map[string]interface{}{restdb.IDField: clientclassID}); err != nil {
			return err
		}
		return sendDeleteClientClass4CmdToDHCPAgent(clientclassID)
	}); err != nil {
		return err
	}
	return nil
}

func sendDeleteClientClass4CmdToDHCPAgent(clientClassID string) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteClientClass4,
		&pbdhcpagent.DeleteClientClass4Request{
			Name: clientClassID,
		})
}
