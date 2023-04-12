package service

import (
	"fmt"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"

	pg "github.com/linkingthing/clxone-utils/postgresql"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
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
			return pg.Error(err)
		}

		return sendCreateClientClass6CmdToAgent(clientClass)
	}); err != nil {
		return fmt.Errorf("create clientclass6 %s failed:%s",
			clientClass.GetID(), err.Error())
	}

	return nil
}

func sendCreateClientClass6CmdToAgent(clientClass *resource.ClientClass6) error {
	return kafka.SendDHCPCmd(kafka.CreateClientClass6,
		&pbdhcpagent.CreateClientClass6Request{
			Name:   clientClass.Name,
			Code:   uint32(clientClass.Code),
			Regexp: genClientClass6Regexp(clientClass),
		}, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteClientClass6,
				&pbdhcpagent.DeleteClientClass6Request{Name: clientClass.Name}); err != nil {
				log.Errorf("add clientclass6 %s failed, rollback with nodes %v failed: %s",
					clientClass.Name, nodesForSucceed, err.Error())
			}
		})
}

func genClientClass6Regexp(clientclass *resource.ClientClass6) string {
	switch clientclass.Condition {
	case resource.OptionConditionEqual:
		return fmt.Sprintf(ClientClassOptionEqual, clientclass.Description, clientclass.Regexp)
	case resource.OptionConditionSubstringEqual:
		return fmt.Sprintf(ClientClassOptionSubstringEqual, clientclass.Code,
			clientclass.BeginIndex, len(clientclass.Regexp), clientclass.Regexp)
	default:
		return fmt.Sprintf(ClientClassOptionExists, clientclass.Description)
	}
}

func (c *ClientClass6Service) List(conditions map[string]interface{}) ([]*resource.ClientClass6, error) {
	var clientClasses []*resource.ClientClass6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &clientClasses)
	}); err != nil {
		return nil, fmt.Errorf("list clientclass6 failed:%s", pg.Error(err).Error())
	}

	return clientClasses, nil
}

func (c *ClientClass6Service) Get(id string) (*resource.ClientClass6, error) {
	var clientClasses []*resource.ClientClass6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &clientClasses)
	}); err != nil {
		return nil, fmt.Errorf("get clientclass6 %s failed:%s", id, pg.Error(err).Error())
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
			resource.SqlColumnClassCondition:   clientClass.Condition,
			resource.SqlColumnClassRegexp:      clientClass.Regexp,
			resource.SqlColumnClassBeginIndex:  clientClass.BeginIndex,
			resource.SqlColumnClassDescription: clientClass.Description,
		}, map[string]interface{}{restdb.IDField: clientClass.GetID()}); err != nil {
			return pg.Error(err)
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
	return kafka.SendDHCPCmd(kafka.UpdateClientClass6,
		&pbdhcpagent.UpdateClientClass6Request{
			Name:   clientClass.Name,
			Code:   uint32(clientClass.Code),
			Regexp: genClientClass6Regexp(clientClass),
		}, nil)
}

func (c *ClientClass6Service) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if count, err := tx.CountEx(resource.TableSubnet6,
			"select count(*) from gr_subnet6 where $1::text = any(white_client_classes) or $1::text = any(black_client_classes)",
			id); err != nil {
			return pg.Error(err)
		} else if count != 0 {
			return fmt.Errorf("client class %s used by subnet6", id)
		}

		if rows, err := tx.Delete(resource.TableClientClass6,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return pg.Error(err)
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
	return kafka.SendDHCPCmd(kafka.DeleteClientClass6,
		&pbdhcpagent.DeleteClientClass6Request{Name: clientClassID}, nil)
}
