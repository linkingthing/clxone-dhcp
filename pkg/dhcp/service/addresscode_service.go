package service

import (
	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type AddressCodeService struct{}

func NewAddressCodeService() *AddressCodeService {
	return &AddressCodeService{}
}

func (d *AddressCodeService) Create(addressCode *resource.AddressCode) error {
	if err := addressCode.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(addressCode); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameAddressCode), pg.Error(err).Error())
		}

		return sendCreateAddressCodeCmdToDHCPAgent(addressCode)
	})
}

func sendCreateAddressCodeCmdToDHCPAgent(addressCode *resource.AddressCode) error {
	return kafka.SendDHCPCmd(kafka.CreateAddressCode,
		&pbdhcpagent.CreateAddressCodeRequest{
			HwAddress: addressCode.HwAddress,
			Duid:      addressCode.Duid,
			Code:      addressCode.Code,
			CodeBegin: addressCode.CodeBegin,
			CodeEnd:   addressCode.CodeEnd,
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAddressCode), pg.Error(err).Error())
	}

	return addressCodes, nil
}

func (d *AddressCodeService) Get(id string) (*resource.AddressCode, error) {
	var addressCodes []*resource.AddressCode
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &addressCodes)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(addressCodes) != 1 {
		return nil, errorno.ErrNotFound(errorno.ErrNameAddressCode, id)
	}

	return addressCodes[0], nil
}

func (d *AddressCodeService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var addressCodes []*resource.AddressCode
		if err := tx.Fill(map[string]interface{}{restdb.IDField: id}, &addressCodes); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
		} else if len(addressCodes) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAddressCode, id)
		}

		if _, err := tx.Delete(resource.TableAddressCode,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		}

		return sendDeleteAddressCodeCmdToDHCPAgent(addressCodes[0])
	})
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
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var addressCodes []*resource.AddressCode
		if err := tx.Fill(map[string]interface{}{restdb.IDField: addressCode.GetID()}, &addressCodes); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, addressCode.GetID(), pg.Error(err).Error())
		} else if len(addressCodes) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAddressCode, addressCode.GetID())
		}

		if _, err := tx.Update(resource.TableAddressCode,
			map[string]interface{}{
				resource.SqlColumnCode:      addressCode.Code,
				resource.SqlColumnCodeBegin: addressCode.CodeBegin,
				resource.SqlColumnCodeEnd:   addressCode.CodeEnd,
				resource.SqlColumnComment:   addressCode.Comment,
			},
			map[string]interface{}{restdb.IDField: addressCode.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, addressCode.GetID(), pg.Error(err).Error())
		}

		if addressCode.Code != addressCodes[0].Code || addressCode.CodeBegin != addressCodes[0].CodeBegin ||
			addressCode.CodeEnd != addressCodes[0].CodeEnd {
			return sendUpdateAddressCodeCmdToDHCPAgent(addressCodes[0], addressCode)
		} else {
			return nil
		}
	})
}

func sendUpdateAddressCodeCmdToDHCPAgent(oldAddressCode, newAddressCode *resource.AddressCode) error {
	return kafka.SendDHCPCmd(kafka.UpdateAddressCode,
		&pbdhcpagent.UpdateAddressCodeRequest{
			HwAddress: oldAddressCode.HwAddress,
			Duid:      oldAddressCode.Duid,
			Code:      newAddressCode.Code,
			CodeBegin: newAddressCode.CodeBegin,
			CodeEnd:   newAddressCode.CodeEnd,
		}, nil)
}
