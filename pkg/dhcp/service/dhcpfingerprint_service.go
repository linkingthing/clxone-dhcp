package service

import (
	"bytes"
	"strings"
	"time"

	"github.com/linkingthing/cement/log"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
	"github.com/linkingthing/clxone-utils/excel"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

var (
	Wildcard = []byte("%")
)

const (
	OrderByCreateTime = "create_time desc"
)

var FingerprintFilterNames = []string{"fingerprint", "vendor_id", "operating_system", "client_type", "data_source"}

type DhcpFingerprintService struct{}

func NewDhcpFingerprintService() *DhcpFingerprintService {
	return &DhcpFingerprintService{}
}

func (h *DhcpFingerprintService) Create(fingerprint *resource.DhcpFingerprint) error {
	if err := fingerprint.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(fingerprint); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameFingerprint, fingerprint.Fingerprint, err)
		}

		return sendCreateFingerprintCmdToAgent(fingerprint)
	})
}

func sendCreateFingerprintCmdToAgent(fingerprint *resource.DhcpFingerprint) error {
	return kafka.SendDHCPCmd(kafka.CreateFingerprint,
		fingerprintToCreateFingerprintRequest(fingerprint), func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteFingerprint,
				fingerprintToDeleteFingerprintRequest(fingerprint)); err != nil {
				log.Errorf("create dhcp fingerprint %s failed, rollback with nodes %v failed: %s",
					fingerprint.Fingerprint, nodesForSucceed, err.Error())
			}
		})
}

func fingerprintToCreateFingerprintRequest(fingerprint *resource.DhcpFingerprint) *pbdhcpagent.CreateFingerprintRequest {
	return &pbdhcpagent.CreateFingerprintRequest{
		Fingerprint:     fingerprint.Fingerprint,
		VendorId:        getVendorIdByMatchPattern(fingerprint.VendorId, fingerprint.MatchPattern),
		OperatingSystem: fingerprint.OperatingSystem,
		ClientType:      fingerprint.ClientType,
		MatchPattern:    string(fingerprint.MatchPattern),
	}
}

func getVendorIdByMatchPattern(vendorId string, matchPattern resource.MatchPattern) string {
	if len(vendorId) == 0 || matchPattern == resource.MatchPatternEqual {
		return vendorId
	}

	vendorBytes := []byte(vendorId)
	switch matchPattern {
	case resource.MatchPatternPrefix:
		vendorBytes = append(vendorBytes, Wildcard...)
	case resource.MatchPatternSuffix:
		vendorBytes = append(Wildcard, vendorBytes...)
	case resource.MatchPatternKeyword:
		vendorBytes = append(Wildcard, vendorBytes...)
		vendorBytes = append(vendorBytes, Wildcard...)
	}

	return string(vendorBytes)
}

func (h *DhcpFingerprintService) List(conditions map[string]interface{}) ([]*resource.DhcpFingerprint, error) {
	var fingerprints []*resource.DhcpFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &fingerprints)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameFingerprint), pg.Error(err).Error())
	}

	return fingerprints, nil
}

func (h *DhcpFingerprintService) Get(id string) (*resource.DhcpFingerprint, error) {
	var fingerprints []*resource.DhcpFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &fingerprints)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(fingerprints) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameFingerprint, id)
	}

	return fingerprints[0], nil
}

func (h *DhcpFingerprintService) Update(fingerprint *resource.DhcpFingerprint) error {
	if err := fingerprint.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		oldFingerprint, err := getFingerprintWithoutReadOnly(tx, fingerprint.GetID())
		if err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableDhcpFingerprint, map[string]interface{}{
			resource.SqlColumnVendorId:        fingerprint.VendorId,
			resource.SqlColumnOperatingSystem: fingerprint.OperatingSystem,
			resource.SqlColumnClientType:      fingerprint.ClientType,
		}, map[string]interface{}{
			restdb.IDField: fingerprint.GetID(),
		}); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameFingerprint, oldFingerprint.Fingerprint, err)
		}

		return sendUpdateFingerprintCmdToDHCPAgent(oldFingerprint, fingerprint)
	})
}

func getFingerprintWithoutReadOnly(tx restdb.Transaction, id string) (*resource.DhcpFingerprint, error) {
	var fingerprints []*resource.DhcpFingerprint
	if err := tx.Fill(map[string]interface{}{restdb.IDField: id},
		&fingerprints); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(fingerprints) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameFingerprint, id)
	} else if fingerprints[0].DataSource == resource.DataSourceSystem {
		return nil, errorno.ErrReadOnly(id)
	} else {
		return fingerprints[0], nil
	}
}

func sendUpdateFingerprintCmdToDHCPAgent(oldFingerprint, newFingerprint *resource.DhcpFingerprint) error {
	return kafka.SendDHCPCmd(kafka.UpdateFingerprint,
		&pbdhcpagent.UpdateFingerprintRequest{
			Old: fingerprintToDeleteFingerprintRequest(oldFingerprint),
			New: fingerprintToCreateFingerprintRequest(newFingerprint)}, nil)
}

func (h *DhcpFingerprintService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		oldFingerprint, err := getFingerprintWithoutReadOnly(tx, id)
		if err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableDhcpFingerprint, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		}

		return sendDeleteFingerprintCmdToDHCPAgent(oldFingerprint)
	})
}

func sendDeleteFingerprintCmdToDHCPAgent(oldFingerprint *resource.DhcpFingerprint) error {
	return kafka.SendDHCPCmd(kafka.DeleteFingerprint,
		fingerprintToDeleteFingerprintRequest(oldFingerprint), nil)
}

func fingerprintToDeleteFingerprintRequest(fingerprint *resource.DhcpFingerprint) *pbdhcpagent.DeleteFingerprintRequest {
	return &pbdhcpagent.DeleteFingerprintRequest{
		Fingerprint:     fingerprint.Fingerprint,
		VendorId:        getVendorIdByMatchPattern(fingerprint.VendorId, fingerprint.MatchPattern),
		OperatingSystem: fingerprint.OperatingSystem,
		ClientType:      fingerprint.ClientType,
	}
}

func (s *DhcpFingerprintService) ImportExcel(file *excel.ImportFile) (interface{}, error) {
	if len(file.Name) == 0 {
		return nil, nil
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(DhcpFingerprintImportFileNamePrefix, TableHeaderDhcpFingerprintFail, response)
	validSql, createFingerprintsRequest, deleteFingerprintsRequest, err := parseDhcpFingerprintsFromFile(
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

		return sendCreateFingerprintsCmdToDHCPAgent(createFingerprintsRequest, deleteFingerprintsRequest)
	}); err != nil {
		return response, err
	}

	return response, nil
}

func parseDhcpFingerprintsFromFile(fileName string, response *excel.ImportResult) (string, *pbdhcpagent.CreateFingerprintsRequest, *pbdhcpagent.DeleteFingerprintsRequest, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return "", nil, nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return "", nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0], TableHeaderDhcpFingerprint, DhcpFingerprintMandatoryFields)
	if err != nil {
		return "", nil, nil, errorno.ErrInvalidTableHeader()
	}

	var oldFingerprints []*resource.DhcpFingerprint
	if err := db.GetResources(nil, &oldFingerprints); err != nil {
		return "", nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameFingerprint), err.Error())
	}

	response.InitData(len(contents) - 1)
	fingerprints := make([]*resource.DhcpFingerprint, 0, len(contents)-1)
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, DhcpFingerprintMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderDhcpFingerprintFailLen,
				localizationDhcpFingerprintToStrSlice(&resource.DhcpFingerprint{}),
				errorno.ErrMissingMandatory(j+2, DhcpFingerprintMandatoryFields).ErrorCN())
			continue
		}

		fingerprint := parseFingerprint(tableHeaderFields, fields)
		if err := fingerprint.Validate(); err != nil {
			addFailDataToResponse(response, TableHeaderDhcpFingerprintFailLen,
				localizationDhcpFingerprintToStrSlice(fingerprint), errorno.TryGetErrorCNMsg(err))
		} else if err := checkDhcpFingerprintConflictWithDhcpFingerprints(fingerprint,
			append(oldFingerprints, fingerprints...)); err != nil {
			addFailDataToResponse(response, TableHeaderDhcpFingerprintFailLen,
				localizationDhcpFingerprintToStrSlice(fingerprint), errorno.TryGetErrorCNMsg(err))
		} else {
			fingerprints = append(fingerprints, fingerprint)
		}
	}

	if len(fingerprints) == 0 {
		return "", nil, nil, nil
	}

	sql, createFingerprintsRequest, deleteFingerprintsRequest := dhcpFingerprintsToInsertSqlAndPbRequest(fingerprints)
	return sql, createFingerprintsRequest, deleteFingerprintsRequest, nil
}

func parseFingerprint(tableHeaderFields, fields []string) *resource.DhcpFingerprint {
	fingerprint := &resource.DhcpFingerprint{}
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameFingerprint:
			fingerprint.Fingerprint = strings.TrimSpace(field)
		case FieldNameVendorId:
			fingerprint.VendorId = strings.TrimSpace(field)
		case FieldNameOperatingSystem:
			fingerprint.OperatingSystem = strings.TrimSpace(field)
		case FieldNameClientType:
			fingerprint.ClientType = strings.TrimSpace(field)
		}
	}

	return fingerprint
}

func checkDhcpFingerprintConflictWithDhcpFingerprints(fingerprint *resource.DhcpFingerprint, fingerprints []*resource.DhcpFingerprint) error {
	for _, f := range fingerprints {
		if f.Equal(fingerprint) {
			return errorno.ErrConflict(errorno.ErrNameFingerprint, errorno.ErrNameFingerprint,
				f.Fingerprint, fingerprint.Fingerprint)
		}
	}

	return nil
}

func dhcpFingerprintsToInsertSqlAndPbRequest(fingerprints []*resource.DhcpFingerprint) (string, *pbdhcpagent.CreateFingerprintsRequest, *pbdhcpagent.DeleteFingerprintsRequest) {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_dhcp_fingerprint VALUES ")
	createFingerprintRequests := make([]*pbdhcpagent.CreateFingerprintRequest, 0, len(fingerprints))
	deleteFingerprintRequests := make([]*pbdhcpagent.DeleteFingerprintRequest, 0, len(fingerprints))
	for _, fingerprint := range fingerprints {
		buf.WriteString(dhcpFingerprintToInsertDBSqlString(fingerprint))
		createFingerprintRequests = append(createFingerprintRequests, fingerprintToCreateFingerprintRequest(fingerprint))
		deleteFingerprintRequests = append(deleteFingerprintRequests, fingerprintToDeleteFingerprintRequest(fingerprint))
	}

	return strings.TrimSuffix(buf.String(), ",") + ";",
		&pbdhcpagent.CreateFingerprintsRequest{Fingerprints: createFingerprintRequests},
		&pbdhcpagent.DeleteFingerprintsRequest{Fingerprints: deleteFingerprintRequests}
}

func sendCreateFingerprintsCmdToDHCPAgent(createFingerprintsRequest *pbdhcpagent.CreateFingerprintsRequest, deleteFingerprintsRequest *pbdhcpagent.DeleteFingerprintsRequest) error {
	return kafka.SendDHCPCmd(kafka.CreateFingerprints,
		createFingerprintsRequest, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
				kafka.DeleteFingerprints, deleteFingerprintsRequest); err != nil {
				log.Warnf("batch create fingerprints failed and rollback failed: %s", err.Error())
			}
		})
}

func (s *DhcpFingerprintService) ExportExcel() (interface{}, error) {
	var fingerprints []*resource.DhcpFingerprint
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&fingerprints,
			"select * from gr_dhcp_fingerprint where data_source != 'system' order by create_time desc")
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameFingerprint), pg.Error(err).Error())
	}

	strMatrix := make([][]string, 0, len(fingerprints))
	for _, fingerprint := range fingerprints {
		strMatrix = append(strMatrix, localizationDhcpFingerprintToStrSlice(fingerprint))
	}

	if filepath, err := excel.WriteExcelFile(DhcpFingerprintFileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderDhcpFingerprint, strMatrix); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameFingerprint), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (s *DhcpFingerprintService) ExportExcelTemplate() (*excel.ExportFile, error) {
	if filepath, err := excel.WriteExcelFile(DhcpFingerprintTemplateFileName,
		TableHeaderDhcpFingerprint, TemplateDhcpFingerprint); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameTemplate), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}
