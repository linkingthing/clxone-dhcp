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

type AdmitMacService struct{}

func NewAdmitMacService() *AdmitMacService {
	return &AdmitMacService{}
}

func (m *AdmitMacService) Create(admitMac *resource.AdmitMac) error {
	if err := admitMac.Validate(); err != nil {
		return err
	}

	admitMac.SetID(admitMac.HwAddress)
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(admitMac); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameMac, admitMac.HwAddress, err)
		}

		return sendCreateAdmitMacCmdToDHCPAgent(admitMac)
	})
}

func sendCreateAdmitMacCmdToDHCPAgent(admitMac *resource.AdmitMac) error {
	return kafka.SendDHCPCmd(kafka.CreateAdmitMac, admitMacToCreateAdmitMacRequest(admitMac),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteAdmitMac,
				&pbdhcpagent.DeleteAdmitMacRequest{HwAddress: admitMac.HwAddress},
			); err != nil {
				log.Errorf("create admit mac %s failed, rollback with nodes %v failed: %s",
					admitMac.HwAddress, nodesForSucceed, err.Error())
			}
		})
}

func admitMacToCreateAdmitMacRequest(admitMac *resource.AdmitMac) *pbdhcpagent.CreateAdmitMacRequest {
	return &pbdhcpagent.CreateAdmitMacRequest{
		HwAddress:  admitMac.HwAddress,
		IsAdmitted: admitMac.IsAdmitted,
	}
}

func (m *AdmitMacService) List(conditions map[string]interface{}) ([]*resource.AdmitMac, error) {
	if mac, ok := conditions[resource.SqlColumnHwAddress].(string); ok {
		if mac, _ = util.NormalizeMac(mac); mac != "" {
			conditions[resource.SqlColumnHwAddress] = mac
		}
	}

	var macs []*resource.AdmitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(conditions, &macs)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameAdmit), pg.Error(err).Error())
	}

	return macs, nil
}

func (m *AdmitMacService) Get(id string) (*resource.AdmitMac, error) {
	var admitMacs []*resource.AdmitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &admitMacs)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(admitMacs) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameAdmit, id)
	}

	return admitMacs[0], nil
}

func (m *AdmitMacService) Delete(id string) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Delete(resource.TableAdmitMac,
			map[string]interface{}{restdb.IDField: id}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, id, pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAdmit, id)
		}

		return sendDeleteAdmitMacCmdToDHCPAgent(id)
	})
}

func sendDeleteAdmitMacCmdToDHCPAgent(admitMacId string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAdmitMac,
		&pbdhcpagent.DeleteAdmitMacRequest{HwAddress: admitMacId}, nil)
}

func (m *AdmitMacService) Update(admitMac *resource.AdmitMac) error {
	if err := admitMac.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var admitMacs []*resource.AdmitMac
		if err := tx.Fill(map[string]interface{}{restdb.IDField: admitMac.GetID()},
			&admitMacs); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, admitMac.GetID(), pg.Error(err).Error())
		} else if len(admitMacs) == 0 {
			return errorno.ErrNotFound(errorno.ErrNameAdmit, admitMac.GetID())
		}

		if _, err := tx.Update(resource.TableAdmitMac,
			map[string]interface{}{
				resource.SqlColumnIsAdmitted: admitMac.IsAdmitted,
				resource.SqlColumnComment:    admitMac.Comment,
			},
			map[string]interface{}{restdb.IDField: admitMac.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, admitMac.GetID(), pg.Error(err).Error())
		}

		if admitMacs[0].IsAdmitted != admitMac.IsAdmitted {
			return sendUpdateAdmitMacCmdToDHCPAgent(admitMac)
		} else {
			return nil
		}
	})
}

func sendUpdateAdmitMacCmdToDHCPAgent(admitMac *resource.AdmitMac) error {
	return kafka.SendDHCPCmd(kafka.UpdateAdmitMac,
		&pbdhcpagent.UpdateAdmitMacRequest{
			HwAddress:  admitMac.HwAddress,
			IsAdmitted: admitMac.IsAdmitted,
		}, nil)
}

func (m *AdmitMacService) ImportExcel(file *excel.ImportFile) (interface{}, error) {
	if len(file.Name) == 0 {
		return nil, nil
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(AdmitMacImportFileNamePrefix, TableHeaderAdmitMacFail, response)
	validSql, createAdmitMacsRequest, deleteAdmitMacsRequest, err := parseAdmitMacsFromFile(
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

		return sendCreateAdmitMacsCmdToDHCPAgent(createAdmitMacsRequest, deleteAdmitMacsRequest)
	}); err != nil {
		return response, err
	}

	return response, nil
}

func parseAdmitMacsFromFile(fileName string, response *excel.ImportResult) (string, *pbdhcpagent.CreateAdmitMacsRequest, *pbdhcpagent.DeleteAdmitMacsRequest, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return "", nil, nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return "", nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0], TableHeaderAdmitMac, AdmitMacMandatoryFields)
	if err != nil {
		return "", nil, nil, errorno.ErrInvalidTableHeader()
	}

	var oldAdmitMacs []*resource.AdmitMac
	if err := db.GetResources(nil, &oldAdmitMacs); err != nil {
		return "", nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameMac), err.Error())
	}

	response.InitData(len(contents) - 1)
	macs := make([]*resource.AdmitMac, 0, len(contents)-1)
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, AdmitMacMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderAdmitMacFailLen,
				localizationAdmitMacToStrSlice(&resource.AdmitMac{}),
				errorno.ErrMissingMandatory(j+2, AdmitMacMandatoryFields).ErrorCN())
			continue
		}

		mac := parseAdmitMac(tableHeaderFields, fields)
		if err := mac.Validate(); err != nil {
			addFailDataToResponse(response, TableHeaderAdmitMacFailLen,
				localizationAdmitMacToStrSlice(mac), errorno.TryGetErrorCNMsg(err))
		} else if err := checkAdmitMacConflictWithAdmitMacs(mac,
			append(oldAdmitMacs, macs...)); err != nil {
			addFailDataToResponse(response, TableHeaderAdmitMacFailLen,
				localizationAdmitMacToStrSlice(mac), errorno.TryGetErrorCNMsg(err))
		} else {
			macs = append(macs, mac)
		}
	}

	if len(macs) == 0 {
		return "", nil, nil, nil
	}

	sql, createAdmitMacsRequest, deleteAdmitMacsRequest := admitMacsToInsertSqlAndPbRequest(macs)
	return sql, createAdmitMacsRequest, deleteAdmitMacsRequest, nil
}

func parseAdmitMac(tableHeaderFields, fields []string) *resource.AdmitMac {
	mac := &resource.AdmitMac{}
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameAdmitMac:
			mac.HwAddress = strings.TrimSpace(field)
		case FieldNameIsAdmitted:
			mac.IsAdmitted = internationalizationBool(strings.TrimSpace(field))
		case FieldNameComment:
			mac.Comment = field
		}
	}

	return mac
}

func checkAdmitMacConflictWithAdmitMacs(mac *resource.AdmitMac, macs []*resource.AdmitMac) error {
	for _, m := range macs {
		if m.HwAddress == mac.HwAddress {
			return errorno.ErrConflict(errorno.ErrNameMac, errorno.ErrNameMac,
				m.HwAddress, mac.HwAddress)
		}
	}

	return nil
}

func admitMacsToInsertSqlAndPbRequest(macs []*resource.AdmitMac) (string, *pbdhcpagent.CreateAdmitMacsRequest, *pbdhcpagent.DeleteAdmitMacsRequest) {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_admit_mac VALUES ")
	createAdmitMacRequests := make([]*pbdhcpagent.CreateAdmitMacRequest, 0, len(macs))
	deleteAdmitMacs := make([]string, 0, len(macs))
	for _, mac := range macs {
		buf.WriteString(admitMacToInsertDBSqlString(mac))
		createAdmitMacRequests = append(createAdmitMacRequests, admitMacToCreateAdmitMacRequest(mac))
		deleteAdmitMacs = append(deleteAdmitMacs, mac.HwAddress)
	}

	return strings.TrimSuffix(buf.String(), ",") + ";",
		&pbdhcpagent.CreateAdmitMacsRequest{Macs: createAdmitMacRequests},
		&pbdhcpagent.DeleteAdmitMacsRequest{HwAddresses: deleteAdmitMacs}
}

func sendCreateAdmitMacsCmdToDHCPAgent(createAdmitMacsRequest *pbdhcpagent.CreateAdmitMacsRequest, deleteAdmitMacsRequest *pbdhcpagent.DeleteAdmitMacsRequest) error {
	return kafka.SendDHCPCmd(kafka.CreateAdmitMacs,
		createAdmitMacsRequest, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
				kafka.DeleteAdmitMacs, deleteAdmitMacsRequest); err != nil {
				log.Warnf("batch create admit macs failed and rollback failed: %s", err.Error())
			}
		})
}

func (m *AdmitMacService) ExportExcel() (interface{}, error) {
	var macs []*resource.AdmitMac
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&macs,
			"select * from gr_admit_mac order by hw_address")
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameMac), pg.Error(err).Error())
	}

	strMatrix := make([][]string, 0, len(macs))
	for _, mac := range macs {
		strMatrix = append(strMatrix, localizationAdmitMacToStrSlice(mac))
	}

	if filepath, err := excel.WriteExcelFile(AdmitMacFileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderAdmitMac, strMatrix); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameMac), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (m *AdmitMacService) ExportExcelTemplate() (*excel.ExportFile, error) {
	if filepath, err := excel.WriteExcelFile(AdmitMacTemplateFileName,
		TableHeaderAdmitMac, TemplateAdmitMac); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameTemplate), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (d *AdmitMacService) BatchDelete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Exec("delete from gr_admit_mac where id in ('" +
			strings.Join(ids, "','") + "')"); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete,
				string(errorno.ErrNameMac), pg.Error(err).Error())
		} else if int(rows) != len(ids) {
			return errorno.ErrNotFound(errorno.ErrNameMac, ids[0])
		}

		return sendDeleteAdmitMacsCmdToDHCPAgent(ids)
	})
}

func sendDeleteAdmitMacsCmdToDHCPAgent(ids []string) error {
	return kafka.SendDHCPCmd(kafka.DeleteAdmitMacs,
		&pbdhcpagent.DeleteAdmitMacsRequest{HwAddresses: ids}, nil)
}
