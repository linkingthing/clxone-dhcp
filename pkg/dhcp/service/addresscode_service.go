package service

import (
	"fmt"

	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type AddressCodeService struct{}

func NewAddressCodeService() *AddressCodeService {
	return &AddressCodeService{}
}

func (d *AddressCodeService) Create(addressCode *resource.AddressCode) error {
	if err := addressCode.Validate(); err != nil {
		return fmt.Errorf("validate address code %s %s failed:%s",
			addressCode.HwAddress, addressCode.Duid, err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(addressCode); err != nil {
			return pg.Error(err)
		}

		return sendCreateAddressCodeCmdToDHCPAgent(addressCode)
	}); err != nil {
		return fmt.Errorf("create address code %s %s failed: %s",
			addressCode.HwAddress, addressCode.Duid, err.Error())
	}

	return nil
}

func sendCreateAddressCodeCmdToDHCPAgent(addressCode *resource.AddressCode) error {
	return kafka.SendDHCPCmd(kafka.CreateAddressCode,
		&pbdhcpagent.CreateAddressCodeRequest{
			HwAddress: addressCode.HwAddress,
			Duid:      addressCode.Duid,
			Code:      addressCode.Code,
			Begin:     addressCode.Begin,
			End:       addressCode.End,
		},
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteAddressCode,
				&pbdhcpagent.DeleteAddressCodeRequest{
					HwAddress: addressCode.HwAddress,
					Duid:      addressCode.Duid,
				}); err != nil {
				log.Errorf("create address code %s %s failed, rollback with nodes %v failed: %s",
					addressCode.HwAddress, addressCode.Duid, nodesForSucceed, err.Error())
			}
		})
}

func (d *AddressCodeService) List(conditions map[string]interface{}) ([]*resource.AddressCode, error) {
	var addressCodes []*resource.AddressCode
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &addressCodes)
	}); err != nil {
		return nil, fmt.Errorf("list address codes failed:%s", pg.Error(err).Error())
	}

	return addressCodes, nil
}

func (d *AddressCodeService) Get(id string) (*resource.AddressCode, error) {
	var addressCodes []*resource.AddressCode
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &addressCodes)
	}); err != nil {
		return nil, fmt.Errorf("get address code %s failed:%s", id, pg.Error(err).Error())
	} else if len(addressCodes) != 1 {
		return nil, fmt.Errorf("no found address code %s", id)
	}

	return addressCodes[0], nil
}

func (d *AddressCodeService) Delete(id string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var addressCodes []*resource.AddressCode
		if err := tx.Fill(map[string]interface{}{restdb.IDField: id}, &addressCodes); err != nil {
			return pg.Error(err)
		} else if len(addressCodes) == 0 {
			return fmt.Errorf("no found address code %s", id)
		}

		if _, err := tx.Delete(resource.TableAddressCode,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return pg.Error(err)
		}

		return sendDeleteAddressCodeCmdToDHCPAgent(addressCodes[0])
	}); err != nil {
		return fmt.Errorf("delete address code %s failed:%s", id, err.Error())
	}

	return nil
}

func sendDeleteAddressCodeCmdToDHCPAgent(addressCode *resource.AddressCode) error {
	return kafka.SendDHCPCmd(kafka.DeleteAddressCode,
		&pbdhcpagent.DeleteAddressCodeRequest{
			HwAddress: addressCode.HwAddress,
			Duid:      addressCode.Duid,
		}, nil)
}

func (d *AddressCodeService) Update(addressCode *resource.AddressCode) error {
	if err := addressCode.ValidateCode(); err != nil {
		return fmt.Errorf("validate address code %s failed: %s", addressCode.GetID(), err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var addressCodes []*resource.AddressCode
		if err := tx.Fill(map[string]interface{}{restdb.IDField: addressCode.GetID()}, &addressCodes); err != nil {
			return pg.Error(err)
		} else if len(addressCodes) == 0 {
			return fmt.Errorf("no found address code %s", addressCode.GetID())
		}

		if _, err := tx.Update(resource.TableAddressCode,
			map[string]interface{}{
				resource.SqlColumnCode:    addressCode.Code,
				resource.SqlColumnBegin:   addressCode.Begin,
				resource.SqlColumnEnd:     addressCode.End,
				resource.SqlColumnComment: addressCode.Comment,
			},
			map[string]interface{}{restdb.IDField: addressCode.GetID()}); err != nil {
			return pg.Error(err)
		}

		if addressCode.Code != addressCodes[0].Code || addressCode.Begin != addressCodes[0].Begin ||
			addressCode.End != addressCodes[0].End {
			return sendUpdateAddressCodeCmdToDHCPAgent(addressCodes[0], addressCode)
		} else {
			return nil
		}
	}); err != nil {
		return fmt.Errorf("update address code %s failed:%s", addressCode.GetID(), err.Error())
	}

	return nil
}

func sendUpdateAddressCodeCmdToDHCPAgent(oldAddressCode, newAddressCode *resource.AddressCode) error {
	return kafka.SendDHCPCmd(kafka.UpdateAddressCode,
		&pbdhcpagent.UpdateAddressCodeRequest{
			HwAddress: oldAddressCode.HwAddress,
			Duid:      oldAddressCode.Duid,
			Code:      newAddressCode.Code,
			Begin:     newAddressCode.Begin,
			End:       newAddressCode.End,
		}, nil)
}
