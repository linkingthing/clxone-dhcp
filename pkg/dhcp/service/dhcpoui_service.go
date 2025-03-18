package service

import (
	"bytes"
	"strings"
	"time"

	"github.com/linkingthing/cement/log"
	"github.com/linkingthing/clxone-utils/excel"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	FieldOUI = "oui"
)

type DhcpOuiService struct{}

func NewDhcpOuiService() *DhcpOuiService {
	return &DhcpOuiService{}
}

func (d *DhcpOuiService) Create(dhcpOui *resource.DhcpOui) error {
	if err := dhcpOui.Validate(); err != nil {
		return err
	}

	dhcpOui.SetID(dhcpOui.Oui)
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(dhcpOui); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameOui, dhcpOui.Oui, err)
		}

		return sendCreateOuiCmdToDHCPAgent(dhcpOui)
	})
}

func sendCreateOuiCmdToDHCPAgent(dhcpOui *resource.DhcpOui) error {
	return kafka.SendDHCPCmd(kafka.CreateOui, ouiToCreateOuiRequest(dhcpOui),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
				kafka.DeleteOui, &pbdhcpagent.DeleteOuiRequest{Oui: dhcpOui.Oui}); err != nil {
				log.Errorf("create oui %s failed, rollback with nodes %v failed: %s",
					dhcpOui.Oui, nodesForSucceed, err.Error())
			}
		})
}

func ouiToCreateOuiRequest(oui *resource.DhcpOui) *pbdhcpagent.CreateOuiRequest {
	return &pbdhcpagent.CreateOuiRequest{
		Oui:          oui.Oui,
		Organization: oui.Organization,
	}
}

func (d *DhcpOuiService) List(ctx *restresource.Context) ([]*resource.DhcpOui, error) {
	listCtx := genGetOUIContext(ctx)
	var ouis []*resource.DhcpOui
	var ouiCount int
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if listCtx.hasPagination {
			if count, err := tx.CountEx(resource.TableDhcpOui,
				listCtx.countSql); err != nil {
				return errorno.ErrDBError(errorno.ErrDBNameCount,
					string(errorno.ErrNameOui), pg.Error(err).Error())
			} else {
				ouiCount = int(count)
			}
		}

		if err := tx.FillEx(&ouis, listCtx.sql, listCtx.params...); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery,
				string(errorno.ErrNameOui), pg.Error(err).Error())
		}
		return nil
	}); err != nil {
		return nil, err
	}

	setPagination(ctx, listCtx.hasPagination, ouiCount)
	return ouis, nil
}

type listOUIContext struct {
	countSql      string
	sql           string
	params        []interface{}
	hasFilterOUI  bool
	hasPagination bool
}

func genGetOUIContext(ctx *restresource.Context) listOUIContext {
	listCtx := listOUIContext{}
	if value, ok := util.GetFilterValueWithEqModifierFromFilters(FieldOUI,
		ctx.GetFilters()); ok {
		listCtx.hasFilterOUI = true
		listCtx.sql = "select * from gr_dhcp_oui where oui = $1"
		listCtx.params = append(listCtx.params, strings.ToUpper(value))
	} else {
		listCtx.sql = "select * from gr_dhcp_oui"
	}

	listCtx.countSql = strings.Replace(listCtx.sql, "*", "count(*)", 1)
	if !listCtx.hasFilterOUI {
		listCtx.sql += " order by create_time desc "
		if pagination := ctx.GetPagination(); pagination.PageSize > 0 &&
			pagination.PageNum > 0 {
			listCtx.hasPagination = true
			listCtx.sql += " limit $1 offset $2"
			listCtx.params = append(listCtx.params, pagination.PageSize)
			listCtx.params = append(listCtx.params, (pagination.PageNum-1)*pagination.PageSize)
		}
	}

	return listCtx
}

func (d *DhcpOuiService) Get(id string) (*resource.DhcpOui, error) {
	var dhcpOuis []*resource.DhcpOui
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &dhcpOuis)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(dhcpOuis) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameOui, id)
	}

	return dhcpOuis[0], nil
}

func (d *DhcpOuiService) Update(dhcpOui *resource.DhcpOui) error {
	if err := dhcpOui.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := d.checkOuiIsReadOnly(tx, dhcpOui.GetID()); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableDhcpOui, map[string]interface{}{
			resource.SqlColumnOuiOrganization: dhcpOui.Organization,
		}, map[string]interface{}{restdb.IDField: dhcpOui.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, dhcpOui.GetID(), pg.Error(err).Error())
		}

		return sendUpdateDhcpOuiCmdToDHCPAgent(dhcpOui)
	})
}

func (d *DhcpOuiService) checkOuiIsReadOnly(tx restdb.Transaction, id string) error {
	var dhcpOuis []*resource.DhcpOui
	if err := tx.Fill(map[string]interface{}{restdb.IDField: id},
		&dhcpOuis); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(dhcpOuis) == 0 {
		return errorno.ErrNotFound(errorno.ErrNameOui, id)
	} else if dhcpOuis[0].DataSource == resource.DataSourceSystem {
		return errorno.ErrReadOnly(id)
	} else {
		return nil
	}
}

func sendUpdateDhcpOuiCmdToDHCPAgent(dhcpoui *resource.DhcpOui) error {
	return kafka.SendDHCPCmd(kafka.UpdateOui, &pbdhcpagent.UpdateOuiRequest{
		Oui:          dhcpoui.Oui,
		Organization: dhcpoui.Organization,
	}, nil)
}

func (d *DhcpOuiService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := d.checkOuiIsReadOnly(tx, id); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableDhcpOui, map[string]interface{}{
			restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		}

		return sendDeleteDhcpOuiCmdToDHCPAgent(id)
	})
}

func sendDeleteDhcpOuiCmdToDHCPAgent(dhcpOuiId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteOui,
		&pbdhcpagent.DeleteOuiRequest{Oui: dhcpOuiId}, nil)
}

func (d *DhcpOuiService) ImportExcel(file *excel.ImportFile) (interface{}, error) {
	if len(file.Name) == 0 {
		return nil, nil
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(DhcpOuiImportFileNamePrefix, TableHeaderDhcpOuiFail, response)
	validSql, createOuisRequest, deleteOuisRequest, err := parseDhcpOuisFromFile(
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
				string(errorno.ErrNameOui), pg.Error(err).Error())
		}

		return sendCreateOuisCmdToDHCPAgent(createOuisRequest, deleteOuisRequest)
	}); err != nil {
		return response, err
	}

	return response, nil
}

func parseDhcpOuisFromFile(fileName string, response *excel.ImportResult) (string, *pbdhcpagent.CreateOuisRequest, *pbdhcpagent.DeleteOuisRequest, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return "", nil, nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return "", nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0], TableHeaderDhcpOui, DhcpOuiMandatoryFields)
	if err != nil {
		return "", nil, nil, errorno.ErrInvalidTableHeader()
	}

	var oldOuis []*resource.DhcpOui
	if err := db.GetResources(nil, &oldOuis); err != nil {
		return "", nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameOui), err.Error())
	}

	response.InitData(len(contents) - 1)
	ouis := make([]*resource.DhcpOui, 0, len(contents)-1)
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, DhcpOuiMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderDhcpOuiFailLen,
				localizationDhcpOuiToStrSlice(&resource.DhcpOui{}),
				errorno.ErrMissingMandatory(j+2, DhcpOuiMandatoryFields).ErrorCN())
			continue
		}

		oui := parseOui(tableHeaderFields, fields)
		if err := oui.Validate(); err != nil {
			addFailDataToResponse(response, TableHeaderDhcpOuiFailLen,
				localizationDhcpOuiToStrSlice(oui), errorno.TryGetErrorCNMsg(err))
		} else if err := checkDhcpOuiConflictWithDhcpOuis(oui,
			append(oldOuis, ouis...)); err != nil {
			addFailDataToResponse(response, TableHeaderDhcpOuiFailLen,
				localizationDhcpOuiToStrSlice(oui), errorno.TryGetErrorCNMsg(err))
		} else {
			ouis = append(ouis, oui)
		}
	}

	if len(ouis) == 0 {
		return "", nil, nil, nil
	}

	sql, createOuisRequest, deleteOuisRequest := dhcpOuisToInsertSqlAndPbRequest(ouis)
	return sql, createOuisRequest, deleteOuisRequest, nil
}

func parseOui(tableHeaderFields, fields []string) *resource.DhcpOui {
	oui := &resource.DhcpOui{}
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameOui:
			oui.Oui = strings.TrimSpace(field)
		case FieldNameOrganization:
			oui.Organization = strings.TrimSpace(field)
		}
	}

	return oui
}

func checkDhcpOuiConflictWithDhcpOuis(oui *resource.DhcpOui, ouis []*resource.DhcpOui) error {
	for _, o := range ouis {
		if o.Oui == oui.Oui {
			return errorno.ErrConflict(errorno.ErrNameOui, errorno.ErrNameOui,
				o.Oui, oui.Oui)
		}
	}

	return nil
}

func dhcpOuisToInsertSqlAndPbRequest(ouis []*resource.DhcpOui) (string, *pbdhcpagent.CreateOuisRequest, *pbdhcpagent.DeleteOuisRequest) {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_dhcp_oui VALUES ")
	createOuiRequests := make([]*pbdhcpagent.CreateOuiRequest, 0, len(ouis))
	deleteOuis := make([]string, 0, len(ouis))
	for _, oui := range ouis {
		buf.WriteString(dhcpOuiToInsertDBSqlString(oui))
		createOuiRequests = append(createOuiRequests, ouiToCreateOuiRequest(oui))
		deleteOuis = append(deleteOuis, oui.Oui)
	}

	return strings.TrimSuffix(buf.String(), ",") + ";",
		&pbdhcpagent.CreateOuisRequest{Ouis: createOuiRequests},
		&pbdhcpagent.DeleteOuisRequest{Ouis: deleteOuis}
}

func sendCreateOuisCmdToDHCPAgent(createOuisRequest *pbdhcpagent.CreateOuisRequest, deleteOuisRequest *pbdhcpagent.DeleteOuisRequest) error {
	return kafka.SendDHCPCmd(kafka.CreateOuis,
		createOuisRequest, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
				kafka.DeleteOuis, deleteOuisRequest); err != nil {
				log.Warnf("batch create ouis failed and rollback failed: %s", err.Error())
			}
		})
}

func (d *DhcpOuiService) ExportExcel() (interface{}, error) {
	var ouis []*resource.DhcpOui
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&ouis,
			"select * from gr_dhcp_oui where data_source != 'system' order by oui")
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameOui), pg.Error(err).Error())
	}

	strMatrix := make([][]string, 0, len(ouis))
	for _, oui := range ouis {
		strMatrix = append(strMatrix, localizationDhcpOuiForExport(oui))
	}

	if filepath, err := excel.WriteExcelFile(DhcpOuiFileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderDhcpOuiForExport, strMatrix); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameOui), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (d *DhcpOuiService) ExportExcelTemplate() (*excel.ExportFile, error) {
	if filepath, err := excel.WriteExcelFile(DhcpOuiTemplateFileName,
		TableHeaderDhcpOui, TemplateDhcpOui); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameTemplate), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (d *DhcpOuiService) BatchDelete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	idsStr := strings.Join(ids, "','")
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var ouis []*resource.DhcpOui
		if err := tx.FillEx(&ouis,
			"select * from gr_dhcp_oui where id in ('"+idsStr+"')"); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery,
				string(errorno.ErrNameOui), pg.Error(err).Error())
		} else if len(ouis) == 0 || len(ouis) != len(ids) {
			return errorno.ErrNotFound(errorno.ErrNameOui, ids[0])
		}

		for _, oui := range ouis {
			if oui.DataSource == resource.DataSourceSystem {
				return errorno.ErrReadOnly(oui.Oui)
			}
		}

		if _, err := tx.Exec("delete from gr_dhcp_oui where id in ('" +
			idsStr + "')"); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete,
				string(errorno.ErrNameOui), pg.Error(err).Error())
		}

		return sendDeleteOuisCmdToDHCPAgent(ids)
	})
}

func sendDeleteOuisCmdToDHCPAgent(ids []string) error {
	return kafka.SendDHCPCmd(kafka.DeleteOuis,
		&pbdhcpagent.DeleteOuisRequest{Ouis: ids}, nil)
}
