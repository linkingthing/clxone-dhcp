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

type AssetService struct{}

func NewAssetService() *AssetService {
	return &AssetService{}
}

func (d *AssetService) Create(asset *resource.Asset) error {
	if err := asset.Validate(); err != nil {
		return err
	}

	asset.SetID(asset.HwAddress)
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(asset); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameMac,
				asset.HwAddress, err)
		}

		return sendCreateAssetCmdToDHCPAgent(asset)
	})
}

func sendCreateAssetCmdToDHCPAgent(asset *resource.Asset) error {
	return kafka.SendDHCPCmd(kafka.CreateAsset,
		&pbdhcpagent.CreateAssetRequest{
			HwAddress:         asset.HwAddress,
			AssetType:         asset.AssetType,
			Manufacturer:      asset.Manufacturer,
			Model:             asset.Model,
			AccessNetworkTime: asset.AccessNetworkTime,
		},
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteAsset,
				&pbdhcpagent.DeleteAssetRequest{
					HwAddress: asset.HwAddress,
				}); err != nil {
				log.Errorf("create asset %s failed, rollback with nodes %v failed: %s",
					asset.HwAddress, nodesForSucceed, err.Error())
			}
		})
}

func (d *AssetService) List(conditions map[string]interface{}) ([]*resource.Asset, error) {
	var assets []*resource.Asset
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &assets)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameAsset), pg.Error(err).Error())
	}

	return assets, nil
}

func (d *AssetService) Get(id string) (*resource.Asset, error) {
	var assets []*resource.Asset
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &assets)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(assets) != 1 {
		return nil, errorno.ErrNotFound(errorno.ErrNameAsset, id)
	}

	return assets[0], nil
}

func (d *AssetService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAsset,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAsset, id)
		}

		return sendDeleteAssetCmdToDHCPAgent(id)
	})
}

func sendDeleteAssetCmdToDHCPAgent(hwAddress string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAsset,
		&pbdhcpagent.DeleteAssetRequest{
			HwAddress: hwAddress,
		}, nil)
}

func (d *AssetService) Update(asset *resource.Asset) error {
	if err := asset.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var assets []*resource.Asset
		if err := tx.Fill(map[string]interface{}{restdb.IDField: asset.GetID()}, &assets); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, asset.GetID(), pg.Error(err).Error())
		} else if len(assets) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAsset, asset.HwAddress)
		}

		if _, err := tx.Update(resource.TableAsset,
			map[string]interface{}{
				resource.SqlColumnName:              asset.Name,
				resource.SqlColumnAssetType:         asset.AssetType,
				resource.SqlColumnManufacturer:      asset.Manufacturer,
				resource.SqlColumnModel:             asset.Model,
				resource.SqlColumnAccessNetworkTime: asset.AccessNetworkTime,
			},
			map[string]interface{}{restdb.IDField: asset.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, asset.GetID(), pg.Error(err).Error())
		}

		if !assets[0].Equal(asset) {
			return sendUpdateAssetCmdToDHCPAgent(asset)
		}

		return nil
	})
}

func sendUpdateAssetCmdToDHCPAgent(asset *resource.Asset) error {
	return kafka.SendDHCPCmd(kafka.UpdateAsset,
		&pbdhcpagent.UpdateAssetRequest{
			HwAddress:         asset.HwAddress,
			AssetType:         asset.AssetType,
			Manufacturer:      asset.Manufacturer,
			Model:             asset.Model,
			AccessNetworkTime: asset.AccessNetworkTime,
		}, nil)
}
