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

type RateLimitDuidService struct{}

func NewRateLimitDuidService() *RateLimitDuidService {
	return &RateLimitDuidService{}
}

func (d *RateLimitDuidService) Create(rateLimitDuid *resource.RateLimitDuid) error {
	if err := rateLimitDuid.Validate(); err != nil {
		return err
	}

	rateLimitDuid.SetID(rateLimitDuid.Duid)
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(rateLimitDuid); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameDuid, rateLimitDuid.Duid, err)
		}

		return sendCreateRateLimitDuidCmdToDHCPAgent(rateLimitDuid)
	})
}

func sendCreateRateLimitDuidCmdToDHCPAgent(rateLimitDuid *resource.RateLimitDuid) error {
	return kafka.SendDHCPCmd(kafka.CreateRateLimitDuid, rateLimitDuidToCreateRateLimitDuidRequest(rateLimitDuid),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteRateLimitDuid,
				&pbdhcpagent.DeleteRateLimitDuidRequest{Duid: rateLimitDuid.Duid}); err != nil {
				log.Errorf("create ratelimit duid %s failed, rollback with nodes %v failed: %s",
					rateLimitDuid.Duid, nodesForSucceed, err.Error())
			}
		})
}

func rateLimitDuidToCreateRateLimitDuidRequest(rateLimitDuid *resource.RateLimitDuid) *pbdhcpagent.CreateRateLimitDuidRequest {
	return &pbdhcpagent.CreateRateLimitDuidRequest{
		Duid:  rateLimitDuid.Duid,
		Limit: rateLimitDuid.RateLimit,
	}
}

func (d *RateLimitDuidService) List(conditions map[string]interface{}) ([]*resource.RateLimitDuid, error) {
	var duids []*resource.RateLimitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &duids)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameRateLimit), pg.Error(err).Error())
	}

	return duids, nil
}

func (d *RateLimitDuidService) Get(id string) (*resource.RateLimitDuid, error) {
	var rateLimitDuids []*resource.RateLimitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &rateLimitDuids)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(rateLimitDuids) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameRateLimit, id)
	}

	return rateLimitDuids[0], nil
}

func (d *RateLimitDuidService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableRateLimitDuid, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameRateLimit, id)
		}

		return sendDeleteRateLimitDuidCmdToDHCPAgent(id)
	})
}

func sendDeleteRateLimitDuidCmdToDHCPAgent(rateLimitDuidId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteRateLimitDuid,
		&pbdhcpagent.DeleteRateLimitDuidRequest{Duid: rateLimitDuidId}, nil)
}

func (d *RateLimitDuidService) Update(rateLimitDuid *resource.RateLimitDuid) error {
	if err := rateLimitDuid.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var rateLimits []*resource.RateLimitDuid
		if err := tx.Fill(map[string]interface{}{restdb.IDField: rateLimitDuid.GetID()},
			&rateLimits); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, rateLimitDuid.GetID(), pg.Error(err).Error())
		} else if len(rateLimits) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameRateLimit, rateLimitDuid.GetID())
		}

		if _, err := tx.Update(resource.TableRateLimitDuid, map[string]interface{}{
			resource.SqlColumnRateLimit: rateLimitDuid.RateLimit,
			resource.SqlColumnComment:   rateLimitDuid.Comment,
		}, map[string]interface{}{restdb.IDField: rateLimitDuid.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, rateLimitDuid.GetID(), pg.Error(err).Error())
		}

		if rateLimits[0].RateLimit != rateLimitDuid.RateLimit {
			return sendUpdateRateLimitDuidCmdToDHCPAgent(rateLimitDuid)
		} else {
			return nil
		}
	})
}

func sendUpdateRateLimitDuidCmdToDHCPAgent(rateLimitDuid *resource.RateLimitDuid) error {
	return kafka.SendDHCPCmd(kafka.UpdateRateLimitDuid,
		&pbdhcpagent.UpdateRateLimitDuidRequest{
			Duid:  rateLimitDuid.Duid,
			Limit: rateLimitDuid.RateLimit,
		}, nil)
}

func (d *RateLimitDuidService) ImportExcel(file *excel.ImportFile) (interface{}, error) {
	if len(file.Name) == 0 {
		return nil, nil
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(RateLimitDuidImportFileNamePrefix, TableHeaderRateLimitDuidFail, response)
	validSql, createRateLimitDuidsRequest, deleteRateLimitDuidsRequest, err := parseRateLimitDuidsFromFile(
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

		return sendCreateRateLimitDuidsCmdToDHCPAgent(createRateLimitDuidsRequest, deleteRateLimitDuidsRequest)
	}); err != nil {
		return response, err
	}

	return response, nil
}

func parseRateLimitDuidsFromFile(fileName string, response *excel.ImportResult) (string, *pbdhcpagent.CreateRateLimitDuidsRequest, *pbdhcpagent.DeleteRateLimitDuidsRequest, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return "", nil, nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return "", nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0], TableHeaderRateLimitDuid, RateLimitDuidMandatoryFields)
	if err != nil {
		return "", nil, nil, errorno.ErrInvalidTableHeader()
	}

	var oldRateLimitDuids []*resource.RateLimitDuid
	if err := db.GetResources(nil, &oldRateLimitDuids); err != nil {
		return "", nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDuid), err.Error())
	}

	response.InitData(len(contents) - 1)
	duids := make([]*resource.RateLimitDuid, 0, len(contents)-1)
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, RateLimitDuidMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderRateLimitDuidFailLen,
				localizationRateLimitDuidToStrSlice(&resource.RateLimitDuid{}),
				errorno.ErrMissingMandatory(j+2, RateLimitDuidMandatoryFields).ErrorCN())
			continue
		}

		duid, err := parseRateLimitDuid(tableHeaderFields, fields)
		if err != nil {
			addFailDataToResponse(response, TableHeaderRateLimitDuidFailLen,
				localizationRateLimitDuidToStrSlice(duid), errorno.TryGetErrorCNMsg(err))
		} else if err := duid.Validate(); err != nil {
			addFailDataToResponse(response, TableHeaderRateLimitDuidFailLen,
				localizationRateLimitDuidToStrSlice(duid), errorno.TryGetErrorCNMsg(err))
		} else if err := checkRateLimitDuidConflictWithRateLimitDuids(duid,
			append(oldRateLimitDuids, duids...)); err != nil {
			addFailDataToResponse(response, TableHeaderRateLimitDuidFailLen,
				localizationRateLimitDuidToStrSlice(duid), errorno.TryGetErrorCNMsg(err))
		} else {
			duids = append(duids, duid)
		}
	}

	if len(duids) == 0 {
		return "", nil, nil, nil
	}

	sql, createRateLimitDuidsRequest, deleteRateLimitDuidsRequest := rateLimitDuidsToInsertSqlAndPbRequest(duids)
	return sql, createRateLimitDuidsRequest, deleteRateLimitDuidsRequest, nil
}

func parseRateLimitDuid(tableHeaderFields, fields []string) (*resource.RateLimitDuid, error) {
	var err error
	duid := &resource.RateLimitDuid{}
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameRateLimitDuid:
			duid.Duid = strings.TrimSpace(field)
		case FieldNameRateLimit:
			if duid.RateLimit, err = parseUint32FromString(strings.TrimSpace(field)); err != nil {
				return duid, errorno.ErrInvalidParams(errorno.ErrNameRateLimit, field)
			}
		case FieldNameComment:
			duid.Comment = field
		}
	}

	return duid, nil
}

func checkRateLimitDuidConflictWithRateLimitDuids(duid *resource.RateLimitDuid, duids []*resource.RateLimitDuid) error {
	for _, d := range duids {
		if d.Duid == duid.Duid {
			return errorno.ErrConflict(errorno.ErrNameDuid, errorno.ErrNameDuid,
				d.Duid, duid.Duid)
		}
	}

	return nil
}

func rateLimitDuidsToInsertSqlAndPbRequest(duids []*resource.RateLimitDuid) (string, *pbdhcpagent.CreateRateLimitDuidsRequest, *pbdhcpagent.DeleteRateLimitDuidsRequest) {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_rate_limit_duid VALUES ")
	createRateLimitDuidRequests := make([]*pbdhcpagent.CreateRateLimitDuidRequest, 0, len(duids))
	deleteRateLimitDuids := make([]string, 0, len(duids))
	for _, duid := range duids {
		buf.WriteString(rateLimitDuidToInsertDBSqlString(duid))
		createRateLimitDuidRequests = append(createRateLimitDuidRequests, rateLimitDuidToCreateRateLimitDuidRequest(duid))
		deleteRateLimitDuids = append(deleteRateLimitDuids, duid.Duid)
	}

	return strings.TrimSuffix(buf.String(), ",") + ";",
		&pbdhcpagent.CreateRateLimitDuidsRequest{Duids: createRateLimitDuidRequests},
		&pbdhcpagent.DeleteRateLimitDuidsRequest{Duids: deleteRateLimitDuids}
}

func sendCreateRateLimitDuidsCmdToDHCPAgent(createRateLimitDuidsRequest *pbdhcpagent.CreateRateLimitDuidsRequest, deleteRateLimitDuidsRequest *pbdhcpagent.DeleteRateLimitDuidsRequest) error {
	return kafka.SendDHCPCmd(kafka.CreateRateLimitDuids,
		createRateLimitDuidsRequest, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
				kafka.DeleteRateLimitDuids, deleteRateLimitDuidsRequest); err != nil {
				log.Warnf("batch create ratelimit duids failed and rollback failed: %s", err.Error())
			}
		})
}

func (d *RateLimitDuidService) ExportExcel() (interface{}, error) {
	var duids []*resource.RateLimitDuid
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&duids,
			"select * from gr_rate_limit_duid order by duid")
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDuid), pg.Error(err).Error())
	}

	strMatrix := make([][]string, 0, len(duids))
	for _, duid := range duids {
		strMatrix = append(strMatrix, localizationRateLimitDuidToStrSlice(duid))
	}

	if filepath, err := excel.WriteExcelFile(RateLimitDuidFileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderRateLimitDuid, strMatrix); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameDuid), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (d *RateLimitDuidService) ExportExcelTemplate() (*excel.ExportFile, error) {
	if filepath, err := excel.WriteExcelFile(RateLimitDuidTemplateFileName,
		TableHeaderRateLimitDuid, TemplateRateLimitDuid); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameTemplate), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (d *RateLimitDuidService) BatchDelete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Exec("delete from gr_rate_limit_duid where id in ('" +
			strings.Join(ids, "','") + "')"); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete,
				string(errorno.ErrNameDuid), pg.Error(err).Error())
		} else if int(rows) != len(ids) {
			return errorno.ErrNotFound(errorno.ErrNameDuid, ids[0])
		}

		return sendDeleteRateLimitDuidsCmdToDHCPAgent(ids)
	})
}

func sendDeleteRateLimitDuidsCmdToDHCPAgent(ids []string) error {
	return kafka.SendDHCPCmd(kafka.DeleteRateLimitDuids,
		&pbdhcpagent.DeleteRateLimitDuidsRequest{Duids: ids}, nil)
}
