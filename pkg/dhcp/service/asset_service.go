package service

import (
	"bytes"
	"strings"
	"time"

	"github.com/linkingthing/cement/log"
	"github.com/linkingthing/clxone-utils/excel"
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
	AssetFileNamePrefix       = "dhcp-asset-"
	AssetTemplateFileName     = "dhcp-asset-template"
	AssetImportFileNamePrefix = "dhcp-asset-import"
)

type AssetService struct{}

func NewAssetService() *AssetService {
	return &AssetService{}
}

func (a *AssetService) Create(asset *resource.Asset) error {
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
		assetToPbCreateAssetRequest(asset),
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

func assetToPbCreateAssetRequest(asset *resource.Asset) *pbdhcpagent.CreateAssetRequest {
	return &pbdhcpagent.CreateAssetRequest{
		HwAddress:         asset.HwAddress,
		AssetType:         asset.AssetType,
		Manufacturer:      asset.Manufacturer,
		Model:             asset.Model,
		AccessNetworkTime: asset.AccessNetworkTime,
	}
}

func (a *AssetService) List(conditions map[string]interface{}) ([]*resource.Asset, error) {
	var assets []*resource.Asset
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &assets)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameAsset), pg.Error(err).Error())
	}

	return assets, nil
}

func (a *AssetService) Get(id string) (*resource.Asset, error) {
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

func (a *AssetService) Delete(id string) error {
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

func (a *AssetService) Update(asset *resource.Asset) error {
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

		if assets[0].Diff(asset) {
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

func (a *AssetService) ImportExcel(file *excel.ImportFile) (interface{}, error) {
	if len(file.Name) == 0 {
		return nil, nil
	}

	sentryNodes, _, _, err := kafka.GetDHCPNodes(kafka.AgentStack6)
	if err != nil {
		return nil, err
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(AssetImportFileNamePrefix, TableHeaderAssetFail, response)
	validSql, createAssetsRequest, deleteAssetsRequest, err := parseAssetsFromFile(file.Name, response)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Exec(validSql); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameAsset), pg.Error(err).Error())
		}

		return sendCreateAssetsCmdToDHCPAgent(sentryNodes, createAssetsRequest, deleteAssetsRequest)
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func parseAssetsFromFile(fileName string, response *excel.ImportResult) (string, *pbdhcpagent.CreateAssetsRequest, *pbdhcpagent.DeleteAssetsRequest, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return "", nil, nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return "", nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0],
		TableHeaderAsset, AssetMandatoryFields)
	if err != nil {
		return "", nil, nil, errorno.ErrInvalidTableHeader()
	}

	var oldAssets []*resource.Asset
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{resource.SqlOrderBy: resource.SqlColumnHwAddress}, &oldAssets)
	}); err != nil {
		return "", nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAsset), pg.Error(err).Error())
	}

	response.InitData(len(contents) - 1)
	assets := make([]*resource.Asset, 0, len(contents)-1)
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, AssetMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderAssetFailLen,
				localizationAssetToStrSlice(&resource.Asset{}),
				errorno.ErrMissingMandatory(j+2, SubnetMandatoryFields).ErrorCN())
			continue
		}

		asset := parseAsset(tableHeaderFields, fields)
		if err := asset.Validate(); err != nil {
			addFailDataToResponse(response, TableHeaderAssetFailLen, localizationAssetToStrSlice(asset),
				errorno.TryGetErrorCNMsg(err))
		} else if err := checkAssetConflictWithAssets(asset, append(oldAssets, assets...)); err != nil {
			addFailDataToResponse(response, TableHeaderAssetFailLen, localizationAssetToStrSlice(asset),
				errorno.TryGetErrorCNMsg(err))
		} else {
			assets = append(assets, asset)
		}
	}

	if len(assets) == 0 {
		return "", nil, nil, nil
	}

	sql, createAssetsRequest, deleteAssetsRequest := assetToInsertSqlAndPbRequest(assets)
	return sql, createAssetsRequest, deleteAssetsRequest, nil
}

func parseAsset(tableHeaderFields, fields []string) *resource.Asset {
	asset := &resource.Asset{}
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameAssetName:
			asset.Name = strings.TrimSpace(field)
		case FieldNameHwAddress:
			asset.HwAddress = strings.TrimSpace(field)
		case FieldNameAssetType:
			asset.AssetType = strings.TrimSpace(field)
		case FieldNameManufacturer:
			asset.Manufacturer = strings.TrimSpace(field)
		case FieldNameModel:
			asset.Model = strings.TrimSpace(field)
		case FieldNameAccessNetworkTime:
			asset.AccessNetworkTime = strings.TrimSpace(field)
		}
	}

	return asset
}

func checkAssetConflictWithAssets(asset *resource.Asset, assets []*resource.Asset) error {
	for _, a := range assets {
		if a.HwAddress == asset.HwAddress {
			return errorno.ErrConflict(errorno.ErrNameAsset, errorno.ErrNameAsset,
				a.HwAddress, asset.HwAddress)
		}
	}

	return nil
}

func assetToInsertSqlAndPbRequest(assets []*resource.Asset) (string, *pbdhcpagent.CreateAssetsRequest, *pbdhcpagent.DeleteAssetsRequest) {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_asset VALUES ")
	createAssetRequests := make([]*pbdhcpagent.CreateAssetRequest, 0, len(assets))
	hwAddresses := make([]string, 0, len(assets))
	for _, asset := range assets {
		buf.WriteString(assetToInsertDBSqlString(asset))
		createAssetRequests = append(createAssetRequests, assetToPbCreateAssetRequest(asset))
		hwAddresses = append(hwAddresses, asset.HwAddress)
	}

	return strings.TrimSuffix(buf.String(), ",") + ";",
		&pbdhcpagent.CreateAssetsRequest{Assets: createAssetRequests},
		&pbdhcpagent.DeleteAssetsRequest{HwAddresses: hwAddresses}
}

func sendCreateAssetsCmdToDHCPAgent(nodes []string, createAssetsRequest *pbdhcpagent.CreateAssetsRequest, deleteAssetsRequest *pbdhcpagent.DeleteAssetsRequest) error {
	if len(nodes) == 0 {
		return nil
	}

	succeedSentryNodes := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			[]string{node}, kafka.CreateAssets, createAssetsRequest); err != nil {
			if err := sendDeleteAssetsCmdToDHCPAgent(succeedSentryNodes, deleteAssetsRequest); err != nil {
				log.Warnf("batch create assets failed and rollback failed: %s", err.Error())
			}
			return err
		}

		succeedSentryNodes = append(succeedSentryNodes, node)
	}

	return nil
}

func (a *AssetService) ExportExcel() (interface{}, error) {
	var assets []*resource.Asset
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{resource.SqlOrderBy: resource.SqlColumnHwAddress}, &assets)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAsset), pg.Error(err).Error())
	}

	strMatrix := make([][]string, 0, len(assets))
	for _, asset := range assets {
		strMatrix = append(strMatrix, localizationAssetToStrSlice(asset))
	}

	if filepath, err := excel.WriteExcelFile(AssetFileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderAsset, strMatrix); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport, string(errorno.ErrNameAsset), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (a *AssetService) ExportExcelTemplate() (interface{}, error) {
	if filepath, err := excel.WriteExcelFile(AssetTemplateFileName,
		TableHeaderAsset, TemplateAsset); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport, string(errorno.ErrNameTemplate), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (a *AssetService) BatchDelete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	sentryNodes, _, _, err := kafka.GetDHCPNodes(kafka.AgentStack6)
	if err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Exec("delete from gr_asset where id in ('" +
			strings.Join(ids, "','") + "')"); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, string(errorno.ErrNameAsset), pg.Error(err).Error())
		} else if int(rows) != len(ids) {
			return errorno.ErrNotFound(errorno.ErrNameAsset, ids[0])
		} else {
			return sendDeleteAssetsCmdToDHCPAgent(sentryNodes, &pbdhcpagent.DeleteAssetsRequest{HwAddresses: ids})
		}
	})
}

func sendDeleteAssetsCmdToDHCPAgent(nodes []string, deleteAssetsRequest *pbdhcpagent.DeleteAssetsRequest) error {
	_, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodes,
		kafka.DeleteAssets, deleteAssetsRequest)
	return err
}
