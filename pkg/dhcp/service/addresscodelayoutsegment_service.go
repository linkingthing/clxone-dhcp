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

type AddressCodeLayoutSegmentService struct{}

func NewAddressCodeLayoutSegmentService() *AddressCodeLayoutSegmentService {
	return &AddressCodeLayoutSegmentService{}
}

func (d *AddressCodeLayoutSegmentService) Create(addressCodeId, layoutId string, addressCodeLayoutSegment *resource.AddressCodeLayoutSegment) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		addressCode, err := getAddressCode(tx, addressCodeId)
		if err != nil {
			return err
		}

		layout, err := getAddressCodeLayout(tx, layoutId)
		if err != nil {
			return err
		}

		if err := addressCodeLayoutSegment.Validate(layout); err != nil {
			return err
		}

		addressCodeLayoutSegment.AddressCodeLayout = layoutId
		if _, err := tx.Insert(addressCodeLayoutSegment); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameAddressCodeLayoutSegment,
				addressCodeLayoutSegment.Code, err)
		}

		return sendCreateAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode.Name, layout.Label, addressCodeLayoutSegment)
	})
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

func sendCreateAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode, layout string, addressCodeLayoutSegment *resource.AddressCodeLayoutSegment) error {
	return kafka.SendDHCPCmd(kafka.CreateAddressCodeLayoutSegment,
		&pbdhcpagent.CreateAddressCodeLayoutSegmentRequest{
			AddressCode: addressCode,
			Layout:      layout,
			Segment: &pbdhcpagent.AddressCodeLayoutSegment{
				Code:  addressCodeLayoutSegment.Code,
				Value: addressCodeLayoutSegment.Value,
			},
		},
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteAddressCodeLayoutSegment,
				&pbdhcpagent.DeleteAddressCodeLayoutSegmentRequest{
					AddressCode: addressCode,
					Layout:      layout,
					SegmentCode: addressCodeLayoutSegment.Code,
				}); err != nil {
				log.Errorf("create address code %s layout %s segment %s failed, rollback with nodes %v failed: %s",
					addressCode, layout, addressCodeLayoutSegment.Code, nodesForSucceed, err.Error())
			}
		})
}

func (d *AddressCodeLayoutSegmentService) List(layoutId string, conditions map[string]interface{}) ([]*resource.AddressCodeLayoutSegment, error) {
	conditions[resource.SqlColumnAddressCodeLayout] = layoutId
	var addressCodeLayoutSegments []*resource.AddressCodeLayoutSegment
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &addressCodeLayoutSegments)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAddressCodeLayoutSegment), pg.Error(err).Error())
	}

	return addressCodeLayoutSegments, nil
}

func (d *AddressCodeLayoutSegmentService) Get(id string) (*resource.AddressCodeLayoutSegment, error) {
	var addressCodeLayoutSegments []*resource.AddressCodeLayoutSegment
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &addressCodeLayoutSegments)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(addressCodeLayoutSegments) != 1 {
		return nil, errorno.ErrNotFound(errorno.ErrNameAddressCodeLayoutSegment, id)
	}

	return addressCodeLayoutSegments[0], nil
}

func (d *AddressCodeLayoutSegmentService) Delete(addressCodeId, layoutId, id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		addressCode, err := getAddressCode(tx, addressCodeId)
		if err != nil {
			return err
		}

		layout, err := getAddressCodeLayout(tx, layoutId)
		if err != nil {
			return err
		}

		var addressCodeLayoutSegments []*resource.AddressCodeLayoutSegment
		if err := tx.Fill(map[string]interface{}{restdb.IDField: id}, &addressCodeLayoutSegments); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
		} else if len(addressCodeLayoutSegments) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAddressCodeLayoutSegment, id)
		}

		if _, err := tx.Delete(resource.TableAddressCodeLayoutSegment,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		}

		return sendDeleteAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode.Name, layout.Label, addressCodeLayoutSegments[0])
	})
}

func sendDeleteAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode, layout string, addressCodeLayoutSegment *resource.AddressCodeLayoutSegment) error {
	return kafka.SendDHCPCmd(kafka.DeleteAddressCodeLayoutSegment,
		&pbdhcpagent.DeleteAddressCodeLayoutSegmentRequest{
			AddressCode: addressCode,
			Layout:      layout,
			SegmentCode: addressCodeLayoutSegment.Code,
		}, nil)
}

func (d *AddressCodeLayoutSegmentService) Update(addressCodeId, layoutId string, addressCodeLayoutSegment *resource.AddressCodeLayoutSegment) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		addressCode, err := getAddressCode(tx, addressCodeId)
		if err != nil {
			return err
		}

		layout, err := getAddressCodeLayout(tx, layoutId)
		if err != nil {
			return err
		}

		if err := addressCodeLayoutSegment.Validate(layout); err != nil {
			return err
		}

		var addressCodeLayoutSegments []*resource.AddressCodeLayoutSegment
		if err := tx.Fill(map[string]interface{}{restdb.IDField: addressCodeLayoutSegment.GetID()}, &addressCodeLayoutSegments); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, addressCodeLayoutSegment.GetID(), pg.Error(err).Error())
		} else if len(addressCodeLayoutSegments) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAddressCodeLayoutSegment, addressCodeLayoutSegment.GetID())
		}

		if addressCodeLayoutSegment.Code == addressCodeLayoutSegments[0].Code &&
			addressCodeLayoutSegment.Value == addressCodeLayoutSegments[0].Value {
			return nil
		}

		if _, err := tx.Update(resource.TableAddressCodeLayoutSegment,
			map[string]interface{}{
				resource.SqlColumnCode:  addressCodeLayoutSegment.Code,
				resource.SqlColumnValue: addressCodeLayoutSegment.Value,
			},
			map[string]interface{}{restdb.IDField: addressCodeLayoutSegment.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, addressCodeLayoutSegment.GetID(), pg.Error(err).Error())
		}

		return sendUpdateAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode.Name, layout.Label,
			addressCodeLayoutSegments[0], addressCodeLayoutSegment)
	})
}

func sendUpdateAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode, layout string, oldAddressCodeLayoutSegment, newAddressCodeLayoutSegment *resource.AddressCodeLayoutSegment) error {
	return kafka.SendDHCPCmd(kafka.UpdateAddressCodeLayoutSegment,
		&pbdhcpagent.UpdateAddressCodeLayoutSegmentRequest{
			AddressCode: addressCode,
			Layout:      layout,
			OldSegment: &pbdhcpagent.AddressCodeLayoutSegment{
				Code:  oldAddressCodeLayoutSegment.Code,
				Value: oldAddressCodeLayoutSegment.Value,
			},
			NewSegment: &pbdhcpagent.AddressCodeLayoutSegment{
				Code:  newAddressCodeLayoutSegment.Code,
				Value: newAddressCodeLayoutSegment.Value,
			},
		}, nil)
}
