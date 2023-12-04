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

		if err := checkAddressCodeLayoutDuplicate(tx, addressCodeId, addressCodeLayout); err != nil {
			return err
		}

		addressCodeLayout.AddressCode = addressCodeId
		if _, err := tx.Insert(addressCodeLayout); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameAddressCodeLayout, string(addressCodeLayout.Label), err)
		}

		return sendCreateAddressCodeLayoutCmdToDHCPAgent(addressCode.Name, addressCodeLayout)
	})
}

func checkAddressCodeLayoutDuplicate(tx restdb.Transaction, addressCode string, addressCodeLayout *resource.AddressCodeLayout) error {
		var addressCodeLayouts []*resource.AddressCodeLayout
		if err := tx.FillEx(&addressCodeLayouts,
			"select * from gr_address_code_layout where address_code = $1 and id != $2 and (label = $3 or (begin_bit <= $4 and end_bit >= $5))",
			addressCode, addressCodeLayout.GetID(), addressCodeLayout.Label, addressCodeLayout.EndBit, addressCodeLayout.BeginBit); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(addressCodeLayout.Label), pg.Error(err).Error())
		} else if len(addressCodeLayouts) != 0 {
			return errorno.ErrConflict(errorno.ErrName(addressCodeLayout.Label), errorno.ErrName(addressCodeLayouts[0].Label), "", "")

		} else {
			return nil
		}
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
	var addressCodeLayoutSegments []*resource.AddressCodeLayoutSegment
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(conditions, &addressCodeLayouts); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery,
				string(errorno.ErrNameAddressCodeLayout), pg.Error(err).Error())
		}

		if err := tx.Fill(map[string]interface{}{resource.SqlOrderBy: "address_code_layout,code"},
			&addressCodeLayoutSegments); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery,
				string(errorno.ErrNameAddressCodeLayoutSegment), pg.Error(err).Error())
		}

		return nil
	}); err != nil {
		return nil, err
	}

	setAddressCodeLayoutSegments(addressCodeLayouts, addressCodeLayoutSegments)
	return addressCodeLayouts, nil
}

func setAddressCodeLayoutSegments(layouts []*resource.AddressCodeLayout, segments []*resource.AddressCodeLayoutSegment) {
	layoutSegments := make(map[string][]*resource.AddressCodeLayoutSegment, len(layouts))
	for _, segment := range segments {
		layoutsegments := layoutSegments[segment.AddressCodeLayout]
		layoutsegments = append(layoutsegments, segment)
		layoutSegments[segment.AddressCodeLayout] = layoutsegments
	}

	for i := range layouts {
		if segments, ok := layoutSegments[layouts[i].GetID()]; ok {
			layouts[i].Segments = segments
		}
	}
}

func (d *AddressCodeLayoutService) Get(id string) (*resource.AddressCodeLayout, error) {
	var addressCodeLayout *resource.AddressCodeLayout
	var addressCodeLayoutSegments []*resource.AddressCodeLayoutSegment
	err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		if addressCodeLayout, err = getAddressCodeLayout(tx, id); err != nil {
			return
		}

		if err = tx.Fill(map[string]interface{}{resource.SqlColumnAddressCodeLayout: id,
			resource.SqlOrderBy: resource.SqlColumnCode}, &addressCodeLayoutSegments); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
		}

		return
	})

	addressCodeLayout.Segments = addressCodeLayoutSegments
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

		oldAddressCodeLayout, err := getAddressCodeLayout(tx, addressCodeLayout.GetID())
		if err != nil {
			return err
		}

		if err := checkAddressCodeLayoutDuplicate(tx, addressCodeId, addressCodeLayout); err != nil {
			return err
		}

		if addressCodeLayout.Label == oldAddressCodeLayout.Label {
			return nil
		}

		if _, err := tx.Update(resource.TableAddressCodeLayout,
			map[string]interface{}{resource.SqlColumnLabel: addressCodeLayout.Label},
			map[string]interface{}{restdb.IDField: addressCodeLayout.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, addressCodeLayout.GetID(), pg.Error(err).Error())
		}

		return sendUpdateAddressCodeLayoutCmdToDHCPAgent(addressCode.Name, oldAddressCodeLayout, addressCodeLayout)
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
