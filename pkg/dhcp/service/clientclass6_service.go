package service

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	"github.com/linkingthing/clxone-dhcp/pkg/util"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
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

func (c *ClientClass6Service) Create(clientclass *resource.ClientClass6) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(clientclass); err != nil {
			return err
		}
		return sendCreateClientClass6CmdToAgent(clientclass)
	}); err != nil {
		return nil, err
	}

	return clientclass, nil
}

func sendCreateClientClass6CmdToAgent(clientclass *resource.ClientClass6) error {
	err := kafka.GetDHCPAgentService().SendDHCPCmd(kafka.CreateClientClass6,
		&pbdhcpagent.CreateClientClass6Request{
			Name:   clientclass.Name,
			Code:   16,
			Regexp: fmt.Sprintf(ClientClass6Option60, clientclass.Regexp),
		})
	if err != nil {
		if err := sendDeleteClientClass6CmdToDHCPAgent(clientclass.Name); err != nil {
			log.Errorf("add clientclass6 %s failed, rollback it failed: %s",
				clientclass.Name, err.Error())
		}
	}

	return err
}

func (c *ClientClass6Service) List() (interface{}, error) {
	var clientclasses []*resource.ClientClass6
	if err := db.GetResources(map[string]interface{}{util.SqlOrderBy: util.SqlColumnsName},
		&clientclasses); err != nil {
		return nil, err
	}

	return clientclasses, nil
}

func (c *ClientClass6Service) Get(clientclassID string) (restresource.Resource, error) {
	var clientclasses []*resource.ClientClass6
	clientclass, err := restdb.GetResourceWithID(db.GetDB(), clientclassID, &clientclasses)
	if err != nil {
		return nil, err
	}
	return clientclass.(*resource.ClientClass6), nil
}

func (c *ClientClass6Service) Update(clientclass *resource.ClientClass6) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableClientClass6,
			map[string]interface{}{resource.SqlColumnClassRegexp: clientclass.Regexp},
			map[string]interface{}{restdb.IDField: clientclass.GetID()}); err != nil {
			return err
		}

		return sendUpdateClientClass6CmdToDHCPAgent(clientclass)
	}); err != nil {
		return nil, err
	}
	return clientclass, nil
}

func sendUpdateClientClass6CmdToDHCPAgent(clientclass *resource.ClientClass6) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.UpdateClientClass6,
		&pbdhcpagent.UpdateClientClass6Request{
			Name:   clientclass.Name,
			Code:   16,
			Regexp: fmt.Sprintf(ClientClass6Option60, clientclass.Regexp),
		})
}

func (c *ClientClass6Service) Delete(clientclassID string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exist, err := tx.Exists(resource.TableSubnet6,
			map[string]interface{}{resource.SqlColumnClassID: clientclassID}); err != nil {
			return err
		} else if exist {
			return fmt.Errorf("client class %s used by subnet6", clientclassID)
		}

		if _, err := tx.Delete(resource.TableClientClass6,
			map[string]interface{}{restdb.IDField: clientclassID}); err != nil {
			return err
		}

		return sendDeleteClientClass6CmdToDHCPAgent(clientclassID)
	}); err != nil {
		return err
	}

	return nil
}

func sendDeleteClientClass6CmdToDHCPAgent(clientClassID string) error {
	return kafka.GetDHCPAgentService().SendDHCPCmd(kafka.DeleteClientClass6,
		&pbdhcpagent.DeleteClientClass6Request{
			Name: clientClassID,
		})
}
