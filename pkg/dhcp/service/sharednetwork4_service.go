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

type SharedNetwork4Service struct{}

func NewSharedNetwork4Service() *SharedNetwork4Service {
	return &SharedNetwork4Service{}
}

func (s *SharedNetwork4Service) Create(sharedNetwork4 *resource.SharedNetwork4) error {
	if err := sharedNetwork4.Validate(); err != nil {
		return fmt.Errorf("validate shared network4 params invalid: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(sharedNetwork4); err != nil {
			return pg.Error(err)
		}

		return sendCreateSharedNetwork4CmdToDHCPAgent(sharedNetwork4)
	}); err != nil {
		return fmt.Errorf("create shared network4 %s failed: %s",
			sharedNetwork4.Name, err.Error())
	}

	return nil
}

func sendCreateSharedNetwork4CmdToDHCPAgent(sharedNetwork4 *resource.SharedNetwork4) error {
	return kafka.SendDHCPCmd(kafka.CreateSharedNetwork4,
		sharedNetwork4ToCreateSharedNetwork4Request(sharedNetwork4),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteSharedNetwork4,
				sharedNetworkNameToDeleteSharedNetwork4Request(sharedNetwork4.Name)); err != nil {
				log.Errorf("create shared network4 %s failed, and rollback with nodes %v failed: %s",
					sharedNetwork4.Name, nodesForSucceed, err.Error())
			}
		})
}

func sharedNetwork4ToCreateSharedNetwork4Request(sharedNetwork4 *resource.SharedNetwork4) *pbdhcpagent.CreateSharedNetwork4Request {
	return &pbdhcpagent.CreateSharedNetwork4Request{
		Name:      sharedNetwork4.Name,
		SubnetIds: sharedNetwork4.SubnetIds,
	}
}

func (s *SharedNetwork4Service) List(condition map[string]interface{}) ([]*resource.SharedNetwork4, error) {
	var sharedNetwork4s []*resource.SharedNetwork4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(condition, &sharedNetwork4s)
	}); err != nil {
		return nil, fmt.Errorf("list shared network4s from db failed: %s", pg.Error(err).Error())
	}

	return sharedNetwork4s, nil
}

func (s *SharedNetwork4Service) Get(id string) (sharedNetwork4 *resource.SharedNetwork4, err error) {
	err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		sharedNetwork4, err = getOldSharedNetwork(tx, id)
		return pg.Error(err)
	})
	return
}

func (s *SharedNetwork4Service) Update(sharedNetwork4 *resource.SharedNetwork4) error {
	if err := sharedNetwork4.Validate(); err != nil {
		return fmt.Errorf("validate shared network4 params invalid: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		oldSharedNetwork4, err := getOldSharedNetwork(tx, sharedNetwork4.GetID())
		if err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSharedNetwork4, map[string]interface{}{
			resource.SqlColumnName:       sharedNetwork4.Name,
			resource.SqlColumnsSubnetIds: sharedNetwork4.SubnetIds,
			resource.SqlColumnsSubnets:   sharedNetwork4.Subnets,
			resource.SqlColumnComment:    sharedNetwork4.Comment,
		}, map[string]interface{}{
			restdb.IDField: sharedNetwork4.GetID()}); err != nil {
			return pg.Error(err)
		}

		return sendUpdateSharedNetwork4CmdToDHCPAgent(oldSharedNetwork4.Name, sharedNetwork4)
	}); err != nil {
		return fmt.Errorf("update sharenetwork4 %s failed:%s",
			sharedNetwork4.GetID(), err.Error())
	}

	return nil
}

func sendUpdateSharedNetwork4CmdToDHCPAgent(name string, sharedNetwork4 *resource.SharedNetwork4) error {
	return kafka.SendDHCPCmd(kafka.UpdateSharedNetwork4,
		&pbdhcpagent.UpdateSharedNetwork4Request{
			Old: sharedNetworkNameToDeleteSharedNetwork4Request(name),
			New: sharedNetwork4ToCreateSharedNetwork4Request(sharedNetwork4),
		}, nil)
}

func (s *SharedNetwork4Service) Delete(sharedNetwork4Id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		oldSharedNetwork4, err := getOldSharedNetwork(tx, sharedNetwork4Id)
		if err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableSharedNetwork4, map[string]interface{}{
			restdb.IDField: sharedNetwork4Id}); err != nil {
			return pg.Error(err)
		}

		return sendDeleteSharedNetwork4CmdToDHCPAgent(oldSharedNetwork4.Name)
	}); err != nil {
		return fmt.Errorf("delete shared network4 %s failed: %s", sharedNetwork4Id, err.Error())
	}

	return nil
}

func getOldSharedNetwork(tx restdb.Transaction, id string) (*resource.SharedNetwork4, error) {
	var sharedNetworks []*resource.SharedNetwork4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: id},
		&sharedNetworks); err != nil {
		return nil, pg.Error(err)
	} else if len(sharedNetworks) == 0 {
		return nil, fmt.Errorf("no found shared network4 %s", id)
	}

	return sharedNetworks[0], nil
}

func sendDeleteSharedNetwork4CmdToDHCPAgent(name string) error {
	return kafka.SendDHCPCmd(kafka.DeleteSharedNetwork4,
		sharedNetworkNameToDeleteSharedNetwork4Request(name), nil)
}

func sharedNetworkNameToDeleteSharedNetwork4Request(name string) *pbdhcpagent.DeleteSharedNetwork4Request {
	return &pbdhcpagent.DeleteSharedNetwork4Request{Name: name}
}

func checkUsedBySharedNetwork(tx restdb.Transaction, subnetId uint64) error {
	var sharedNetwork4s []*resource.SharedNetwork4
	if err := tx.FillEx(&sharedNetwork4s,
		"select * from gr_shared_network4 where $1::numeric = any(subnet_ids)",
		subnetId); err != nil {
		return fmt.Errorf("check subnet4 is used by shared network4 failed: %s", pg.Error(err).Error())
	} else if len(sharedNetwork4s) != 0 {
		return fmt.Errorf("subnet4 used by shared network4 %s", sharedNetwork4s[0].Name)
	} else {
		return nil
	}
}
