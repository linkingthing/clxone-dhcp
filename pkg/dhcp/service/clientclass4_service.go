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

const (
	ClientClassOptionExists         = "option %s exists"
	ClientClassOptionEqual          = "option %s == '%s'"
	ClientClassOptionSubstringEqual = "substring(option[%d],%d,%d) == '%s'"
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
			return pg.Error(err)
		}

		return sendCreateClientClass4CmdToAgent(clientClass)
	}); err != nil {
		return fmt.Errorf("create clientclass4 %s failed:%s",
			clientClass.GetID(), err.Error())
	}

	return nil
}

func sendCreateClientClass4CmdToAgent(clientClass4 *resource.ClientClass4) error {
	return kafka.SendDHCPCmd(kafka.CreateClientClass4,
		&pbdhcpagent.CreateClientClass4Request{
			Name:   clientClass4.Name,
			Code:   uint32(clientClass4.Code),
			Regexp: genClientClass4Regexp(clientClass4),
		}, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteClientClass4,
				&pbdhcpagent.DeleteClientClass4Request{Name: clientClass4.Name}); err != nil {
				log.Errorf("add clientclass4 %s failed, rollback with nodes %v failed: %s",
					clientClass4.Name, nodesForSucceed, err.Error())
			}
		})
}

func genClientClass4Regexp(clientclass *resource.ClientClass4) string {
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

func (c *ClientClass4Service) List(conditions map[string]interface{}) ([]*resource.ClientClass4, error) {
	var clientClasses []*resource.ClientClass4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &clientClasses)
	}); err != nil {
		return nil, fmt.Errorf("list clientclass4 failed:%s", pg.Error(err).Error())
	}

	return clientClasses, nil
}

func (c *ClientClass4Service) Get(id string) (*resource.ClientClass4, error) {
	var clientClasses []*resource.ClientClass4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &clientClasses)
	}); err != nil {
		return nil, fmt.Errorf("get clientclass4 of %s failed:%s", id, pg.Error(err).Error())
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
		if rows, err := tx.Update(resource.TableClientClass4,
			map[string]interface{}{
				resource.SqlColumnClassCondition:   clientClass.Condition,
				resource.SqlColumnClassRegexp:      clientClass.Regexp,
				resource.SqlColumnClassBeginIndex:  clientClass.BeginIndex,
				resource.SqlColumnClassDescription: clientClass.Description,
			},
			map[string]interface{}{restdb.IDField: clientClass.GetID()}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found clientclass4 %s", clientClass.GetID())
		}

		return sendUpdateClientClass4CmdToDHCPAgent(clientClass)
	}); err != nil {
		return fmt.Errorf("update clientclass4 %s failed:%s",
			clientClass.GetID(), err.Error())
	}

	return nil
}

func sendUpdateClientClass4CmdToDHCPAgent(clientClass *resource.ClientClass4) error {
	return kafka.SendDHCPCmd(kafka.UpdateClientClass4,
		&pbdhcpagent.UpdateClientClass4Request{
			Name:   clientClass.Name,
			Code:   uint32(clientClass.Code),
			Regexp: genClientClass4Regexp(clientClass),
		}, nil)
}

func (c *ClientClass4Service) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if count, err := tx.CountEx(resource.TableSubnet4,
			"select count(*) from gr_subnet4 where $1::text = any(white_client_classes) or $1::text = any(black_client_classes)",
			id); err != nil {
			return pg.Error(err)
		} else if count != 0 {
			return fmt.Errorf("client class %s used by subnet4", id)
		}

		if rows, err := tx.Delete(resource.TableClientClass4,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return pg.Error(err)
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
	return kafka.SendDHCPCmd(kafka.DeleteClientClass4,
		&pbdhcpagent.DeleteClientClass4Request{Name: clientClassID}, nil)
}
