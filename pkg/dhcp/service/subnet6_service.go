package service

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/golang/protobuf/proto"
	"github.com/linkingthing/cement/log"
	csvutil "github.com/linkingthing/clxone-utils/csv"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	Subnet6FileNamePrefix   = "subnet6-"
	Subnet6TemplateFileName = "subnet6-template"
)

type Subnet6Service struct{}

func NewSubnet6Service() *Subnet6Service {
	return &Subnet6Service{}
}

func (s *Subnet6Service) Create(subnet *resource.Subnet6) error {
	if err := subnet.Validate(); err != nil {
		return fmt.Errorf("validate subnet6 params invalid: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkSubnet6CouldBeCreated(tx, subnet.Subnet); err != nil {
			return err
		}

		if err := setSubnet6ID(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Insert(subnet); err != nil {
			return err
		}

		return sendCreateSubnet6CmdToDHCPAgent(subnet)
	}); err != nil {
		return fmt.Errorf("create subnet6 %s failed: %s", subnet.Subnet, err.Error())
	}

	return nil
}

func checkSubnet6CouldBeCreated(tx restdb.Transaction, subnet string) error {
	if count, err := tx.Count(resource.TableSubnet6, nil); err != nil {
		return fmt.Errorf("get subnet6s count failed: %s", err.Error())
	} else if count >= MaxSubnetsCount {
		return fmt.Errorf("subnet6s count has reached maximum (1w)")
	}

	var subnets []*resource.Subnet6
	if err := tx.FillEx(&subnets,
		"select * from gr_subnet6 where $1 && ipnet", subnet); err != nil {
		return fmt.Errorf("check subnet6 conflict failed: %s", err.Error())
	} else if len(subnets) != 0 {
		return fmt.Errorf("subnet6 conflict with subnet6 %s", subnets[0].Subnet)
	}

	return nil
}

func setSubnet6ID(tx restdb.Transaction, subnet *resource.Subnet6) error {
	var subnets []*resource.Subnet6
	if err := tx.Fill(map[string]interface{}{
		util.SqlOrderBy: "subnet_id desc", "offset": 0, "limit": 1},
		&subnets); err != nil {
		return err
	}

	if len(subnets) != 0 {
		subnet.SubnetId = subnets[0].SubnetId + 1
	} else {
		subnet.SubnetId = 1
	}

	subnet.SetID(strconv.FormatUint(subnet.SubnetId, 10))
	return nil
}

func sendCreateSubnet6CmdToDHCPAgent(subnet *resource.Subnet6) error {
	nodesForSucceed, err := kafka.SendDHCPCmdWithNodes(false, subnet.Nodes, kafka.CreateSubnet6,
		subnet6ToCreateSubnet6Request(subnet))
	if err != nil {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, kafka.DeleteSubnet6,
			&pbdhcpagent.DeleteSubnet6Request{Id: subnet.SubnetId}); err != nil {
			log.Errorf("create subnet6 %s failed, and rollback it failed: %s",
				subnet.Subnet, err.Error())
		}
	}

	return err
}

func subnet6ToCreateSubnet6Request(subnet *resource.Subnet6) *pbdhcpagent.CreateSubnet6Request {
	return &pbdhcpagent.CreateSubnet6Request{
		Id:                    subnet.SubnetId,
		Subnet:                subnet.Subnet,
		ValidLifetime:         subnet.ValidLifetime,
		MaxValidLifetime:      subnet.MaxValidLifetime,
		MinValidLifetime:      subnet.MinValidLifetime,
		PreferredLifetime:     subnet.PreferredLifetime,
		MinPreferredLifetime:  subnet.PreferredLifetime,
		MaxPreferredLifetime:  subnet.PreferredLifetime,
		RenewTime:             subnet.PreferredLifetime / 2,
		RebindTime:            subnet.PreferredLifetime * 3 / 4,
		ClientClass:           subnet.ClientClass,
		IfaceName:             subnet.IfaceName,
		RelayAgentAddresses:   subnet.RelayAgentAddresses,
		RelayAgentInterfaceId: subnet.RelayAgentInterfaceId,
		RapidCommit:           subnet.RapidCommit,
		UseEui64:              subnet.UseEui64,
		SubnetOptions:         pbSubnetOptionsFromSubnet6(subnet),
	}
}

func pbSubnetOptionsFromSubnet6(subnet *resource.Subnet6) []*pbdhcpagent.SubnetOption {
	var subnetOptions []*pbdhcpagent.SubnetOption
	if len(subnet.DomainServers) != 0 {
		subnetOptions = append(subnetOptions, &pbdhcpagent.SubnetOption{
			Name: "name-servers",
			Code: 23,
			Data: strings.Join(subnet.DomainServers, ","),
		})
	}

	return subnetOptions
}

func (s *Subnet6Service) List(ctx *restresource.Context) ([]*resource.Subnet6, error) {
	listCtx := genGetSubnetsContext(ctx, resource.TableSubnet6)
	var subnets []*resource.Subnet6
	var subnetsCount int
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if listCtx.hasPagination {
			if count, err := tx.CountEx(resource.TableSubnet6, listCtx.countSql,
				listCtx.params[:len(listCtx.params)-2]...); err != nil {
				return err
			} else {
				subnetsCount = int(count)
			}
		}

		return tx.FillEx(&subnets, listCtx.sql, listCtx.params...)
	}); err != nil {
		return nil, fmt.Errorf("list subnet6s failed:%s", err.Error())
	}

	if err := SetSubnet6sLeasesUsedInfo(subnets, listCtx.isUseIds()); err != nil {
		log.Warnf("set subnet6s leases used info failed: %s", err.Error())
	}

	if nodeNames, err := GetNodeNames(false); err != nil {
		log.Warnf("get node names failed: %s", err.Error())
	} else {
		setSubnet6sNodeNames(subnets, nodeNames)
	}

	setPagination(ctx, listCtx.hasPagination, subnetsCount)
	return subnets, nil
}

func SetSubnet6sLeasesUsedInfo(subnets []*resource.Subnet6, useIds bool) error {
	if len(subnets) == 0 {
		return nil
	}

	var resp *pbdhcpagent.GetSubnetsLeasesCountResponse
	var err error
	if useIds {
		var ids []uint64
		for _, subnet := range subnets {
			if subnet.Capacity != 0 {
				ids = append(ids, subnet.SubnetId)
			}
		}

		if len(ids) != 0 {
			resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets6LeasesCountWithIds(
				context.TODO(), &pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: ids})
		}
	} else {
		resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets6LeasesCount(
			context.TODO(), &pbdhcpagent.GetSubnetsLeasesCountRequest{})
	}

	if err != nil {
		return err
	}

	subnetsLeasesCount := resp.GetSubnetsLeasesCount()
	for _, subnet := range subnets {
		if subnet.Capacity != 0 {
			if leasesCount, ok := subnetsLeasesCount[subnet.SubnetId]; ok {
				subnet.UsedCount = leasesCount
				subnet.UsedRatio = fmt.Sprintf("%.4f",
					float64(leasesCount)/float64(subnet.Capacity))
			}
		}
	}

	return nil
}

func setSubnet6sNodeNames(subnets []*resource.Subnet6, nodeNames map[string]string) {
	for _, subnet := range subnets {
		subnet.NodeNames = getSubnetNodeNames(subnet.Nodes, nodeNames)
	}
}

func (s *Subnet6Service) Get(id string) (*resource.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &subnets)
	}); err != nil {
		return nil, fmt.Errorf("get subnet %s failed: %s", id, err.Error())
	} else if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet %s", id)
	}

	if err := setSubnet6LeasesUsedRatio(subnets[0]); err != nil {
		log.Warnf("get subnet6 %s leases used ratio failed: %s", id, err.Error())
	}

	if nodeNames, err := GetNodeNames(false); err != nil {
		log.Warnf("get node names failed: %s", err.Error())
	} else {
		subnets[0].NodeNames = getSubnetNodeNames(subnets[0].Nodes, nodeNames)
	}

	return subnets[0], nil
}

func setSubnet6LeasesUsedRatio(subnet *resource.Subnet6) error {
	leasesCount, err := getSubnet6LeasesCount(subnet)
	if err != nil {
		return err
	}

	if leasesCount != 0 {
		subnet.UsedCount = leasesCount
		subnet.UsedRatio = fmt.Sprintf("%.4f",
			float64(leasesCount)/float64(subnet.Capacity))
	}
	return nil
}

func getSubnet6LeasesCount(subnet *resource.Subnet6) (uint64, error) {
	if subnet.Capacity == 0 {
		return 0, nil
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6LeasesCount(context.TODO(),
		&pbdhcpagent.GetSubnet6LeasesCountRequest{Id: subnet.SubnetId})
	if err != nil {
		return 0, err
	}
	return resp.GetLeasesCount(), err
}

func (s *Subnet6Service) Update(subnet *resource.Subnet6) error {
	if err := subnet.ValidateParams(); err != nil {
		return fmt.Errorf("validate subnet6 params failed: %s", err.Error())
	}

	newUseEUI64 := subnet.UseEui64
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkUseEUI64(tx, subnet, newUseEUI64); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			resource.SqlColumnValidLifetime:         subnet.ValidLifetime,
			resource.SqlColumnMaxValidLifetime:      subnet.MaxValidLifetime,
			resource.SqlColumnMinValidLifetime:      subnet.MinValidLifetime,
			resource.SqlColumnPreferredLifetime:     subnet.PreferredLifetime,
			resource.SqlColumnDomainServers:         subnet.DomainServers,
			resource.SqlColumnClientClass:           subnet.ClientClass,
			resource.SqlColumnIfaceName:             subnet.IfaceName,
			resource.SqlColumnRelayAgentAddresses:   subnet.RelayAgentAddresses,
			resource.SqlColumnRelayAgentInterfaceId: subnet.RelayAgentInterfaceId,
			resource.SqlColumnTags:                  subnet.Tags,
			resource.SqlColumnRapidCommit:           subnet.RapidCommit,
			resource.SqlColumnUseEui64:              subnet.UseEui64,
			resource.SqlColumnCapacity:              subnet.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendUpdateSubnet6CmdToDHCPAgent(subnet)
	}); err != nil {
		return fmt.Errorf("update subnet6 %s failed: %s", subnet.GetID(), err.Error())
	}

	return nil
}

func setSubnet6FromDB(tx restdb.Transaction, subnet *resource.Subnet6) error {
	oldSubnet, err := getSubnet6FromDB(tx, subnet.GetID())
	if err != nil {
		return err
	}

	subnet.SubnetId = oldSubnet.SubnetId
	subnet.Capacity = oldSubnet.Capacity
	subnet.Subnet = oldSubnet.Subnet
	subnet.Ipnet = oldSubnet.Ipnet
	subnet.Nodes = oldSubnet.Nodes
	subnet.UseEui64 = oldSubnet.UseEui64
	return nil
}

func getSubnet6FromDB(tx restdb.Transaction, subnetId string) (*resource.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: subnetId},
		&subnets); err != nil {
		return nil, fmt.Errorf("get subnet6 %s from db failed: %s", subnetId, err.Error())
	} else if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet6 %s", subnetId)
	}

	return subnets[0], nil
}

func checkUseEUI64(tx restdb.Transaction, subnet *resource.Subnet6, newUseEUI64 bool) error {
	if newUseEUI64 {
		if ones, _ := subnet.Ipnet.Mask.Size(); ones != 64 {
			return fmt.Errorf("subnet6 use EUI64, mask size %d is not 64", ones)
		}

		if subnet.UseEui64 == false {
			if exists, err := subnetHasPools(tx, subnet); err != nil {
				return err
			} else if exists {
				return fmt.Errorf("subnet6 has pools, can not enabled use eui64")
			}
			subnet.Capacity = resource.MaxUint64
		}
	} else if subnet.UseEui64 {
		subnet.Capacity = 0
	}

	subnet.UseEui64 = newUseEUI64
	return nil
}

func subnetHasPools(tx restdb.Transaction, subnet *resource.Subnet6) (bool, error) {
	if subnet.Capacity != 0 {
		return true, nil
	}

	if exists, err := tx.Exists(resource.TableReservedPool6,
		map[string]interface{}{resource.SqlColumnSubnet6: subnet.GetID()}); err != nil {
		return false, err
	} else if exists {
		return true, nil
	}

	return tx.Exists(resource.TableReservedPdPool, map[string]interface{}{resource.SqlColumnSubnet6: subnet.GetID()})
}

func sendUpdateSubnet6CmdToDHCPAgent(subnet *resource.Subnet6) error {
	_, err := kafka.SendDHCPCmdWithNodes(false, subnet.Nodes, kafka.UpdateSubnet6,
		&pbdhcpagent.UpdateSubnet6Request{
			Id:                    subnet.SubnetId,
			Subnet:                subnet.Subnet,
			ValidLifetime:         subnet.ValidLifetime,
			MaxValidLifetime:      subnet.MaxValidLifetime,
			MinValidLifetime:      subnet.MinValidLifetime,
			PreferredLifetime:     subnet.PreferredLifetime,
			MinPreferredLifetime:  subnet.PreferredLifetime,
			MaxPreferredLifetime:  subnet.PreferredLifetime,
			RenewTime:             subnet.PreferredLifetime / 2,
			RebindTime:            subnet.PreferredLifetime * 3 / 4,
			ClientClass:           subnet.ClientClass,
			IfaceName:             subnet.IfaceName,
			RelayAgentAddresses:   subnet.RelayAgentAddresses,
			RelayAgentInterfaceId: subnet.RelayAgentInterfaceId,
			RapidCommit:           subnet.RapidCommit,
			UseEui64:              subnet.UseEui64,
			SubnetOptions:         pbSubnetOptionsFromSubnet6(subnet),
		})
	return err
}

func (s *Subnet6Service) Delete(subnet *resource.Subnet6) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkSubnet6CouldBeDelete(subnet); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableSubnet6,
			map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendDeleteSubnet6CmdToDHCPAgent(subnet, subnet.Nodes)
	}); err != nil {
		return fmt.Errorf("delete subnet6 %s failed: %s", subnet.GetID(), err.Error())
	}

	return nil
}

func checkSubnet6CouldBeDelete(subnet6 *resource.Subnet6) error {
	if leasesCount, err := getSubnet6LeasesCount(subnet6); err != nil {
		return fmt.Errorf("get subnet6 %s leases count failed: %s",
			subnet6.Subnet, err.Error())
	} else if leasesCount != 0 {
		return fmt.Errorf("can not delete subnet6 with %d ips had been allocated",
			leasesCount)
	} else {
		return nil
	}

}

func sendDeleteSubnet6CmdToDHCPAgent(subnet *resource.Subnet6, nodes []string) error {
	_, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeleteSubnet6,
		&pbdhcpagent.DeleteSubnet6Request{Id: subnet.SubnetId})
	return err
}

func (s *Subnet6Service) UpdateNodes(subnetID string, subnetNode *resource.SubnetNode) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnetID)
		if err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			resource.SqlColumnNodes: subnetNode.Nodes},
			map[string]interface{}{restdb.IDField: subnetID}); err != nil {
			return err
		}

		return sendUpdateSubnet6NodesCmdToDHCPAgent(tx, subnet6, subnetNode.Nodes)
	}); err != nil {
		return err
	}

	return nil
}

func (h *Subnet6Service) ImportCSV(file *csvutil.ImportFile) error {
	var oldSubnet6s []*resource.Subnet6
	if err := db.GetResources(map[string]interface{}{"orderby": "subnet_id desc"},
		&oldSubnet6s); err != nil {
		return fmt.Errorf("get subnet6s from db failed: %s", err.Error())
	}

	if len(oldSubnet6s) >= MaxSubnetsCount {
		return fmt.Errorf("subnet6s count has reached maximum (1w)")
	}

	validSqls, reqsForSentryCreate, reqsForSentryDelete,
		reqForServerCreate, reqForServerDelete, err := parseSubnet6sFromFile(file.Name, oldSubnet6s)
	if err != nil {
		return fmt.Errorf("parse subnet6s from file %s failed: %s",
			file.Name, err.Error())
	}

	if len(validSqls) == 0 {
		return nil
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, validSql := range validSqls {
			if _, err := tx.Exec(validSql); err != nil {
				return fmt.Errorf("batch insert subnet6s to db failed: %s",
					err.Error())
			}
		}

		return sendCreateSubnet6sAndPoolsCmdToDHCPAgent(reqsForSentryCreate, reqsForSentryDelete,
			reqForServerCreate, reqForServerDelete)
	}); err != nil {
		return fmt.Errorf("import subnet6s from file %s failed: %s",
			file.Name, err.Error())
	}

	return nil
}

func parseSubnet6sFromFile(fileName string, oldSubnets []*resource.Subnet6) ([]string, map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, map[string]*pbdhcpagent.DeleteSubnets6Request, *pbdhcpagent.CreateSubnets6AndPoolsRequest, *pbdhcpagent.DeleteSubnets6Request, error) {
	contents, err := csvutil.ReadCSVFile(fileName)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	if len(contents) < 2 {
		return nil, nil, nil, nil, nil, nil
	}

	tableHeaderFields, err := csvutil.ParseTableHeader(contents[0],
		TableHeaderSubnet6, SubnetMandatoryFields)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	oldSubnetsLen := len(oldSubnets)
	subnets := make([]*resource.Subnet6, 0)
	subnetPools := make(map[uint64][]*resource.Pool6)
	subnetReservedPools := make(map[uint64][]*resource.ReservedPool6)
	subnetReservations := make(map[uint64][]*resource.Reservation6)
	subnetPdPools := make(map[uint64][]*resource.PdPool)
	fieldcontents := contents[1:]
	for _, fieldcontent := range fieldcontents {
		fields, missingMandatory, emptyLine := csvutil.ParseTableFields(fieldcontent,
			tableHeaderFields, SubnetMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			log.Warnf("subnet6 missing mandatory fields subnet")
			continue
		}

		subnet, pools, reservedPools, reservations, pdpools, err := parseSubnet6sAndPools(
			tableHeaderFields, fields)
		if err != nil {
			log.Warnf("parse subnet6 %s fields failed: %s", subnet.Subnet, err.Error())
		} else if err := subnet.Validate(); err != nil {
			log.Warnf("subnet6 %s is invalid: %s", subnet.Subnet, err.Error())
		} else if err := checkSubnet6ConflictWithSubnet6s(subnet,
			append(oldSubnets, subnets...)); err != nil {
			log.Warnf(err.Error())
		} else if err := checkReservation6sValid(subnet, reservations); err != nil {
			log.Warnf("subnet6 %s reservation6s is invalid: %s", subnet.Subnet, err.Error())
		} else if err := checkReservedPool6sValid(subnet, reservedPools,
			reservations); err != nil {
			log.Warnf("subnet6 %s reserved pool6s is invalid: %s",
				subnet.Subnet, err.Error())
		} else if err := checkPool6sValid(subnet, pools, reservedPools,
			reservations); err != nil {
			log.Warnf("subnet6 %s pool6s is invalid: %s", subnet.Subnet, err.Error())
		} else if err := checkPdPoolsValid(subnet, pdpools, reservations); err != nil {
			log.Warnf("subnet6 %s pdpools is invalid: %s", subnet.Subnet, err.Error())
		} else {
			subnet.SubnetId = uint64(oldSubnetsLen + len(subnets) + 1)
			subnet.SetID(strconv.FormatUint(subnet.SubnetId, 10))
			subnets = append(subnets, subnet)
			if len(pools) != 0 {
				subnetPools[subnet.SubnetId] = pools
			}

			if len(reservedPools) != 0 {
				subnetReservedPools[subnet.SubnetId] = reservedPools
			}

			if len(reservations) != 0 {
				subnetReservations[subnet.SubnetId] = reservations
			}

			if len(pdpools) != 0 {
				subnetPdPools[subnet.SubnetId] = pdpools
			}
		}
	}

	if len(subnets) == 0 {
		return nil, nil, nil, nil, nil, nil
	}

	var sqls []string
	reqsForSentryCreate := make(map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest)
	reqForServerCreate := &pbdhcpagent.CreateSubnets6AndPoolsRequest{}
	reqsForSentryDelete := make(map[string]*pbdhcpagent.DeleteSubnets6Request)
	reqForServerDelete := &pbdhcpagent.DeleteSubnets6Request{}
	subnetAndNodes := make(map[uint64][]string)
	sqls = append(sqls,
		subnet6sToInsertSqlAndRequest(subnets, reqsForSentryCreate, reqForServerCreate,
			reqsForSentryDelete, reqForServerDelete, subnetAndNodes))
	if len(subnetPools) != 0 {
		sqls = append(sqls, pool6sToInsertSqlAndRequest(subnetPools,
			reqForServerCreate, reqsForSentryCreate, subnetAndNodes))
	}

	if len(subnetReservedPools) != 0 {
		sqls = append(sqls, reservedPool6sToInsertSqlAndRequest(subnetReservedPools,
			reqForServerCreate, reqsForSentryCreate, subnetAndNodes))
	}

	if len(subnetReservations) != 0 {
		sqls = append(sqls, reservation6sToInsertSqlAndRequest(subnetReservations,
			reqForServerCreate, reqsForSentryCreate, subnetAndNodes))
	}

	if len(subnetPdPools) != 0 {
		sqls = append(sqls, pdpoolsToInsertSqlAndRequest(subnetPdPools,
			reqForServerCreate, reqsForSentryCreate, subnetAndNodes))
	}

	return sqls, reqsForSentryCreate, reqsForSentryDelete, reqForServerCreate, reqForServerDelete, nil
}

func parseSubnet6sAndPools(tableHeaderFields, fields []string) (*resource.Subnet6, []*resource.Pool6, []*resource.ReservedPool6, []*resource.Reservation6, []*resource.PdPool, error) {
	subnet := &resource.Subnet6{}
	var pools []*resource.Pool6
	var reservedPools []*resource.ReservedPool6
	var reservations []*resource.Reservation6
	var pdpools []*resource.PdPool
	var err error
	for i, field := range fields {
		if csvutil.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameSubnet:
			subnet.Subnet = strings.TrimSpace(field)
		case FieldNameSubnetName:
			subnet.Tags = field
		case FieldNameEUI64:
			subnet.UseEui64 = eui64FromString(strings.TrimSpace(field))
		case FieldNameValidLifetime:
			if subnet.ValidLifetime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameMaxValidLifetime:
			if subnet.MaxValidLifetime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameMinValidLifetime:
			if subnet.MinValidLifetime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNamePreferredLifetime:
			if subnet.PreferredLifetime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameDomainServers:
			subnet.DomainServers = strings.Split(strings.TrimSpace(field), ",")
		case FieldNameIfaceName:
			subnet.IfaceName = strings.TrimSpace(field)
		case FieldNameRelayAddresses:
			subnet.RelayAgentAddresses = strings.Split(strings.TrimSpace(field), ",")
		case FieldNameOption16:
			subnet.ClientClass = field
		case FieldNameOption18:
			subnet.RelayAgentInterfaceId = field
		case FieldNameNodes:
			subnet.Nodes = strings.Split(strings.TrimSpace(field), ",")
		case FieldNamePools:
			if pools, err = parsePool6sFromString(strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameReservedPools:
			if reservedPools, err = parseReservedPool6sFromString(
				strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameReservations:
			if reservations, err = parseReservation6sFromString(strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNamePdPools:
			if pdpools, err = parsePdPoolsFromString(strings.TrimSpace(field)); err != nil {
				break
			}
		}
	}

	return subnet, pools, reservedPools, reservations, pdpools, err
}

func parsePool6sFromString(field string) ([]*resource.Pool6, error) {
	var pools []*resource.Pool6
	for _, poolStr := range strings.Split(field, ",") {
		if poolSlices := strings.SplitN(poolStr, "-", 3); len(poolSlices) != 3 {
			return nil, fmt.Errorf("parse subnet6 pool6 %s failed with wrong regexp",
				poolStr)
		} else {
			pools = append(pools, &resource.Pool6{
				BeginAddress: poolSlices[0],
				EndAddress:   poolSlices[1],
				Comment:      poolSlices[2],
			})
		}
	}

	return pools, nil
}

func parseReservedPool6sFromString(field string) ([]*resource.ReservedPool6, error) {
	var pools []*resource.ReservedPool6
	for _, poolStr := range strings.Split(field, ",") {
		if poolSlices := strings.SplitN(poolStr, "-", 3); len(poolSlices) != 3 {
			return nil, fmt.Errorf("parse subnet6 reserved pool6 %s failed with wrong regexp",
				poolStr)
		} else {
			pools = append(pools, &resource.ReservedPool6{
				BeginAddress: poolSlices[0],
				EndAddress:   poolSlices[1],
				Comment:      poolSlices[2],
			})
		}
	}

	return pools, nil
}

func parseReservation6sFromString(field string) ([]*resource.Reservation6, error) {
	var reservations []*resource.Reservation6
	for _, reservationStr := range strings.Split(field, ",") {
		if reservationSlices := strings.SplitN(reservationStr,
			"-", 5); len(reservationSlices) != 5 {
			return nil, fmt.Errorf("parse subnet6 reservation6 %s failed with wrong regexp",
				reservationStr)
		} else {
			reservation := &resource.Reservation6{
				Comment: reservationSlices[4],
			}
			if reservationSlices[0] == "duid" {
				reservation.Duid = reservationSlices[1]
			} else {
				reservation.HwAddress = reservationSlices[1]
			}

			if reservationSlices[2] == "ips" {
				reservation.IpAddresses = strings.Split(reservationSlices[3], "_")
			} else {
				reservation.Prefixes = strings.Split(reservationSlices[3], "_")
			}

			reservations = append(reservations, reservation)
		}
	}

	return reservations, nil
}

func parsePdPoolsFromString(field string) ([]*resource.PdPool, error) {
	var pdpools []*resource.PdPool
	for _, pdpoolStr := range strings.Split(field, ",") {
		if pdpoolSlices := strings.SplitN(pdpoolStr, "-", 4); len(pdpoolSlices) != 4 {
			return nil, fmt.Errorf("parse subnet6 pdpool %s failed with wrong regexp",
				pdpoolStr)
		} else {
			prefixLen, err := strconv.Atoi(pdpoolSlices[1])
			if err != nil {
				return nil, fmt.Errorf("parse subnet6 pdpool prefixlen %s failed: %s",
					pdpoolSlices[1], err.Error())
			}

			delegatedLen, err := strconv.Atoi(pdpoolSlices[2])
			if err != nil {
				return nil, fmt.Errorf("parse subnet6 pdpool delegatedlen %s failed: %s",
					pdpoolSlices[2], err.Error())
			}

			pdpools = append(pdpools, &resource.PdPool{
				Prefix:       pdpoolSlices[0],
				PrefixLen:    uint32(prefixLen),
				DelegatedLen: uint32(delegatedLen),
				Comment:      pdpoolSlices[3],
			})
		}
	}

	return pdpools, nil
}

func checkSubnet6ConflictWithSubnet6s(subnet6 *resource.Subnet6, subnets []*resource.Subnet6) error {
	for _, subnet := range subnets {
		if subnet.CheckConflictWithAnother(subnet6) {
			return fmt.Errorf("subnet6 %s conflict with subnet6 %s",
				subnet6.Subnet, subnet.Subnet)
		}
	}

	return nil
}

func checkReservation6sValid(subnet *resource.Subnet6, reservations []*resource.Reservation6) error {
	if len(reservations) == 0 {
		return nil
	}

	if subnet.UseEui64 {
		return fmt.Errorf("subnet6 use EUI64, can not create reservation6")
	}

	reservationFieldMap := make(map[string]struct{})
	for _, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return err
		}

		if err := checkReservation6BelongsToIpnet(subnet.Ipnet, reservation); err != nil {
			return err
		}

		if len(reservation.Duid) != 0 {
			if _, ok := reservationFieldMap[reservation.Duid]; ok {
				return fmt.Errorf("duplicate reservation6 with duid %s", reservation.Duid)
			} else {
				reservationFieldMap[reservation.Duid] = struct{}{}
			}
		} else {
			if _, ok := reservationFieldMap[reservation.HwAddress]; ok {
				return fmt.Errorf("duplicate reservation6 with mac %s", reservation.HwAddress)
			} else {
				reservationFieldMap[reservation.HwAddress] = struct{}{}
			}
		}

		if len(reservation.IpAddresses) != 0 {
			for _, ip := range reservation.IpAddresses {
				if _, ok := reservationFieldMap[ip]; ok {
					return fmt.Errorf("duplicate reservation6 with ip %s", ip)
				} else {
					reservationFieldMap[ip] = struct{}{}
				}
			}
		} else {
			for _, prefix := range reservation.Prefixes {
				if _, ok := reservationFieldMap[prefix]; ok {
					return fmt.Errorf("duplicate reservation6 with prefix %s", prefix)
				} else {
					reservationFieldMap[prefix] = struct{}{}
				}
			}
		}

		subnet.Capacity += reservation.Capacity
	}

	return nil
}

func checkReservedPool6sValid(subnet *resource.Subnet6, reservedPools []*resource.ReservedPool6, reservations []*resource.Reservation6) error {
	reservedPoolsLen := len(reservedPools)
	if reservedPoolsLen == 0 {
		return nil
	}

	if err := checkSubnet6IfCanCreateDynamicPool(subnet); err != nil {
		return err
	}

	for i := 0; i < reservedPoolsLen; i++ {
		if err := reservedPools[i].Validate(); err != nil {
			return err
		}

		if checkIPsBelongsToIpnet(subnet.Ipnet, reservedPools[i].BeginIp,
			reservedPools[i].EndIp) == false {
			return fmt.Errorf("reserved pool6 %s not belongs to subnet6 %s",
				reservedPools[i].String(), subnet.Subnet)
		}

		for j := i + 1; j < reservedPoolsLen; j++ {
			if reservedPools[i].CheckConflictWithAnother(reservedPools[j]) {
				return fmt.Errorf("reserved pool6 %s conflict with another %s",
					reservedPools[i].String(), reservedPools[j].String())
			}
		}

		if err := checkReservedPool6ConflictWithReservation6s(reservedPools[i],
			reservations); err != nil {
			return err
		}
	}

	return nil
}

func checkPool6sValid(subnet *resource.Subnet6, pools []*resource.Pool6, reservedPools []*resource.ReservedPool6, reservations []*resource.Reservation6) error {
	poolsLen := len(pools)
	if poolsLen == 0 {
		return nil
	}

	if err := checkSubnet6IfCanCreateDynamicPool(subnet); err != nil {
		return err
	}

	for i := 0; i < poolsLen; i++ {
		if err := pools[i].Validate(); err != nil {
			return err
		}

		if checkIPsBelongsToIpnet(subnet.Ipnet,
			pools[i].BeginIp, pools[i].EndIp) == false {
			return fmt.Errorf("pool6 %s not belongs to subnet6 %s",
				pools[i].String(), subnet.Subnet)
		}

		for j := i + 1; j < poolsLen; j++ {
			if pools[i].CheckConflictWithAnother(pools[j]) {
				return fmt.Errorf("pool6 %s conflict with another %s",
					pools[i].String(), pools[j].String())
			}
		}

		recalculatePool6CapacityWithReservations(pools[i], reservations)
		recalculatePool6CapacityWithReservedPools(pools[i], reservedPools)
		subnet.Capacity += pools[i].Capacity
	}

	return nil
}

func checkPdPoolsValid(subnet *resource.Subnet6, pdpools []*resource.PdPool, reservations []*resource.Reservation6) error {
	pdpoolsLen := len(pdpools)
	if pdpoolsLen == 0 {
		return nil
	}

	if subnet.UseEui64 {
		return fmt.Errorf("subnet6 use EUI64, can not create pdpool")
	}

	for i := 0; i < pdpoolsLen; i++ {
		if err := pdpools[i].Validate(); err != nil {
			return err
		}

		if err := checkPrefixBelongsToIpnet(subnet.Ipnet, pdpools[i].PrefixIpnet,
			pdpools[i].PrefixLen); err != nil {
			return err
		}

		for j := i + 1; j < pdpoolsLen; j++ {
			if pdpools[i].CheckConflictWithAnother(pdpools[j]) {
				return fmt.Errorf("pdpool %s conflict with another %s",
					pdpools[i].String(), pdpools[j].String())
			}
		}

		recalculatePdPoolCapacityWithReservations(pdpools[i], reservations)
		subnet.Capacity += pdpools[i].Capacity
	}

	return nil
}

func subnet6sToInsertSqlAndRequest(subnets []*resource.Subnet6, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryDelete map[string]*pbdhcpagent.DeleteSubnets6Request, reqForServerDelete *pbdhcpagent.DeleteSubnets6Request, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_subnet6 values ")
	for _, subnet := range subnets {
		buf.WriteString(subnet6ToInsertDBSqlString(subnet))
		nodes := subnetAndNodes[subnet.SubnetId]
		nodes = append(nodes, subnet.Nodes...)
		subnetAndNodes[subnet.SubnetId] = nodes
		pbSubnet := subnet6ToCreateSubnet6Request(subnet)
		reqForServerCreate.Subnets = append(reqForServerCreate.Subnets, pbSubnet)
		reqForServerDelete.Ids = append(reqForServerDelete.Ids, subnet.SubnetId)
		for _, node := range subnet.Nodes {
			createReq, ok := reqsForSentryCreate[node]
			deleteReq := reqsForSentryDelete[node]
			if ok == false {
				createReq = &pbdhcpagent.CreateSubnets6AndPoolsRequest{}
				deleteReq = &pbdhcpagent.DeleteSubnets6Request{}
			}
			createReq.Subnets = append(createReq.Subnets, pbSubnet)
			deleteReq.Ids = append(deleteReq.Ids, subnet.SubnetId)
			reqsForSentryCreate[node] = createReq
			reqsForSentryDelete[node] = deleteReq
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func pool6sToInsertSqlAndRequest(subnetPools map[uint64][]*resource.Pool6, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_pool6 values ")
	for subnetId, pools := range subnetPools {
		for _, pool := range pools {
			buf.WriteString(pool6ToInsertDBSqlString(subnetId, pool))
			pbPool := pool6ToCreatePool6Request(subnetId, pool)
			reqForServerCreate.Pools = append(reqForServerCreate.Pools, pbPool)
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqsForSentryCreate[node]; ok {
					req.Pools = append(req.Pools, pbPool)
				}
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func reservedPool6sToInsertSqlAndRequest(subnetReservedPools map[uint64][]*resource.ReservedPool6, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_reserved_pool6 values ")
	for subnetId, pools := range subnetReservedPools {
		for _, pool := range pools {
			buf.WriteString(reservedPool6ToInsertDBSqlString(subnetId, pool))
			pbReservedPool := reservedPool6ToCreateReservedPool6Request(subnetId, pool)
			reqForServerCreate.ReservedPools = append(reqForServerCreate.ReservedPools, pbReservedPool)
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqsForSentryCreate[node]; ok {
					req.ReservedPools = append(req.ReservedPools, pbReservedPool)
				}
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func reservation6sToInsertSqlAndRequest(subnetReservations map[uint64][]*resource.Reservation6, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_reservation6 values ")
	for subnetId, reservations := range subnetReservations {
		for _, reservation := range reservations {
			buf.WriteString(reservation6ToInsertDBSqlString(subnetId, reservation))
			pbReservation := reservation6ToCreateReservation6Request(subnetId, reservation)
			reqForServerCreate.Reservations = append(reqForServerCreate.Reservations, pbReservation)
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqsForSentryCreate[node]; ok {
					req.Reservations = append(req.Reservations, pbReservation)
				}
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func pdpoolsToInsertSqlAndRequest(subnetPdPools map[uint64][]*resource.PdPool, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_pd_pool values ")
	for subnetId, pdpools := range subnetPdPools {
		for _, pdpool := range pdpools {
			buf.WriteString(pdpoolToInsertDBSqlString(subnetId, pdpool))
			pbPdPool := pdPoolToCreatePdPoolRequest(subnetId, pdpool)
			reqForServerCreate.PdPools = append(reqForServerCreate.PdPools, pbPdPool)
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqsForSentryCreate[node]; ok {
					req.PdPools = append(req.PdPools, pbPdPool)
				}
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func sendCreateSubnet6sAndPoolsCmdToDHCPAgent(reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets6AndPoolsRequest, reqsForSentryDelete map[string]*pbdhcpagent.DeleteSubnets6Request, reqForServerCreate *pbdhcpagent.CreateSubnets6AndPoolsRequest, reqForServerDelete *pbdhcpagent.DeleteSubnets6Request) error {
	if len(reqsForSentryCreate) == 0 {
		return nil
	}

	var sentryNodes []string
	for node := range reqsForSentryCreate {
		sentryNodes = append(sentryNodes, node)
	}

	nodes, err := kafka.GetDHCPNodes(sentryNodes, false)
	if err != nil {
		return err
	}

	var succeedSentryNodes []string
	for node, req := range reqsForSentryCreate {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			[]string{node}, kafka.CreateSubnet6sAndPools,
			req); err != nil {
			deleteSentrySubnet6s(reqsForSentryDelete, succeedSentryNodes)
			return err
		}

		succeedSentryNodes = append(succeedSentryNodes, node)
	}

	var succeedServerNodes []string
	for _, node := range nodes[len(sentryNodes):] {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			[]string{node}, kafka.CreateSubnet6sAndPools,
			reqForServerCreate); err != nil {
			deleteSentrySubnet6s(reqsForSentryDelete, succeedSentryNodes)
			deleteServerSubnet6s(reqForServerDelete, succeedServerNodes)
			return err
		}

		succeedServerNodes = append(succeedServerNodes, node)
	}

	return nil
}

func deleteSentrySubnet6s(reqs map[string]*pbdhcpagent.DeleteSubnets6Request, nodes []string) {
	for _, node := range nodes {
		if req, ok := reqs[node]; ok {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				[]string{node}, kafka.DeleteSubnet6s, req); err != nil {
				log.Errorf("delete sentry subnet6s with node %s when rollback failed: %s",
					node, err.Error())
			}
		}
	}
}

func deleteServerSubnet6s(req *pbdhcpagent.DeleteSubnets6Request, nodes []string) {
	for _, node := range nodes {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			[]string{node}, kafka.DeleteSubnet6s, req); err != nil {
			log.Errorf("delete server subnet6s with node %s when rollback failed: %s",
				node, err.Error())
		}
	}
}

func (s *Subnet6Service) ExportCSV() (*csvutil.ExportFile, error) {
	var subnet6s []*resource.Subnet6
	var pools []*resource.Pool6
	var reservedPools []*resource.ReservedPool6
	var reservations []*resource.Reservation6
	var pdpools []*resource.PdPool
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{util.SqlOrderBy: resource.SqlColumnSubnetId},
			&subnet6s); err != nil {
			return err
		}

		if err := tx.Fill(nil, &pools); err != nil {
			return err
		}

		if err := tx.Fill(nil, &reservedPools); err != nil {
			return err
		}

		if err := tx.Fill(nil, &reservations); err != nil {
			return err
		}

		return tx.Fill(nil, &pdpools)
	}); err != nil {
		return nil, fmt.Errorf("export subnet6 failed: %s", err.Error())
	}

	subnetPools := make(map[string][]string)
	for _, pool := range pools {
		poolSlices := subnetPools[pool.Subnet6]
		poolSlices = append(poolSlices, pool.String()+"-"+pool.Comment)
		subnetPools[pool.Subnet6] = poolSlices
	}

	subnetReservedPools := make(map[string][]string)
	for _, reservedPool := range reservedPools {
		reservedPoolSlices := subnetReservedPools[reservedPool.Subnet6]
		reservedPoolSlices = append(reservedPoolSlices, reservedPool.String()+"-"+reservedPool.Comment)
		subnetReservedPools[reservedPool.Subnet6] = reservedPoolSlices
	}

	subnetReservations := make(map[string][]string)
	for _, reservation := range reservations {
		reservationSlices := subnetReservations[reservation.Subnet6]
		reservationSlices = append(reservationSlices,
			reservation.String()+"-"+reservation.AddrString()+"-"+reservation.Comment)
		subnetReservations[reservation.Subnet6] = reservationSlices
	}

	subnetPdPools := make(map[string][]string)
	for _, pdpool := range pdpools {
		pdpoolSlices := subnetPdPools[pdpool.Subnet6]
		pdpoolSlices = append(pdpoolSlices, pdpool.String()+"-"+pdpool.Comment)
		subnetPools[pdpool.Subnet6] = pdpoolSlices
	}

	var strMatrix [][]string
	for _, subnet6 := range subnet6s {
		subnetSlices := localizationSubnet6ToStrSlice(subnet6)
		slices := make([]string, TableHeaderSubnet6Len)
		copy(slices, subnetSlices)
		if poolSlices, ok := subnetPools[subnet6.GetID()]; ok {
			slices[TableHeaderSubnet6Len-4] = strings.Join(poolSlices, ",")
		}

		if reservedPools, ok := subnetReservedPools[subnet6.GetID()]; ok {
			slices[TableHeaderSubnet6Len-3] = strings.Join(reservedPools, ",")
		}

		if reservations, ok := subnetReservations[subnet6.GetID()]; ok {
			slices[TableHeaderSubnet6Len-2] = strings.Join(reservations, ",")
		}

		if pdpools, ok := subnetPdPools[subnet6.GetID()]; ok {
			slices[TableHeaderSubnet6Len-1] = strings.Join(pdpools, ",")
		}

		strMatrix = append(strMatrix, slices)
	}

	if filepath, err := csvutil.WriteCSVFile(Subnet6FileNamePrefix+
		time.Now().Format(csvutil.TimeFormat), TableHeaderSubnet6, strMatrix); err != nil {
		return nil, fmt.Errorf("export subnet6 failed: %s", err.Error())
	} else {
		return &csvutil.ExportFile{Path: filepath}, nil
	}
}

func (s *Subnet6Service) ExportCSVTemplate() (*csvutil.ExportFile, error) {
	if filepath, err := csvutil.WriteCSVFile(Subnet6TemplateFileName,
		TableHeaderSubnet6, nil); err != nil {
		return nil, fmt.Errorf("export subnet6 template failed: %s", err.Error())
	} else {
		return &csvutil.ExportFile{Path: filepath}, nil
	}
}

func sendUpdateSubnet6NodesCmdToDHCPAgent(tx restdb.Transaction, subnet6 *resource.Subnet6, newNodes []string) error {
	if len(subnet6.Nodes) == 0 && len(newNodes) == 0 {
		return nil
	}

	if len(subnet6.Nodes) != 0 && len(newNodes) == 0 {
		if err := checkSubnet6CouldBeDelete(subnet6); err != nil {
			return err
		}
	}

	nodesForDelete, nodesForCreate, err := getChangedNodes(subnet6.Nodes, newNodes, false)
	if err != nil {
		return err
	}

	if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
		nodesForDelete, kafka.DeleteSubnet6,
		&pbdhcpagent.DeleteSubnet6Request{Id: subnet6.SubnetId}); err != nil {
		return err
	}

	if len(nodesForCreate) == 0 {
		return nil
	}

	req, cmd, err := genCreateSubnets6AndPoolsRequestWithSubnet6(tx, subnet6)
	if err != nil {
		return err
	}

	if succeedNodes, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
		nodesForCreate, cmd, req); err != nil {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			succeedNodes, kafka.DeleteSubnet6,
			&pbdhcpagent.DeleteSubnet6Request{Id: subnet6.SubnetId}); err != nil {
			log.Errorf("delete subnet6 %s with node %v when rollback failed: %s",
				subnet6.Subnet, succeedNodes, err.Error())
		}
		return err
	}

	return nil
}

func genCreateSubnets6AndPoolsRequestWithSubnet6(tx restdb.Transaction, subnet6 *resource.Subnet6) (proto.Message, kafka.DHCPCmd, error) {
	var pools []*resource.Pool6
	var reservedPools []*resource.ReservedPool6
	var reservations []*resource.Reservation6
	var pdpools []*resource.PdPool
	var reservedPdPools []*resource.ReservedPdPool
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet6.GetID()},
		&pools); err != nil {
		return nil, "", err
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet6.GetID()},
		&reservedPools); err != nil {
		return nil, "", err
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet6.GetID()},
		&reservations); err != nil {
		return nil, "", err
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet6.GetID()},
		&pdpools); err != nil {
		return nil, "", err
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet6.GetID()},
		&reservedPdPools); err != nil {
		return nil, "", err
	}

	if len(pools) == 0 && len(reservedPools) == 0 && len(reservations) == 0 &&
		len(pdpools) == 0 && len(reservedPdPools) == 0 {
		return subnet6ToCreateSubnet6Request(subnet6), kafka.CreateSubnet6, nil
	}

	req := &pbdhcpagent.CreateSubnets6AndPoolsRequest{
		Subnets: []*pbdhcpagent.CreateSubnet6Request{subnet6ToCreateSubnet6Request(subnet6)},
	}
	for _, pool := range pools {
		req.Pools = append(req.Pools, pool6ToCreatePool6Request(subnet6.SubnetId, pool))
	}

	for _, pool := range reservedPools {
		req.ReservedPools = append(req.ReservedPools,
			reservedPool6ToCreateReservedPool6Request(subnet6.SubnetId, pool))
	}

	for _, reservation := range reservations {
		req.Reservations = append(req.Reservations,
			reservation6ToCreateReservation6Request(subnet6.SubnetId, reservation))
	}

	for _, pdpool := range pdpools {
		req.PdPools = append(req.PdPools,
			pdPoolToCreatePdPoolRequest(subnet6.SubnetId, pdpool))
	}

	for _, pdpool := range reservedPdPools {
		req.ReservedPdPools = append(req.ReservedPdPools,
			reservedPdPoolToCreateReservedPdPoolRequest(subnet6.SubnetId, pdpool))
	}

	return req, kafka.CreateSubnet6sAndPools, nil
}

func (s *Subnet6Service) CouldBeCreated(couldBeCreatedSubnet *resource.CouldBeCreatedSubnet) (interface{}, error) {
	if _, err := gohelperip.ParseCIDRv6(couldBeCreatedSubnet.Subnet); err != nil {
		return nil, fmt.Errorf("action check subnet could be created input subnet %s invalid: %s",
			couldBeCreatedSubnet.Subnet, err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkSubnet6CouldBeCreated(tx, couldBeCreatedSubnet.Subnet)
	}); err != nil {
		return nil, fmt.Errorf("action check subnet6 could be created: %s", err.Error())
	}

	return nil, nil
}

func (s *Subnet6Service) ListWithSubnets(subnetListInput *resource.SubnetListInput) (*resource.Subnet6ListOutput, error) {
	for _, subnet := range subnetListInput.Subnets {
		if _, err := gohelperip.ParseCIDRv6(subnet); err != nil {
			return nil, fmt.Errorf("action check subnet could be created input subnet %s invalid: %s",
				subnet, err.Error())
		}
	}

	subnets, err := ListSubnet6sByPrefixes(subnetListInput.Subnets)
	if err != nil {
		return nil, err
	}

	return &resource.Subnet6ListOutput{Subnet6s: subnets}, nil
}

func ListSubnet6sByPrefixes(prefixes []string) ([]*resource.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "select * from gr_subnet6 where subnet = any ($1)", prefixes)
	}); err != nil {
		return nil, fmt.Errorf("action list subnet6s failed: %s", err.Error())
	}

	if err := SetSubnet6sLeasesUsedInfo(subnets, true); err != nil {
		log.Warnf("set subnet6s leases used info failed: %s", err.Error())
	}

	return subnets, nil
}

func GetSubnet6ByIP(ip string) (*resource.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "select * from gr_subnet6 where ipnet >>= $1", ip)
	}); err != nil {
		return nil, err
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("not found subnet of ip %s", ip)
	} else {
		return subnets[0], nil
	}
}

func GetSubnet6ByPrefix(prefix string) (*resource.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "select * from gr_subnet6 where subnet = $1", prefix)
	}); err != nil {
		return nil, err
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("not found subnet of prefix %s", prefix)
	} else {
		return subnets[0], nil
	}
}
