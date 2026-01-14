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
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	ClientClassOptionExists         = `option %s exists`
	ClientClassOptionEqual          = `option %s == "%s"`
	ClientClassOptionSubstringEqual = `substring(option[%d],%d,%d) == "%s"`
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
			return util.FormatDbInsertError(errorno.ErrNameClientClass, clientClass.Name, err)
		}

		return sendCreateClientClass4CmdToAgent(clientClass)
	})
}

func sendCreateClientClass4CmdToAgent(clientClass4 *resource.ClientClass4) error {
	return kafka.SendDHCP4Cmd(kafka.CreateClientClass4,
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
			map[string]interface{}{
				resource.SqlColumnClassCondition:   clientClass.Condition,
				resource.SqlColumnClassRegexp:      clientClass.Regexp,
				resource.SqlColumnClassBeginIndex:  clientClass.BeginIndex,
				resource.SqlColumnClassDescription: clientClass.Description,
			},
			map[string]interface{}{restdb.IDField: clientClass.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, clientClass.GetID(), pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameClientClass, clientClass.GetID())
		}

		return sendUpdateClientClass4CmdToDHCPAgent(clientClass)
	})
}

func sendUpdateClientClass4CmdToDHCPAgent(clientClass *resource.ClientClass4) error {
	return kafka.SendDHCP4Cmd(kafka.UpdateClientClass4,
		&pbdhcpagent.UpdateClientClass4Request{
			Name:   clientClass.Name,
			Code:   uint32(clientClass.Code),
			Regexp: genClientClass4Regexp(clientClass),
		}, nil)
}

func (c *ClientClass4Service) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if count, err := tx.CountEx(resource.TableSubnet4,
			"select count(*) from gr_subnet4 where $1::text = any(white_client_classes) or $1::text = any(black_client_classes)",
			id); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameCount, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
		} else if count != 0 {
			return errorno.ErrBeenUsed(errorno.ErrNameClientClass, id)
		}

		if count, err := tx.CountEx(resource.TableDhcpConfig,
			"select count(*) from gr_dhcp_config where $1::text = any(subnet4_white_client_classes) or $1::text = any(subnet4_black_client_classes)",
			id); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameCount, string(errorno.ErrNameClientClass), pg.Error(err).Error())
		} else if count != 0 {
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
	return kafka.SendDHCP4Cmd(kafka.DeleteClientClass4,
		&pbdhcpagent.DeleteClientClass4Request{Name: clientClassID}, nil)
}
