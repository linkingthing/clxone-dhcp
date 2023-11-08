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
	SegmentFileNamePrefix       = "addresscode-layout-segment-"
	SegmentTemplateFileName     = "addresscode-layout-segment-template"
	SegmentImportFileNamePrefix = "addresscode-layout-segment-import"
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

		return sendCreateAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode.Name, string(layout.Label), addressCodeLayoutSegment)
	})
}

func sendCreateAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode, layout string, addressCodeLayoutSegment *resource.AddressCodeLayoutSegment) error {
	return kafka.SendDHCP6Cmd(kafka.CreateAddressCodeLayoutSegment,
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

		return sendDeleteAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode.Name, string(layout.Label), addressCodeLayoutSegments[0])
	})
}

func sendDeleteAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode, layout string, addressCodeLayoutSegment *resource.AddressCodeLayoutSegment) error {
	return kafka.SendDHCP6Cmd(kafka.DeleteAddressCodeLayoutSegment,
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
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, addressCodeLayoutSegment.GetID(), pg.Error(err).Error())
		}

		return sendUpdateAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode.Name, string(layout.Label),
			addressCodeLayoutSegments[0], addressCodeLayoutSegment)
	})
}

func sendUpdateAddressCodeLayoutSegmentCmdToDHCPAgent(addressCode, layout string, oldAddressCodeLayoutSegment, newAddressCodeLayoutSegment *resource.AddressCodeLayoutSegment) error {
	return kafka.SendDHCP6Cmd(kafka.UpdateAddressCodeLayoutSegment,
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

func (a *AddressCodeLayoutSegmentService) ImportExcel(addressCodeId, layoutId string, file *excel.ImportFile) (interface{}, error) {
	if len(file.Name) == 0 {
		return nil, nil
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(SegmentImportFileNamePrefix, TableHeaderSegmentFail, response)
	validSql, createSegmentsRequest, deleteSegmentsRequest, err := parseSegmentsFromFile(file.Name, addressCodeId, layoutId, response)
	if err != nil {
		return response, err
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Exec(validSql); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert,
				string(errorno.ErrNameAddressCodeLayoutSegment), pg.Error(err).Error())
		}

		return sendCreateSegmentsCmdToDHCPAgent(createSegmentsRequest, deleteSegmentsRequest)
	}); err != nil {
		return response, err
	}

	return response, nil
}

func parseSegmentsFromFile(fileName, addressCodeId, layoutId string, response *excel.ImportResult) (string, *pbdhcpagent.CreateAddressCodeLayoutSegmentsRequest, *pbdhcpagent.DeleteAddressCodeLayoutSegmentsRequest, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return "", nil, nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return "", nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0], TableHeaderSegment, SegmentMandatoryFields)
	if err != nil {
		return "", nil, nil, errorno.ErrInvalidTableHeader()
	}

	addressCode, layout, oldSegments, err := getSegmentAndParentResources(addressCodeId, layoutId)
	if err != nil {
		return "", nil, nil, err
	}

	response.InitData(len(contents) - 1)
	segments := make([]*resource.AddressCodeLayoutSegment, 0, len(contents)-1)
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, SegmentMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderSegmentFailLen,
				localizationSegmentToStrSlice(&resource.AddressCodeLayoutSegment{}),
				errorno.ErrMissingMandatory(j+2, SubnetMandatoryFields).ErrorCN())
			continue
		}

		segment := parseSegment(tableHeaderFields, fields)
		if err := segment.Validate(layout); err != nil {
			addFailDataToResponse(response, TableHeaderSegmentFailLen, localizationSegmentToStrSlice(segment),
				errorno.TryGetErrorCNMsg(err))
		} else if err := checkSegmentConflictWithSegments(segment, append(oldSegments, segments...)); err != nil {
			addFailDataToResponse(response, TableHeaderSegmentFailLen, localizationSegmentToStrSlice(segment),
				errorno.TryGetErrorCNMsg(err))
		} else {
			segments = append(segments, segment)
		}
	}

	if len(segments) == 0 {
		return "", nil, nil, nil
	}

	sql, createSegmentsRequest, deleteSegmentsRequest := segmentToInsertSqlAndPbRequest(segments, addressCode, layout)
	return sql, createSegmentsRequest, deleteSegmentsRequest, nil
}

func getSegmentAndParentResources(addressCodeId, layoutId string) (*resource.AddressCode, *resource.AddressCodeLayout, []*resource.AddressCodeLayoutSegment, error) {
	var oldSegments []*resource.AddressCodeLayoutSegment
	var addressCode *resource.AddressCode
	var layout *resource.AddressCodeLayout
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{
			resource.SqlColumnAddressCodeLayout: layoutId,
			resource.SqlOrderBy:                 resource.SqlColumnCode}, &oldSegments)
		if err != nil {
			return err
		}

		addressCode, err = getAddressCode(tx, addressCodeId)
		if err != nil {
			return err
		}

		layout, err = getAddressCodeLayout(tx, layoutId)
		return err
	}); err != nil {
		return nil, nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameAddressCodeLayoutSegment), pg.Error(err).Error())
	}

	return addressCode, layout, oldSegments, nil
}

func parseSegment(tableHeaderFields, fields []string) *resource.AddressCodeLayoutSegment {
	segment := &resource.AddressCodeLayoutSegment{}
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameCode:
			segment.Code = strings.TrimSpace(field)
		case FieldNameValue:
			segment.Value = strings.TrimSpace(field)
		}
	}

	return segment
}

func checkSegmentConflictWithSegments(segment *resource.AddressCodeLayoutSegment, segments []*resource.AddressCodeLayoutSegment) error {
	for _, s := range segments {
		if s.Code == segment.Code {
			return errorno.ErrConflict(errorno.ErrNameAddressCodeLayoutSegment, errorno.ErrNameAddressCodeLayoutSegment,
				s.Code, segment.Code)
		}
	}

	return nil
}

func segmentToInsertSqlAndPbRequest(segments []*resource.AddressCodeLayoutSegment, addressCode *resource.AddressCode, layout *resource.AddressCodeLayout) (string, *pbdhcpagent.CreateAddressCodeLayoutSegmentsRequest, *pbdhcpagent.DeleteAddressCodeLayoutSegmentsRequest) {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_address_code_layout_segment VALUES ")
	createSegmentRequests := make([]*pbdhcpagent.AddressCodeLayoutSegment, 0, len(segments))
	segmentCodes := make([]string, 0, len(segments))
	for _, segment := range segments {
		buf.WriteString(segmentToInsertDBSqlString(layout.GetID(), segment))
		createSegmentRequests = append(createSegmentRequests, &pbdhcpagent.AddressCodeLayoutSegment{
			Code: segment.Code, Value: segment.Value})
		segmentCodes = append(segmentCodes, segment.Code)
	}

	return strings.TrimSuffix(buf.String(), ",") + ";",
		&pbdhcpagent.CreateAddressCodeLayoutSegmentsRequest{
			AddressCode: addressCode.Name,
			Layout:      string(layout.Label),
			Segments:    createSegmentRequests},
		&pbdhcpagent.DeleteAddressCodeLayoutSegmentsRequest{
			AddressCode:  addressCode.Name,
			Layout:       string(layout.Label),
			SegmentCodes: segmentCodes,
		}
}

func sendCreateSegmentsCmdToDHCPAgent(createSegmentsRequest *pbdhcpagent.CreateAddressCodeLayoutSegmentsRequest, deleteSegmentsRequest *pbdhcpagent.DeleteAddressCodeLayoutSegmentsRequest) error {
	return kafka.SendDHCP6Cmd(kafka.CreateAddressCodeLayoutSegments,
		createSegmentsRequest, func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
				kafka.DeleteAddressCodeLayoutSegments, deleteSegmentsRequest); err != nil {
				log.Warnf("batch create segments failed and rollback failed: %s", err.Error())
			}
		})
}

func (a *AddressCodeLayoutSegmentService) ExportExcel(layoutId string) (interface{}, error) {
	var segments []*resource.AddressCodeLayoutSegment
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{
			resource.SqlColumnAddressCodeLayout: layoutId,
			resource.SqlOrderBy:                 resource.SqlColumnCode},
			&segments)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameAddressCodeLayoutSegment), pg.Error(err).Error())
	}

	strMatrix := make([][]string, 0, len(segments))
	for _, segment := range segments {
		strMatrix = append(strMatrix, localizationSegmentToStrSlice(segment))
	}

	if filepath, err := excel.WriteExcelFile(SegmentFileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderSegment, strMatrix); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameAddressCodeLayoutSegment), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (a *AddressCodeLayoutSegmentService) ExportExcelTemplate() (interface{}, error) {
	if filepath, err := excel.WriteExcelFile(SegmentTemplateFileName,
		TableHeaderSegment, TemplateSegment); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport, string(errorno.ErrNameTemplate), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (a *AddressCodeLayoutSegmentService) BatchDelete(addressCodeId, layoutId string, codes []string) error {
	if len(codes) == 0 {
		return nil
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		addressCode, err := getAddressCode(tx, addressCodeId)
		if err != nil {
			return err
		}

		layout, err := getAddressCodeLayout(tx, layoutId)
		if err != nil {
			return err
		}

		if rows, err := tx.Exec("delete from gr_address_code_layout_segment where address_code_layout = $1 and code in ('"+
			strings.Join(codes, "','")+"')", layoutId); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete,
				string(errorno.ErrNameAddressCodeLayoutSegment), pg.Error(err).Error())
		} else if int(rows) != len(codes) {
			return errorno.ErrNotFound(errorno.ErrNameAddressCodeLayoutSegment, codes[0])
		} else {
			return sendDeleteSegmentsCmdToDHCPAgent(&pbdhcpagent.DeleteAddressCodeLayoutSegmentsRequest{
				AddressCode:  addressCode.Name,
				Layout:       string(layout.Label),
				SegmentCodes: codes,
			})
		}
	})
}

func sendDeleteSegmentsCmdToDHCPAgent(deleteSegmentsRequest *pbdhcpagent.DeleteAddressCodeLayoutSegmentsRequest) error {
	return kafka.SendDHCP6Cmd(kafka.DeleteAddressCodeLayoutSegments, deleteSegmentsRequest, nil)
}
