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

type AdmitDuidService struct{}

func NewAdmitDuidService() *AdmitDuidService {
	return &AdmitDuidService{}
}

func (d *AdmitDuidService) Create(admitDuid *resource.AdmitDuid) error {
	admitDuid.SetID(admitDuid.Duid)
	if err := admitDuid.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitDuid); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameDuid, admitDuid.Duid, err)
		}

		return sendCreateAdmitDuidCmdToDHCPAgent(admitDuid)
	})
}

func sendCreateAdmitDuidCmdToDHCPAgent(admitDuid *resource.AdmitDuid) error {
	return kafka.SendDHCPCmd(kafka.CreateAdmitDuid, adminDuidToCreateAdmitDuidRequest(admitDuid),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteAdmitDuid,
				&pbdhcpagent.DeleteAdmitDuidRequest{Duid: admitDuid.Duid}); err != nil {
				log.Errorf("create admit duid %s failed, rollback with nodes %v failed: %s",
					admitDuid.Duid, nodesForSucceed, err.Error())
			}
		})
}

func adminDuidToCreateAdmitDuidRequest(admitDuid *resource.AdmitDuid) *pbdhcpagent.CreateAdmitDuidRequest {
	return &pbdhcpagent.CreateAdmitDuidRequest{
		Duid:       admitDuid.Duid,
		IsAdmitted: admitDuid.IsAdmitted,
	}
}

func (d *AdmitDuidService) List(conditions map[string]interface{}) ([]*resource.AdmitDuid, error) {
	var duids []*resource.AdmitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &duids)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDuid), pg.Error(err).Error())
	}

	return duids, nil
}

func (d *AdmitDuidService) Get(id string) (*resource.AdmitDuid, error) {
	var admitDuids []*resource.AdmitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &admitDuids)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(admitDuids) != 1 {
		return nil, errorno.ErrNotFound(errorno.ErrNameDuid, id)
	}

	return admitDuids[0], nil
}

func (d *AdmitDuidService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitDuid,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameDuid, id)
		}

		return sendDeleteAdmitDuidCmdToDHCPAgent(id)
	})
}

func sendDeleteAdmitDuidCmdToDHCPAgent(admitDuidId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAdmitDuid,
		&pbdhcpagent.DeleteAdmitDuidRequest{Duid: admitDuidId}, nil)
}

func (d *AdmitDuidService) Update(admitDuid *resource.AdmitDuid) error {
	if err := admitDuid.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var admitDuids []*resource.AdmitDuid
		if err := tx.Fill(map[string]interface{}{restdb.IDField: admitDuid.GetID()},
			&admitDuids); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, admitDuid.GetID(), pg.Error(err).Error())
		} else if len(admitDuids) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameDuid, admitDuid.GetID())
		}

		if _, err := tx.Update(resource.TableAdmitDuid,
			map[string]interface{}{
				resource.SqlColumnIsAdmitted: admitDuid.IsAdmitted,
				resource.SqlColumnComment:    admitDuid.Comment,
			},
			map[string]interface{}{restdb.IDField: admitDuid.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, admitDuid.GetID(), pg.Error(err).Error())
		}

		if admitDuids[0].IsAdmitted != admitDuid.IsAdmitted {
			return sendUpdateAdmitDuidCmdToDHCPAgent(admitDuid)
		} else {
			return nil
		}
	})
}

func sendUpdateAdmitDuidCmdToDHCPAgent(admitDuid *resource.AdmitDuid) error {
	return kafka.SendDHCPCmd(kafka.UpdateAdmitDuid,
		&pbdhcpagent.UpdateAdmitDuidRequest{
			Duid:       admitDuid.Duid,
			IsAdmitted: admitDuid.IsAdmitted,
		}, nil)
}

func (d *AdmitDuidService) ImportExcel(file *excel.ImportFile) (interface{}, error) {
	if len(file.Name) == 0 {
		return nil, nil
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(AdmitDuidImportFileNamePrefix, TableHeaderAdmitDuidFail, response)
	validSql, createAdmitDuidsRequest, deleteAdmitDuidsRequest, err := parseAdmitDuidsFromFile(
		file.Name, response)
	if err != nil {
		return response, err
	}

	if len(validSql) == 0 {
		return response, nil
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Exec(validSql); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert,
				string(errorno.ErrNameDuid), pg.Error(err).Error())
		}

		return sendCreateAdmitDuidsCmdToDHCPAgent(createAdmitDuidsRequest, deleteAdmitDuidsRequest)
	}); err != nil {
		return response, err
	}

	return response, nil
}

func parseAdmitDuidsFromFile(fileName string, response *excel.ImportResult) (string, *pbdhcpagent.CreateAdmitDuidsRequest, *pbdhcpagent.DeleteAdmitDuidsRequest, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return "", nil, nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return "", nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0], TableHeaderAdmitDuid, AdmitDuidMandatoryFields)
	if err != nil {
		return "", nil, nil, errorno.ErrInvalidTableHeader()
	}

	var oldAdmitDuids []*resource.AdmitDuid
	if err := db.GetResources(nil, &oldAdmitDuids); err != nil {
		return "", nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDuid), err.Error())
	}

	response.InitData(len(contents) - 1)
	duids := make([]*resource.AdmitDuid, 0, len(contents)-1)
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, AdmitDuidMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderAdmitDuidFailLen,
				localizationAdmitDuidToStrSlice(&resource.AdmitDuid{}),
				errorno.ErrMissingMandatory(j+2, AdmitDuidMandatoryFields).ErrorCN())
			continue
		}

		duid := parseAdmitDuid(tableHeaderFields, fields)
		if err := duid.Validate(); err != nil {
			addFailDataToResponse(response, TableHeaderAdmitDuidFailLen,
				localizationAdmitDuidToStrSlice(duid), errorno.TryGetErrorCNMsg(err))
		} else if err := checkAdmitDuidConflictWithAdmitDuids(duid,
			append(oldAdmitDuids, duids...)); err != nil {
			addFailDataToResponse(response, TableHeaderAdmitDuidFailLen,
				localizationAdmitDuidToStrSlice(duid), errorno.TryGetErrorCNMsg(err))
		} else {
			duids = append(duids, duid)
		}
	}

	if len(duids) == 0 {
		return "", nil, nil, nil
	}

	sql, createAdmitDuidsRequest, deleteAdmitDuidsRequest := admitDuidsToInsertSqlAndPbRequest(duids)
	return sql, createAdmitDuidsRequest, deleteAdmitDuidsRequest, nil
}

func parseAdmitDuid(tableHeaderFields, fields []string) *resource.AdmitDuid {
	duid := &resource.AdmitDuid{}
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameAdmitDuid:
			duid.Duid = strings.TrimSpace(field)
		case FieldNameIsAdmitted:
			duid.IsAdmitted = internationalizationBool(strings.TrimSpace(field))
		case FieldNameComment:
			duid.Comment = field
		}
	}

	return duid
}

func checkAdmitDuidConflictWithAdmitDuids(duid *resource.AdmitDuid, duids []*resource.AdmitDuid) error {
	for _, d := range duids {
		if d.Duid == duid.Duid {
			return errorno.ErrConflict(errorno.ErrNameDuid, errorno.ErrNameDuid,
				d.Duid, duid.Duid)
		}
	}

	return nil
}

func admitDuidsToInsertSqlAndPbRequest(duids []*resource.AdmitDuid) (string, *pbdhcpagent.CreateAdmitDuidsRequest, *pbdhcpagent.DeleteAdmitDuidsRequest) {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_admit_duid VALUES ")
	createAdmitDuidRequests := make([]*pbdhcpagent.CreateAdmitDuidRequest, 0, len(duids))
	deleteAdmitDuids := make([]string, 0, len(duids))
	for _, duid := range duids {
		buf.WriteString(admitDuidToInsertDBSqlString(duid))
		createAdmitDuidRequests = append(createAdmitDuidRequests, adminDuidToCreateAdmitDuidRequest(duid))
		deleteAdmitDuids = append(deleteAdmitDuids, duid.Duid)
	}

	return strings.TrimSuffix(buf.String(), ",") + ";",
		&pbdhcpagent.CreateAdmitDuidsRequest{Duids: createAdmitDuidRequests},
		&pbdhcpagent.DeleteAdmitDuidsRequest{Duids: deleteAdmitDuids}
}

func sendCreateAdmitDuidsCmdToDHCPAgent(createAdmitDuidsRequest *pbdhcpagent.CreateAdmitDuidsRequest, deleteAdmitDuidsRequest *pbdhcpagent.DeleteAdmitDuidsRequest) error {
	return kafka.SendDHCPCmd(kafka.CreateAdmitDuids,
		createAdmitDuidsRequest, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
				kafka.DeleteAdmitDuids, deleteAdmitDuidsRequest); err != nil {
				log.Warnf("batch create admit duids failed and rollback failed: %s", err.Error())
			}
		})
}

func (d *AdmitDuidService) ExportExcel() (interface{}, error) {
	var duids []*resource.AdmitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&duids,
			"select * from gr_admit_duid order by duid")
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDuid), pg.Error(err).Error())
	}

	strMatrix := make([][]string, 0, len(duids))
	for _, duid := range duids {
		strMatrix = append(strMatrix, localizationAdmitDuidToStrSlice(duid))
	}

	if filepath, err := excel.WriteExcelFile(AdmitDuidFileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderAdmitDuid, strMatrix); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameDuid), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (d *AdmitDuidService) ExportExcelTemplate() (*excel.ExportFile, error) {
	if filepath, err := excel.WriteExcelFile(AdmitDuidTemplateFileName,
		TableHeaderAdmitDuid, TemplateAdmitDuid); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameTemplate), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (d *AdmitDuidService) BatchDelete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Exec("delete from gr_admit_duid where id in ('" +
			strings.Join(ids, "','") + "')"); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete,
				string(errorno.ErrNameDuid), pg.Error(err).Error())
		} else if int(rows) != len(ids) {
			return errorno.ErrNotFound(errorno.ErrNameDuid, ids[0])
		}

		return sendDeleteAdmitDuidsCmdToDHCPAgent(ids)
	})
}

func sendDeleteAdmitDuidsCmdToDHCPAgent(ids []string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAdmitDuids,
		&pbdhcpagent.DeleteAdmitDuidsRequest{Duids: ids}, nil)
}
