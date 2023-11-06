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

type AddressCodeLayoutService struct{}

func NewAddressCodeLayoutService() *AddressCodeLayoutService {
	return &AddressCodeLayoutService{}
}

func (d *AddressCodeLayoutService) Create(addressCodeId string, addressCodeLayout *resource.AddressCodeLayout) error {
	if err := addressCodeLayout.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		addressCode, err := getAddressCode(tx, addressCodeId)
		if err != nil {
			return err
		}

		var addressCodeLayouts []*resource.AddressCodeLayout
		if err := tx.FillEx(&addressCodeLayouts,
			"select * from gr_address_code_layout where address_code = $1 and (label = $2 or (begin_bit <= $3 and end_bit >= $4))",
			addressCodeId, addressCodeLayout.Label, addressCodeLayout.EndBit, addressCodeLayout.BeginBit); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(addressCodeLayout.Label), pg.Error(err).Error())
		} else if len(addressCodeLayouts) != 0 {
			return errorno.ErrExistIntersection(addressCodeLayouts[0].Label, addressCodeLayout.Label)
		}

		addressCodeLayout.AddressCode = addressCodeId
		if _, err := tx.Insert(addressCodeLayout); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameAddressCodeLayout, string(addressCodeLayout.Label), err)
		}

		return sendCreateAddressCodeLayoutCmdToDHCPAgent(addressCode.Name, addressCodeLayout)
	})
}

func sendCreateAddressCodeLayoutCmdToDHCPAgent(addressCode string, addressCodeLayout *resource.AddressCodeLayout) error {
	return kafka.SendDHCP6Cmd(kafka.CreateAddressCodeLayout,
		&pbdhcpagent.CreateAddressCodeLayoutRequest{
			AddressCode: addressCode,
			Label:       string(addressCodeLayout.Label),
			Begin:       addressCodeLayout.BeginBit,
			End:         addressCodeLayout.EndBit,
		},
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteAddressCodeLayout,
				&pbdhcpagent.DeleteAddressCodeLayoutRequest{
					AddressCode: addressCode,
					Label:       string(addressCodeLayout.Label),
				}); err != nil {
				log.Errorf("create address code %s layout %s failed, rollback with nodes %v failed: %s",
					addressCode, addressCodeLayout.Label, nodesForSucceed, err.Error())
			}
		})
}

func (d *AddressCodeLayoutService) List(addressCodeId string, conditions map[string]interface{}) ([]*resource.AddressCodeLayout, error) {
	conditions[resource.SqlColumnAddressCode] = addressCodeId
	var addressCodeLayouts []*resource.AddressCodeLayout
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &addressCodeLayouts)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAddressCodeLayout), pg.Error(err).Error())
	}

	return addressCodeLayouts, nil
}

func (d *AddressCodeLayoutService) Get(id string) (*resource.AddressCodeLayout, error) {
	var addressCodeLayout *resource.AddressCodeLayout
	err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		addressCodeLayout, err = getAddressCodeLayout(tx, id)
		return
	})
	return addressCodeLayout, err
}

func getAddressCodeLayout(tx restdb.Transaction, layout string) (*resource.AddressCodeLayout, error) {
	var addressCodeLayouts []*resource.AddressCodeLayout
	if err := tx.Fill(map[string]interface{}{restdb.IDField: layout}, &addressCodeLayouts); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, layout, pg.Error(err).Error())
	} else if len(addressCodeLayouts) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameAddressCode, layout)
	} else {
		return addressCodeLayouts[0], nil
	}
}

func (d *AddressCodeLayoutService) Delete(addressCodeId, id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		addressCode, err := getAddressCode(tx, addressCodeId)
		if err != nil {
			return err
		}

		var addressCodeLayouts []*resource.AddressCodeLayout
		if err := tx.Fill(map[string]interface{}{restdb.IDField: id}, &addressCodeLayouts); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
		} else if len(addressCodeLayouts) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAddressCodeLayout, id)
		}

		if _, err := tx.Delete(resource.TableAddressCodeLayout,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		}

		return sendDeleteAddressCodeLayoutCmdToDHCPAgent(addressCode.Name, addressCodeLayouts[0])
	})
}

func sendDeleteAddressCodeLayoutCmdToDHCPAgent(addressCode string, addressCodeLayout *resource.AddressCodeLayout) error {
	return kafka.SendDHCP6Cmd(kafka.DeleteAddressCodeLayout,
		&pbdhcpagent.DeleteAddressCodeLayoutRequest{
			AddressCode: addressCode,
			Label:       string(addressCodeLayout.Label),
		}, nil)
}

func (d *AddressCodeLayoutService) Update(addressCodeId string, addressCodeLayout *resource.AddressCodeLayout) error {
	if err := addressCodeLayout.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		addressCode, err := getAddressCode(tx, addressCodeId)
		if err != nil {
			return err
		}

		var addressCodeLayouts []*resource.AddressCodeLayout
		if err := tx.Fill(map[string]interface{}{restdb.IDField: addressCodeLayout.GetID()}, &addressCodeLayouts); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, addressCodeLayout.GetID(), pg.Error(err).Error())
		} else if len(addressCodeLayouts) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAddressCodeLayout, addressCodeLayout.GetID())
		}

		if addressCodeLayout.Label == addressCodeLayouts[0].Label {
			return nil
		}

		if _, err := tx.Update(resource.TableAddressCodeLayout,
			map[string]interface{}{resource.SqlColumnLabel: addressCodeLayout.Label},
			map[string]interface{}{restdb.IDField: addressCodeLayout.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, addressCodeLayout.GetID(), pg.Error(err).Error())
		}

		return sendUpdateAddressCodeLayoutCmdToDHCPAgent(addressCode.Name, addressCodeLayouts[0], addressCodeLayout)
	})
}

func sendUpdateAddressCodeLayoutCmdToDHCPAgent(addressCode string, oldAddressCodeLayout, newAddressCodeLayout *resource.AddressCodeLayout) error {
	return kafka.SendDHCP6Cmd(kafka.UpdateAddressCodeLayout,
		&pbdhcpagent.UpdateAddressCodeLayoutRequest{
			AddressCode: addressCode,
			OldLabel:    string(oldAddressCodeLayout.Label),
			NewLabel:    string(newAddressCodeLayout.Label),
		}, nil)
}
