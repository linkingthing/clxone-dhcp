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

type AdmitFingerprintService struct{}

func NewAdmitFingerprintService() *AdmitFingerprintService {
	return &AdmitFingerprintService{}
}

func (f *AdmitFingerprintService) Create(admitFingerprint *resource.AdmitFingerprint) error {
	admitFingerprint.SetID(admitFingerprint.ClientType)
	if err := admitFingerprint.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitFingerprint); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameFingerprint, admitFingerprint.ClientType, err)
		}

		return sendCreateAdmitFingerprintCmdToDHCPAgent(admitFingerprint)
	})
}

func sendCreateAdmitFingerprintCmdToDHCPAgent(admitFingerprint *resource.AdmitFingerprint) error {
	return kafka.SendDHCPCmd(kafka.CreateAdmitFingerprint, admitFingerprintToCreateAdmitFingerprintRequest(admitFingerprint),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteAdmitFingerprint,
				&pbdhcpagent.DeleteAdmitFingerprintRequest{
					ClientType: admitFingerprint.ClientType,
				}); err != nil {
				log.Errorf("create admit fingerprint %s failed, rollback with nodes %v failed: %s",
					admitFingerprint.ClientType, nodesForSucceed, err.Error())
			}
		})
}

func admitFingerprintToCreateAdmitFingerprintRequest(admitFingerprint *resource.AdmitFingerprint) *pbdhcpagent.CreateAdmitFingerprintRequest {
	return &pbdhcpagent.CreateAdmitFingerprintRequest{
		ClientType: admitFingerprint.ClientType,
		IsAdmitted: admitFingerprint.IsAdmitted,
	}
}

func (f *AdmitFingerprintService) List() ([]*resource.AdmitFingerprint, error) {
	var fingerprints []*resource.AdmitFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{
			resource.SqlOrderBy: resource.SqlColumnClientType}, &fingerprints)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAdmit), pg.Error(err).Error())
	}

	return fingerprints, nil
}

func (f *AdmitFingerprintService) Get(id string) (*resource.AdmitFingerprint, error) {
	var admitFingerprints []*resource.AdmitFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &admitFingerprints)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(admitFingerprints) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameAdmit, id)
	}

	return admitFingerprints[0], nil
}

func (f *AdmitFingerprintService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitFingerprint,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAdmit, id)
		}

		return sendDeleteAdmitFingerprintCmdToDHCPAgent(id)
	})
}

func sendDeleteAdmitFingerprintCmdToDHCPAgent(admitFingerprintId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAdmitFingerprint,
		&pbdhcpagent.DeleteAdmitFingerprintRequest{ClientType: admitFingerprintId}, nil)
}

func (f *AdmitFingerprintService) Update(admitFingerprint *resource.AdmitFingerprint) error {
	if err := admitFingerprint.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var admitFingerprints []*resource.AdmitFingerprint
		if err := tx.Fill(map[string]interface{}{restdb.IDField: admitFingerprint.GetID()},
			&admitFingerprints); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, admitFingerprint.GetID(), pg.Error(err).Error())
		} else if len(admitFingerprints) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAdmit, admitFingerprint.GetID())
		}

		if _, err := tx.Update(resource.TableAdmitFingerprint,
			map[string]interface{}{
				resource.SqlColumnIsAdmitted: admitFingerprint.IsAdmitted,
				resource.SqlColumnComment:    admitFingerprint.Comment,
			},
			map[string]interface{}{restdb.IDField: admitFingerprint.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, admitFingerprint.GetID(), pg.Error(err).Error())
		}

		if admitFingerprints[0].IsAdmitted != admitFingerprint.IsAdmitted {
			return sendUpdateAdmitFingerprintCmdToDHCPAgent(admitFingerprint)
		} else {
			return nil
		}
	})
}

func sendUpdateAdmitFingerprintCmdToDHCPAgent(admitFingerprint *resource.AdmitFingerprint) error {
	return kafka.SendDHCPCmd(kafka.UpdateAdmitFingerprint,
		&pbdhcpagent.UpdateAdmitFingerprintRequest{
			ClientType: admitFingerprint.ClientType,
			IsAdmitted: admitFingerprint.IsAdmitted,
		}, nil)
}

func (f *AdmitFingerprintService) ImportExcel(file *excel.ImportFile) (interface{}, error) {
	if len(file.Name) == 0 {
		return nil, nil
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(AdmitFingerprintImportFileNamePrefix, TableHeaderAdmitFingerprintFail, response)
	validSql, createAdmitFingerprintsRequest, deleteAdmitFingerprintsRequest, err := parseAdmitFingerprintsFromFile(
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
				string(errorno.ErrNameFingerprint), pg.Error(err).Error())
		}

		return sendCreateAdmitFingerprintsCmdToDHCPAgent(createAdmitFingerprintsRequest, deleteAdmitFingerprintsRequest)
	}); err != nil {
		return response, err
	}

	return response, nil
}

func parseAdmitFingerprintsFromFile(fileName string, response *excel.ImportResult) (string, *pbdhcpagent.CreateAdmitFingerprintsRequest, *pbdhcpagent.DeleteAdmitFingerprintsRequest, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return "", nil, nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return "", nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0], TableHeaderAdmitFingerprint, AdmitFingerprintMandatoryFields)
	if err != nil {
		return "", nil, nil, errorno.ErrInvalidTableHeader()
	}

	var oldAdmitFingerprints []*resource.AdmitFingerprint
	if err := db.GetResources(nil, &oldAdmitFingerprints); err != nil {
		return "", nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameFingerprint), err.Error())
	}

	response.InitData(len(contents) - 1)
	fingerprints := make([]*resource.AdmitFingerprint, 0, len(contents)-1)
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, AdmitFingerprintMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderAdmitFingerprintFailLen,
				localizationAdmitFingerprintToStrSlice(&resource.AdmitFingerprint{}),
				errorno.ErrMissingMandatory(j+2, AdmitFingerprintMandatoryFields).ErrorCN())
			continue
		}

		fingerprint := parseAdmitFingerprint(tableHeaderFields, fields)
		if err := fingerprint.Validate(); err != nil {
			addFailDataToResponse(response, TableHeaderAdmitFingerprintFailLen,
				localizationAdmitFingerprintToStrSlice(fingerprint), errorno.TryGetErrorCNMsg(err))
		} else if err := checkAdmitFingerprintConflictWithAdmitFingerprints(fingerprint,
			append(oldAdmitFingerprints, fingerprints...)); err != nil {
			addFailDataToResponse(response, TableHeaderAdmitFingerprintFailLen,
				localizationAdmitFingerprintToStrSlice(fingerprint), errorno.TryGetErrorCNMsg(err))
		} else {
			fingerprints = append(fingerprints, fingerprint)
		}
	}

	if len(fingerprints) == 0 {
		return "", nil, nil, nil
	}

	sql, createAdmitFingerprintsRequest, deleteAdmitFingerprintsRequest := admitFingerprintsToInsertSqlAndPbRequest(fingerprints)
	return sql, createAdmitFingerprintsRequest, deleteAdmitFingerprintsRequest, nil
}

func parseAdmitFingerprint(tableHeaderFields, fields []string) *resource.AdmitFingerprint {
	fingerprint := &resource.AdmitFingerprint{}
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameAdmitClientType:
			fingerprint.ClientType = strings.TrimSpace(field)
		case FieldNameIsAdmitted:
			fingerprint.IsAdmitted = internationalizationBool(strings.TrimSpace(field))
		case FieldNameComment:
			fingerprint.Comment = field
		}
	}

	return fingerprint
}

func checkAdmitFingerprintConflictWithAdmitFingerprints(fingerprint *resource.AdmitFingerprint, fingerprints []*resource.AdmitFingerprint) error {
	for _, f := range fingerprints {
		if f.ClientType == fingerprint.ClientType {
			return errorno.ErrConflict(errorno.ErrNameFingerprint, errorno.ErrNameFingerprint,
				f.ClientType, fingerprint.ClientType)
		}
	}

	return nil
}

func admitFingerprintsToInsertSqlAndPbRequest(fingerprints []*resource.AdmitFingerprint) (string, *pbdhcpagent.CreateAdmitFingerprintsRequest, *pbdhcpagent.DeleteAdmitFingerprintsRequest) {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_admit_fingerprint VALUES ")
	createAdmitFingerprintRequests := make([]*pbdhcpagent.CreateAdmitFingerprintRequest, 0, len(fingerprints))
	deleteAdmitFingerprints := make([]string, 0, len(fingerprints))
	for _, fingerprint := range fingerprints {
		buf.WriteString(admitFingerprintToInsertDBSqlString(fingerprint))
		createAdmitFingerprintRequests = append(createAdmitFingerprintRequests,
			admitFingerprintToCreateAdmitFingerprintRequest(fingerprint))
		deleteAdmitFingerprints = append(deleteAdmitFingerprints, fingerprint.ClientType)
	}

	return strings.TrimSuffix(buf.String(), ",") + ";",
		&pbdhcpagent.CreateAdmitFingerprintsRequest{Fingerprints: createAdmitFingerprintRequests},
		&pbdhcpagent.DeleteAdmitFingerprintsRequest{ClientTypes: deleteAdmitFingerprints}
}

func sendCreateAdmitFingerprintsCmdToDHCPAgent(createAdmitFingerprintsRequest *pbdhcpagent.CreateAdmitFingerprintsRequest, deleteAdmitFingerprintsRequest *pbdhcpagent.DeleteAdmitFingerprintsRequest) error {
	return kafka.SendDHCPCmd(kafka.CreateAdmitFingerprints,
		createAdmitFingerprintsRequest, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
				kafka.DeleteAdmitFingerprints, deleteAdmitFingerprintsRequest); err != nil {
				log.Warnf("batch create admit fingerprints failed and rollback failed: %s", err.Error())
			}
		})
}

func (f *AdmitFingerprintService) ExportExcel() (interface{}, error) {
	var fingerprints []*resource.AdmitFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&fingerprints,
			"select * from gr_admit_fingerprint order by client_type")
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameFingerprint), pg.Error(err).Error())
	}

	strMatrix := make([][]string, 0, len(fingerprints))
	for _, fingerprint := range fingerprints {
		strMatrix = append(strMatrix, localizationAdmitFingerprintToStrSlice(fingerprint))
	}

	if filepath, err := excel.WriteExcelFile(AdmitFingerprintFileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderAdmitFingerprint, strMatrix); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameFingerprint), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (f *AdmitFingerprintService) ExportExcelTemplate() (*excel.ExportFile, error) {
	if filepath, err := excel.WriteExcelFile(AdmitFingerprintTemplateFileName,
		TableHeaderAdmitFingerprint, TemplateAdmitFingerprint); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameTemplate), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (f *AdmitFingerprintService) BatchDelete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Exec("delete from gr_admit_fingerprint where id in ('" +
			strings.Join(ids, "','") + "')"); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete,
				string(errorno.ErrNameFingerprint), pg.Error(err).Error())
		} else if int(rows) != len(ids) {
			return errorno.ErrNotFound(errorno.ErrNameFingerprint, ids[0])
		}

		return sendDeleteAdmitFingerprintsCmdToDHCPAgent(ids)
	})
}

func sendDeleteAdmitFingerprintsCmdToDHCPAgent(ids []string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAdmitFingerprints,
		&pbdhcpagent.DeleteAdmitFingerprintsRequest{ClientTypes: ids}, nil)
}
