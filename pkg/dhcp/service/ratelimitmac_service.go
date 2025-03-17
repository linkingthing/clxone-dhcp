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

type RateLimitMacService struct{}

func NewRateLimitMacService() *RateLimitMacService {
	return &RateLimitMacService{}
}

func (d *RateLimitMacService) Create(rateLimitMac *resource.RateLimitMac) error {
	if err := rateLimitMac.Validate(); err != nil {
		return err
	}

	rateLimitMac.SetID(rateLimitMac.HwAddress)
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(rateLimitMac); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameMac, rateLimitMac.HwAddress, err)
		}

		return sendCreateRateLimitMacCmdToDHCPAgent(rateLimitMac)
	})
}

func sendCreateRateLimitMacCmdToDHCPAgent(rateLimitMac *resource.RateLimitMac) error {
	return kafka.SendDHCPCmd(kafka.CreateRateLimitMac, rateLimitMacToCreateRateLimitMacRequest(rateLimitMac),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
				kafka.DeleteRateLimitMac, &pbdhcpagent.DeleteRateLimitMacRequest{
					HwAddress: rateLimitMac.HwAddress}); err != nil {
				log.Errorf("create ratelimit mac %s failed, rollback with nodes %v failed: %s",
					rateLimitMac.HwAddress, nodesForSucceed, err.Error())
			}
		})
}

func rateLimitMacToCreateRateLimitMacRequest(rateLimitMac *resource.RateLimitMac) *pbdhcpagent.CreateRateLimitMacRequest {
	return &pbdhcpagent.CreateRateLimitMacRequest{
		HwAddress: rateLimitMac.HwAddress,
		Limit:     rateLimitMac.RateLimit,
	}
}

func (d *RateLimitMacService) List(condition map[string]interface{}) ([]*resource.RateLimitMac, error) {
	var rateLimitMacs []*resource.RateLimitMac
	if mac, ok := condition[resource.SqlColumnHwAddress].(string); ok {
		if mac, _ = util.NormalizeMac(mac); mac != "" {
			condition[resource.SqlColumnHwAddress] = mac
		}
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(condition, &rateLimitMacs)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameRateLimit), pg.Error(err).Error())
	}

	return rateLimitMacs, nil
}

func (d *RateLimitMacService) Get(id string) (*resource.RateLimitMac, error) {
	var rateLimitMacs []*resource.RateLimitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &rateLimitMacs)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(rateLimitMacs) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameRateLimit, id)
	}

	return rateLimitMacs[0], nil
}

func (d *RateLimitMacService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableRateLimitMac, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameRateLimit, id)
		}

		return sendDeleteRateLimitMacCmdToDHCPAgent(id)
	})
}

func sendDeleteRateLimitMacCmdToDHCPAgent(ratelimitMacId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteRateLimitMac,
		&pbdhcpagent.DeleteRateLimitMacRequest{HwAddress: ratelimitMacId}, nil)
}

func (d *RateLimitMacService) Update(rateLimitMac *resource.RateLimitMac) error {
	if err := rateLimitMac.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var rateLimits []*resource.RateLimitMac
		if err := tx.Fill(map[string]interface{}{restdb.IDField: rateLimitMac.GetID()},
			&rateLimits); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, rateLimitMac.GetID(), pg.Error(err).Error())
		} else if len(rateLimits) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameRateLimit, rateLimitMac.GetID())
		}

		if _, err := tx.Update(resource.TableRateLimitMac, map[string]interface{}{
			resource.SqlColumnRateLimit: rateLimitMac.RateLimit,
			resource.SqlColumnComment:   rateLimitMac.Comment,
		}, map[string]interface{}{restdb.IDField: rateLimitMac.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, rateLimitMac.GetID(), pg.Error(err).Error())
		}

		if rateLimits[0].RateLimit != rateLimitMac.RateLimit {
			return sendUpdateRateLimitMacCmdToDHCPAgent(rateLimitMac)
		} else {
			return nil
		}
	})
}

func sendUpdateRateLimitMacCmdToDHCPAgent(ratelimitMac *resource.RateLimitMac) error {
	return kafka.SendDHCPCmd(kafka.UpdateRateLimitMac,
		&pbdhcpagent.UpdateRateLimitMacRequest{
			HwAddress: ratelimitMac.HwAddress,
			Limit:     ratelimitMac.RateLimit,
		}, nil)
}

func (m *RateLimitMacService) ImportExcel(file *excel.ImportFile) (interface{}, error) {
	if len(file.Name) == 0 {
		return nil, nil
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(RateLimitMacImportFileNamePrefix, TableHeaderRateLimitMacFail, response)
	validSql, createRateLimitMacsRequest, deleteRateLimitMacsRequest, err := parseRateLimitMacsFromFile(
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
				string(errorno.ErrNameMac), pg.Error(err).Error())
		}

		return sendCreateRateLimitMacsCmdToDHCPAgent(createRateLimitMacsRequest, deleteRateLimitMacsRequest)
	}); err != nil {
		return response, err
	}

	return response, nil
}

func parseRateLimitMacsFromFile(fileName string, response *excel.ImportResult) (string, *pbdhcpagent.CreateRateLimitMacsRequest, *pbdhcpagent.DeleteRateLimitMacsRequest, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return "", nil, nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return "", nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0], TableHeaderRateLimitMac, RateLimitMacMandatoryFields)
	if err != nil {
		return "", nil, nil, errorno.ErrInvalidTableHeader()
	}

	var oldRateLimitMacs []*resource.RateLimitMac
	if err := db.GetResources(nil, &oldRateLimitMacs); err != nil {
		return "", nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameMac), err.Error())
	}

	response.InitData(len(contents) - 1)
	macs := make([]*resource.RateLimitMac, 0, len(contents)-1)
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, RateLimitMacMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderRateLimitMacFailLen,
				localizationRateLimitMacToStrSlice(&resource.RateLimitMac{}),
				errorno.ErrMissingMandatory(j+2, RateLimitMacMandatoryFields).ErrorCN())
			continue
		}

		mac, err := parseRateLimitMac(tableHeaderFields, fields)
		if err != nil {
			addFailDataToResponse(response, TableHeaderRateLimitMacFailLen,
				localizationRateLimitMacToStrSlice(mac), errorno.TryGetErrorCNMsg(err))
		} else if err := mac.Validate(); err != nil {
			addFailDataToResponse(response, TableHeaderRateLimitMacFailLen,
				localizationRateLimitMacToStrSlice(mac), errorno.TryGetErrorCNMsg(err))
		} else if err := checkRateLimitMacConflictWithRateLimitMacs(mac,
			append(oldRateLimitMacs, macs...)); err != nil {
			addFailDataToResponse(response, TableHeaderRateLimitMacFailLen,
				localizationRateLimitMacToStrSlice(mac), errorno.TryGetErrorCNMsg(err))
		} else {
			macs = append(macs, mac)
		}
	}

	if len(macs) == 0 {
		return "", nil, nil, nil
	}

	sql, createRateLimitMacsRequest, deleteRateLimitMacsRequest := rateLimitMacsToInsertSqlAndPbRequest(macs)
	return sql, createRateLimitMacsRequest, deleteRateLimitMacsRequest, nil
}

func parseRateLimitMac(tableHeaderFields, fields []string) (*resource.RateLimitMac, error) {
	mac := &resource.RateLimitMac{}
	var err error
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameRateLimitMac:
			mac.HwAddress = strings.TrimSpace(field)
		case FieldNameRateLimit:
			if mac.RateLimit, err = parseUint32FromString(strings.TrimSpace(field)); err != nil {
				return mac, errorno.ErrInvalidParams(errorno.ErrNameRateLimit, field)
			}
		case FieldNameComment:
			mac.Comment = field
		}
	}

	return mac, nil
}

func checkRateLimitMacConflictWithRateLimitMacs(mac *resource.RateLimitMac, macs []*resource.RateLimitMac) error {
	for _, m := range macs {
		if m.HwAddress == mac.HwAddress {
			return errorno.ErrConflict(errorno.ErrNameMac, errorno.ErrNameMac,
				m.HwAddress, mac.HwAddress)
		}
	}

	return nil
}

func rateLimitMacsToInsertSqlAndPbRequest(macs []*resource.RateLimitMac) (string, *pbdhcpagent.CreateRateLimitMacsRequest, *pbdhcpagent.DeleteRateLimitMacsRequest) {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_rate_limit_mac VALUES ")
	createRateLimitMacRequests := make([]*pbdhcpagent.CreateRateLimitMacRequest, 0, len(macs))
	deleteRateLimitMacs := make([]string, 0, len(macs))
	for _, mac := range macs {
		buf.WriteString(rateLimitMacToInsertDBSqlString(mac))
		createRateLimitMacRequests = append(createRateLimitMacRequests, rateLimitMacToCreateRateLimitMacRequest(mac))
		deleteRateLimitMacs = append(deleteRateLimitMacs, mac.HwAddress)
	}

	return strings.TrimSuffix(buf.String(), ",") + ";",
		&pbdhcpagent.CreateRateLimitMacsRequest{Macs: createRateLimitMacRequests},
		&pbdhcpagent.DeleteRateLimitMacsRequest{HwAddresses: deleteRateLimitMacs}
}

func sendCreateRateLimitMacsCmdToDHCPAgent(createRateLimitMacsRequest *pbdhcpagent.CreateRateLimitMacsRequest, deleteRateLimitMacsRequest *pbdhcpagent.DeleteRateLimitMacsRequest) error {
	return kafka.SendDHCPCmd(kafka.CreateRateLimitMacs,
		createRateLimitMacsRequest, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
				kafka.DeleteRateLimitMacs, deleteRateLimitMacsRequest); err != nil {
				log.Warnf("batch create ratelimit macs failed and rollback failed: %s", err.Error())
			}
		})
}

func (m *RateLimitMacService) ExportExcel() (interface{}, error) {
	var macs []*resource.RateLimitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&macs,
			"select * from gr_rate_limit_mac order by hw_address")
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameMac), pg.Error(err).Error())
	}

	strMatrix := make([][]string, 0, len(macs))
	for _, mac := range macs {
		strMatrix = append(strMatrix, localizationRateLimitMacToStrSlice(mac))
	}

	if filepath, err := excel.WriteExcelFile(RateLimitMacFileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderRateLimitMac, strMatrix); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameMac), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (m *RateLimitMacService) ExportExcelTemplate() (*excel.ExportFile, error) {
	if filepath, err := excel.WriteExcelFile(RateLimitMacTemplateFileName,
		TableHeaderRateLimitMac, TemplateRateLimitMac); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameTemplate), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (d *RateLimitMacService) BatchDelete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Exec("delete from gr_rate_limit_mac where id in ('" +
			strings.Join(ids, "','") + "')"); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete,
				string(errorno.ErrNameMac), pg.Error(err).Error())
		} else if int(rows) != len(ids) {
			return errorno.ErrNotFound(errorno.ErrNameMac, ids[0])
		}

		return sendDeleteRateLimitMacsCmdToDHCPAgent(ids)
	})
}

func sendDeleteRateLimitMacsCmdToDHCPAgent(ids []string) error {
	return kafka.SendDHCPCmd(kafka.DeleteRateLimitMacs,
		&pbdhcpagent.DeleteRateLimitMacsRequest{HwAddresses: ids}, nil)
}
