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
	"github.com/linkingthing/clxone-dhcp/pkg/util"
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
			return util.FormatDbInsertError(errorno.ErrNameAddressCode,
				addressCode.Name, err)
		}

		return sendCreateAddressCodeCmdToDHCPAgent(addressCode)
	})
}

func sendCreateAddressCodeCmdToDHCPAgent(addressCode *resource.AddressCode) error {
	return kafka.SendDHCP6Cmd(kafka.CreateAddressCode,
		&pbdhcpagent.CreateAddressCodeRequest{Name: addressCode.Name},
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteAddressCode,
				&pbdhcpagent.DeleteAddressCodeRequest{
					Name: addressCode.Name,
				}); err != nil {
				log.Errorf("create address code %s failed, rollback with nodes %v failed: %s",
					addressCode.Name, nodesForSucceed, err.Error())
			}
		})
}

func (d *AddressCodeService) List(conditions map[string]interface{}) ([]*resource.AddressCode, error) {
	var addressCodes []*resource.AddressCode
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &addressCodes)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameAddressCode), pg.Error(err).Error())
	}

	return addressCodes, nil
}

func (d *AddressCodeService) Get(id string) (*resource.AddressCode, error) {
	var addrCode *resource.AddressCode
	err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		addrCode, err = getAddressCode(tx, id)
		return err
	})
	return addrCode, err
}

func getAddressCode(tx restdb.Transaction, addressCode string) (*resource.AddressCode, error) {
	var addressCodes []*resource.AddressCode
	if err := tx.Fill(map[string]interface{}{restdb.IDField: addressCode}, &addressCodes); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, addressCode, pg.Error(err).Error())
	} else if len(addressCodes) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameAddressCode, addressCode)
	} else {
		return addressCodes[0], nil
	}
}

func (d *AddressCodeService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(resource.TableSubnet6, map[string]interface{}{
			"address_code": id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameExists, id, pg.Error(err).Error())
		} else if exists {
			return errorno.ErrBeenUsed(errorno.ErrNameAddressCode, id)
		}

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
	return kafka.SendDHCP6Cmd(kafka.DeleteAddressCode,
		&pbdhcpagent.DeleteAddressCodeRequest{Name: addressCode.Name}, nil)
}

func (d *AddressCodeService) Update(addressCode *resource.AddressCode) error {
	if err := addressCode.Validate(); err != nil {
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
				resource.SqlColumnName:    addressCode.Name,
				resource.SqlColumnComment: addressCode.Comment,
			},
			map[string]interface{}{restdb.IDField: addressCode.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, addressCode.GetID(), pg.Error(err).Error())
		}

		if addressCode.Name != addressCodes[0].Name {
			return sendUpdateAddressCodeCmdToDHCPAgent(addressCodes[0], addressCode)
		}

		return nil
	})
}

func sendUpdateAddressCodeCmdToDHCPAgent(oldAddressCode, newAddressCode *resource.AddressCode) error {
	return kafka.SendDHCP6Cmd(kafka.UpdateAddressCode,
		&pbdhcpagent.UpdateAddressCodeRequest{
			OldName: oldAddressCode.Name,
			NewName: newAddressCode.Name,
		}, nil)
}
