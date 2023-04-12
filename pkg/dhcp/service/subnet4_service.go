package service

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/golang/protobuf/proto"
	"github.com/linkingthing/cement/log"
	"github.com/linkingthing/cement/slice"
	"github.com/linkingthing/clxone-utils/excel"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	MaxSubnetsCount = 10000

	Subnet4FileNamePrefix       = "subnet4-"
	Subnet4TemplateFileName     = "subnet4-template"
	Subnet4ImportFileNamePrefix = "subnet4-import"

	FilterNameExcludeShared  = "exclude_shared"
	FilterNameSharedNetwork4 = "shared_network4"
)

type Subnet4Service struct {
}

func NewSubnet4Service() *Subnet4Service {
	return &Subnet4Service{}
}

func (s *Subnet4Service) Create(subnet *resource.Subnet4) error {
	if err := subnet.Validate(nil, nil); err != nil {
		return fmt.Errorf("validate subnet4 params invalid: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkSubnet4CouldBeCreated(tx, subnet.Subnet); err != nil {
			return err
		}

		if err := setSubnet4ID(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Insert(subnet); err != nil {
			return pg.Error(err)
		}

		return sendCreateSubnet4CmdToDHCPAgent(subnet)
	}); err != nil {
		return fmt.Errorf("create subnet4 %s failed: %s", subnet.Subnet, err.Error())
	}

	return nil
}

func checkSubnet4CouldBeCreated(tx restdb.Transaction, subnet string) error {
	if count, err := tx.Count(resource.TableSubnet4, nil); err != nil {
		return fmt.Errorf("get subnet4s count failed: %s", pg.Error(err).Error())
	} else if count >= MaxSubnetsCount {
		return fmt.Errorf("subnet4s count has reached maximum (1w)")
	}

	var subnets []*resource.Subnet4
	if err := tx.FillEx(&subnets,
		"SELECT * FROM gr_subnet4 WHERE $1 && ipnet", subnet); err != nil {
		return fmt.Errorf("check subnet4 conflict failed: %s", pg.Error(err).Error())
	} else if len(subnets) != 0 {
		return fmt.Errorf("conflict with subnet4 %s", subnets[0].Subnet)
	}

	return nil
}

func setSubnet4ID(tx restdb.Transaction, subnet *resource.Subnet4) error {
	var subnets []*resource.Subnet4
	if err := tx.Fill(map[string]interface{}{
		resource.SqlOrderBy: "subnet_id desc",
		resource.SqlOffset:  0,
		resource.SqlLimit:   1},
		&subnets); err != nil {
		return pg.Error(err)
	}

	if len(subnets) != 0 {
		subnet.SubnetId = subnets[0].SubnetId + 1
	} else {
		subnet.SubnetId = 1
	}

	subnet.SetID(strconv.FormatUint(subnet.SubnetId, 10))
	return nil
}

func sendCreateSubnet4CmdToDHCPAgent(subnet *resource.Subnet4) error {
	return kafka.SendDHCPCmdWithNodes(true, subnet.Nodes, kafka.CreateSubnet4,
		subnet4ToCreateSubnet4Request(subnet), func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteSubnet4,
				&pbdhcpagent.DeleteSubnet4Request{Id: subnet.SubnetId}); err != nil {
				log.Errorf("create subnet4 %s failed, and rollback with nodes %v failed: %s",
					subnet.Subnet, nodesForSucceed, err.Error())
			}
		})
}

func subnet4ToCreateSubnet4Request(subnet *resource.Subnet4) *pbdhcpagent.CreateSubnet4Request {
	return &pbdhcpagent.CreateSubnet4Request{
		Id:                  subnet.SubnetId,
		Subnet:              subnet.Subnet,
		ValidLifetime:       subnet.ValidLifetime,
		MaxValidLifetime:    subnet.MaxValidLifetime,
		MinValidLifetime:    subnet.MinValidLifetime,
		RenewTime:           subnet.ValidLifetime / 2,
		RebindTime:          subnet.ValidLifetime * 3 / 4,
		WhiteClientClasses:  subnet.WhiteClientClasses,
		BlackClientClasses:  subnet.BlackClientClasses,
		IfaceName:           subnet.IfaceName,
		RelayAgentAddresses: subnet.RelayAgentAddresses,
		NextServer:          subnet.NextServer,
		SubnetOptions:       pbSubnetOptionsFromSubnet4(subnet),
	}
}

func pbSubnetOptionsFromSubnet4(subnet *resource.Subnet4) []*pbdhcpagent.SubnetOption {
	var subnetOptions []*pbdhcpagent.SubnetOption
	if len(subnet.SubnetMask) != 0 {
		subnetOptions = append(subnetOptions, &pbdhcpagent.SubnetOption{
			Name: "subnet-mask",
			Code: 1,
			Data: subnet.SubnetMask,
		})
	}

	if len(subnet.Routers) != 0 {
		subnetOptions = append(subnetOptions, &pbdhcpagent.SubnetOption{
			Name: "routers",
			Code: 3,
			Data: strings.Join(subnet.Routers, ","),
		})
	}

	if len(subnet.DomainServers) != 0 {
		subnetOptions = append(subnetOptions, &pbdhcpagent.SubnetOption{
			Name: "name-servers",
			Code: 6,
			Data: strings.Join(subnet.DomainServers, ","),
		})
	}

	if subnet.TftpServer != "" {
		subnetOptions = append(subnetOptions, &pbdhcpagent.SubnetOption{
			Name: "tftp-server",
			Code: 66,
			Data: subnet.TftpServer,
		})
	}

	if subnet.Bootfile != "" {
		subnetOptions = append(subnetOptions, &pbdhcpagent.SubnetOption{
			Name: "bootfile",
			Code: 67,
			Data: subnet.Bootfile,
		})
	}

	if subnet.Ipv6OnlyPreferred != 0 {
		subnetOptions = append(subnetOptions, &pbdhcpagent.SubnetOption{
			Name: "ipv6-only-perferred",
			Code: 108,
			Data: uint32ToString(subnet.Ipv6OnlyPreferred),
		})
	}

	return subnetOptions
}

func (s *Subnet4Service) List(ctx *restresource.Context) ([]*resource.Subnet4, error) {
	listCtx := genGetSubnetsContext(ctx, resource.TableSubnet4)
	var subnets []*resource.Subnet4
	var subnetsCount int
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if listCtx.hasPagination {
			if count, err := tx.CountEx(resource.TableSubnet4, listCtx.countSql,
				listCtx.params[:len(listCtx.params)-2]...); err != nil {
				return err
			} else {
				subnetsCount = int(count)
			}
		}

		return tx.FillEx(&subnets, listCtx.sql, listCtx.params...)
	}); err != nil {
		return nil, fmt.Errorf("list subnet4s failed: %s", pg.Error(err).Error())
	}

	if len(subnets) > 0 && listCtx.needSetSubnetsLeasesUsedInfo() {
		if err := SetSubnet4UsedInfo(subnets, listCtx.isUseIds()); err != nil {
			log.Warnf("set subnet4s leases used info failed: %s", err.Error())
		}
	}

	if nodeNames, err := GetAgentInfo(false, kafka.AgentRoleSentry4); err != nil {
		log.Warnf("get node names failed: %s", err.Error())
	} else {
		setSubnet4sNodeNames(subnets, nodeNames)
	}

	setPagination(ctx, listCtx.hasPagination, subnetsCount)
	return subnets, nil
}

type listSubnetContext struct {
	countSql        string
	sql             string
	params          []interface{}
	hasFilterSubnet bool
	hasPagination   bool
	hasExclude      bool
	hasShared       bool
}

func (l listSubnetContext) isUseIds() bool {
	return l.hasPagination || l.hasFilterSubnet
}

func (l listSubnetContext) needSetSubnetsLeasesUsedInfo() bool {
	return !l.hasExclude && !l.hasShared
}

func genGetSubnetsContext(ctx *restresource.Context, table restdb.ResourceType) listSubnetContext {
	seq := 1
	listCtx := listSubnetContext{}
	var subnetState string
	var sharedNetworkState string
	var excludeSharedState string
	var excludeNodesState string
	for _, filter := range ctx.GetFilters() {
		switch filter.Name {
		case util.FilterNameSubnet:
			if value, ok := util.GetFilterValueWithEqModifierFromFilter(filter); ok {
				listCtx.hasFilterSubnet = true
				listCtx.params = append(listCtx.params, value)
				subnetState = "subnet = $" + strconv.Itoa(seq)
				seq += 1
			}
		case FilterNameExcludeShared:
			if value, ok := util.GetFilterValueWithEqModifierFromFilter(filter); ok &&
				value == "true" {
				listCtx.hasExclude = true
				excludeNodesState = "nodes != '{}'"
				excludeSharedState =
					"subnet_id not in (select subnet_id from gr_shared_network4 where subnet_id=any(subnet_ids))"
			}
		case FilterNameSharedNetwork4:
			if value, ok := util.GetFilterValueWithEqModifierFromFilter(filter); ok {
				listCtx.hasShared = true
				sharedNetworkState =
					"subnet_id = any((select subnet_ids from gr_shared_network4 where name = $" +
						strconv.Itoa(seq) + ")::numeric[])"
				listCtx.params = append(listCtx.params, value)
				seq += 1
			}
		}
	}

	sqls := []string{"SELECT * FROM gr_" + string(table)}
	var whereStates []string
	if listCtx.hasFilterSubnet {
		whereStates = append(whereStates, subnetState)
	}

	if listCtx.hasExclude {
		whereStates = append(whereStates, excludeNodesState)
	}

	if listCtx.hasExclude && listCtx.hasShared {
		whereStates = append(whereStates,
			"("+strings.Join([]string{sharedNetworkState, excludeSharedState}, " or ")+")")
	} else if listCtx.hasExclude {
		whereStates = append(whereStates, excludeSharedState)
	} else if listCtx.hasShared {
		whereStates = append(whereStates, sharedNetworkState)
	}

	if len(whereStates) != 0 {
		sqls = append(sqls, "where")
		sqls = append(sqls, strings.Join(whereStates, " and "))
	}

	listCtx.countSql = strings.Replace(strings.Join(sqls, " "), "*", "count(*)", 1)
	if !listCtx.hasFilterSubnet {
		sqls = append(sqls, "order by subnet_id")
		if pagination := ctx.GetPagination(); pagination.PageSize > 0 &&
			pagination.PageNum > 0 {
			listCtx.hasPagination = true
			sqls = append(sqls, "limit $"+strconv.Itoa(seq))
			seq += 1
			sqls = append(sqls, "offset $"+strconv.Itoa(seq))
			listCtx.params = append(listCtx.params, pagination.PageSize)
			listCtx.params = append(listCtx.params, (pagination.PageNum-1)*pagination.PageSize)
		}
	}

	listCtx.sql = strings.Join(sqls, " ")
	return listCtx
}

func SetSubnet4UsedInfo(subnets []*resource.Subnet4, useIds bool) (err error) {
	if len(subnets) == 0 {
		return
	}

	var resp *pbdhcpagent.GetSubnetsLeasesCountResponse
	if useIds {
		var ids []uint64
		for _, subnet := range subnets {
			if subnet.Capacity != 0 && len(subnet.Nodes) != 0 {
				ids = append(ids, subnet.SubnetId)
			}
		}

		if len(ids) != 0 {
			err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
				resp, err = client.GetSubnets4LeasesCountWithIds(
					ctx, &pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: ids})
				return err
			})
		} else {
			return
		}
	} else {
		err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
			resp, err = client.GetSubnets4LeasesCount(
				ctx, &pbdhcpagent.GetSubnetsLeasesCountRequest{})
			return err
		})
	}

	if err != nil {
		return
	}

	subnetsLeasesCount := resp.GetSubnetsLeasesCount()
	for _, subnet := range subnets {
		setSubnet4LeasesUsedRatio(subnet, subnetsLeasesCount[subnet.SubnetId])
	}

	return
}

func setSubnet4LeasesUsedRatio(subnet *resource.Subnet4, leasesCount uint64) {
	if leasesCount != 0 && subnet.Capacity != 0 {
		subnet.UsedCount = leasesCount
		subnet.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(subnet.Capacity))
	}
}

func setPagination(ctx *restresource.Context, hasPagination bool, pageTotal int) {
	if hasPagination && pageTotal != 0 {
		pagination := ctx.GetPagination()
		pagination.Total = pageTotal
		pagination.PageTotal = int(math.Ceil(float64(pageTotal) /
			float64(pagination.PageSize)))
		ctx.SetPagination(pagination)
	}
}

func setSubnet4sNodeNames(subnets []*resource.Subnet4, nodeNames map[string]Agent) {
	for _, subnet := range subnets {
		subnet.NodeNames, subnet.NodeIds = getSubnetNodeNamesAndIds(subnet.Nodes, nodeNames)
	}
}

func getSubnetNodeNamesAndIds(nodes []string, nodeNames map[string]Agent) ([]string, []string) {
	names := make([]string, 0, len(nodes))
	ids := make([]string, 0, len(nodes))
	uniqueIds := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		for _, agent := range nodeNames {
			if agent.HasNode(node) {
				if !uniqueIds[agent.Id] {
					ids = append(ids, agent.Id)
					names = append(names, agent.Name)
					uniqueIds[agent.Id] = true
				}
				break
			}
		}
	}
	return names, ids
}

func (s *Subnet4Service) Get(id string) (*resource.Subnet4, error) {
	var subnets []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: id}, &subnets)
	}); err != nil {
		return nil, fmt.Errorf("get subnet4 %s from db failed: %s", id, pg.Error(err).Error())
	} else if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet4 %s", id)
	}

	setSubnet4LeasesUsedInfo(subnets[0])
	if nodeNames, err := GetAgentInfo(false, kafka.AgentRoleSentry4); err != nil {
		log.Warnf("get node names failed: %s", err.Error())
	} else {
		subnets[0].NodeNames, subnets[0].NodeIds = getSubnetNodeNamesAndIds(subnets[0].Nodes, nodeNames)
	}

	return subnets[0], nil
}

func setSubnet4LeasesUsedInfo(subnet *resource.Subnet4) {
	leasesCount, err := getSubnet4LeasesCount(subnet)
	if err != nil {
		log.Warnf("get subnet4 %s leases used ratio failed: %s", subnet.GetID(), err.Error())
	}

	setSubnet4LeasesUsedRatio(subnet, leasesCount)
}

func getSubnet4LeasesCount(subnet *resource.Subnet4) (uint64, error) {
	if subnet.Capacity == 0 || len(subnet.Nodes) == 0 {
		return 0, nil
	}

	var err error
	var resp *pbdhcpagent.GetLeasesCountResponse
	err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet4LeasesCount(ctx,
			&pbdhcpagent.GetSubnet4LeasesCountRequest{Id: subnet.SubnetId})
		return err
	})

	return resp.GetLeasesCount(), err
}

func (s *Subnet4Service) Update(subnet *resource.Subnet4) error {
	if err := subnet.ValidateParams(nil); err != nil {
		return fmt.Errorf("validate subnet4 params invalid: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			resource.SqlColumnValidLifetime:       subnet.ValidLifetime,
			resource.SqlColumnMaxValidLifetime:    subnet.MaxValidLifetime,
			resource.SqlColumnMinValidLifetime:    subnet.MinValidLifetime,
			resource.SqlColumnSubnetMask:          subnet.SubnetMask,
			resource.SqlColumnDomainServers:       subnet.DomainServers,
			resource.SqlColumnRouters:             subnet.Routers,
			resource.SqlColumnWhiteClientClasses:  subnet.WhiteClientClasses,
			resource.SqlColumnBlackClientClasses:  subnet.BlackClientClasses,
			resource.SqlColumnIfaceName:           subnet.IfaceName,
			resource.SqlColumnRelayAgentAddresses: subnet.RelayAgentAddresses,
			resource.SqlColumnNextServer:          subnet.NextServer,
			resource.SqlColumnTftpServer:          subnet.TftpServer,
			resource.SqlColumnBootfile:            subnet.Bootfile,
			resource.SqlColumnIpv6OnlyPreferred:   subnet.Ipv6OnlyPreferred,
			resource.SqlColumnTags:                subnet.Tags,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return pg.Error(err)
		}

		return sendUpdateSubnet4CmdToDHCPAgent(subnet)
	}); err != nil {
		return fmt.Errorf("update subnet4 %s failed: %s", subnet.GetID(), err.Error())
	}

	return nil
}

func setSubnet4FromDB(tx restdb.Transaction, subnet *resource.Subnet4) error {
	oldSubnet, err := getSubnet4FromDB(tx, subnet.GetID())
	if err != nil {
		return err
	}

	subnet.SubnetId = oldSubnet.SubnetId
	subnet.Capacity = oldSubnet.Capacity
	subnet.Subnet = oldSubnet.Subnet
	subnet.Ipnet = oldSubnet.Ipnet
	subnet.Nodes = oldSubnet.Nodes
	return nil
}

func getSubnet4FromDB(tx restdb.Transaction, subnetId string) (*resource.Subnet4, error) {
	var subnets []*resource.Subnet4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: subnetId},
		&subnets); err != nil {
		return nil, fmt.Errorf("get subnet4 %s from db failed: %s",
			subnetId, pg.Error(err).Error())
	} else if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet4 %s", subnetId)
	}

	return subnets[0], nil
}

func sendUpdateSubnet4CmdToDHCPAgent(subnet *resource.Subnet4) error {
	return kafka.SendDHCPCmdWithNodes(true, subnet.Nodes, kafka.UpdateSubnet4,
		&pbdhcpagent.UpdateSubnet4Request{
			Id:                  subnet.SubnetId,
			Subnet:              subnet.Subnet,
			ValidLifetime:       subnet.ValidLifetime,
			MaxValidLifetime:    subnet.MaxValidLifetime,
			MinValidLifetime:    subnet.MinValidLifetime,
			RenewTime:           subnet.ValidLifetime / 2,
			RebindTime:          subnet.ValidLifetime * 3 / 4,
			WhiteClientClasses:  subnet.WhiteClientClasses,
			BlackClientClasses:  subnet.BlackClientClasses,
			IfaceName:           subnet.IfaceName,
			RelayAgentAddresses: subnet.RelayAgentAddresses,
			NextServer:          subnet.NextServer,
			SubnetOptions:       pbSubnetOptionsFromSubnet4(subnet),
		}, nil)
}

func (s *Subnet4Service) Delete(subnet *resource.Subnet4) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkSubnet4CouldBeDelete(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableSubnet4,
			map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return pg.Error(err)
		}

		return sendDeleteSubnet4CmdToDHCPAgent(subnet, subnet.Nodes)
	}); err != nil {
		return fmt.Errorf("delete subnet4 %s failed: %s", subnet.GetID(), err.Error())
	}

	return nil
}

func checkSubnet4CouldBeDelete(tx restdb.Transaction, subnet4 *resource.Subnet4) error {
	if err := checkUsedBySharedNetwork(tx, subnet4.SubnetId); err != nil {
		return err
	}

	if leasesCount, err := getSubnet4LeasesCount(subnet4); err != nil {
		return fmt.Errorf("get subnet4 %s leases count failed: %s",
			subnet4.Subnet, err.Error())
	} else if leasesCount != 0 {
		return fmt.Errorf("can not delete subnet4 with %d ips had been allocated",
			leasesCount)
	}

	return nil
}

func sendDeleteSubnet4CmdToDHCPAgent(subnet *resource.Subnet4, nodes []string) error {
	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.DeleteSubnet4,
		&pbdhcpagent.DeleteSubnet4Request{Id: subnet.SubnetId}, nil)
}

func (s *Subnet4Service) ImportExcel(file *excel.ImportFile) (interface{}, error) {
	var oldSubnet4s []*resource.Subnet4
	if err := db.GetResources(map[string]interface{}{resource.SqlOrderBy: "subnet_id desc"},
		&oldSubnet4s); err != nil {
		return nil, fmt.Errorf("get subnet4s from db failed: %s", err.Error())
	}

	if len(oldSubnet4s) >= MaxSubnetsCount {
		return nil, fmt.Errorf("subnet4s count has reached maximum (1w)")
	}

	sentryNodes, serverNodes, sentryVip, err := kafka.GetDHCPNodes(kafka.AgentStack4)
	if err != nil {
		return nil, err
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(Subnet4ImportFileNamePrefix, TableHeaderSubnet4Fail, response)
	validSqls, reqsForSentryCreate, reqsForSentryDelete,
		reqForServerCreate, reqForServerDelete, err := parseSubnet4sFromFile(file.Name, oldSubnet4s,
		sentryNodes, sentryVip, response)
	if err != nil {
		return response, fmt.Errorf("parse subnet4s from file %s failed: %s",
			file.Name, err.Error())
	}

	if len(validSqls) == 0 {
		return response, nil
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, validSql := range validSqls {
			if _, err := tx.Exec(validSql); err != nil {
				return fmt.Errorf("batch insert subnet4s to db failed: %s",
					pg.Error(err).Error())
			}
		}

		if sentryVip != "" {
			return sendCreateSubnet4sAndPoolsCmdToDHCPAgentWithHA(sentryNodes, reqForServerCreate)
		} else {
			return sendCreateSubnet4sAndPoolsCmdToDHCPAgent(serverNodes, reqsForSentryCreate, reqsForSentryDelete,
				reqForServerCreate, reqForServerDelete)
		}
	}); err != nil {
		return response, fmt.Errorf("import subnet4s from file %s failed: %s", file.Name, err.Error())
	}

	return response, nil
}

func sendImportFieldResponse(fileName string, tableHeader []string, response *excel.ImportResult) {
	if response.Failed != 0 {
		if err := response.FlushResult(fmt.Sprintf("%s-error-%s", fileName, time.Now().Format(excel.TimeFormat)),
			tableHeader); err != nil {
			log.Warnf("write error excel file failed: %s", err.Error())
		}
	}
}

func parseSubnet4sFromFile(fileName string, oldSubnets []*resource.Subnet4, sentryNodes []string, sentryVip string, response *excel.ImportResult) ([]string, map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, map[string]*pbdhcpagent.DeleteSubnets4Request, *pbdhcpagent.CreateSubnets4AndPoolsRequest, *pbdhcpagent.DeleteSubnets4Request, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	if len(contents) < 2 {
		return nil, nil, nil, nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0],
		TableHeaderSubnet4, SubnetMandatoryFields)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	dhcpConfig, err := resource.GetDhcpConfig(true)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	clientClass4s, err := resource.GetClientClass4s()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	response.InitData(len(contents) - 1)
	var maxOldSubnetId uint64
	if len(oldSubnets) != 0 {
		maxOldSubnetId = oldSubnets[0].SubnetId
	}

	sentryNodesForCheck := sentryNodes
	if sentryVip != "" {
		sentryNodesForCheck = []string{sentryVip}
	}

	subnets := make([]*resource.Subnet4, 0, len(contents)-1)
	subnetPools := make(map[uint64][]*resource.Pool4)
	subnetReservedPools := make(map[uint64][]*resource.ReservedPool4)
	subnetReservations := make(map[uint64][]*resource.Reservation4)
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, SubnetMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addSubnetFailDataToResponse(response, TableHeaderSubnet4FailLen,
				localizationSubnet4ToStrSlice(&resource.Subnet4{}),
				fmt.Sprintf("line %d rr missing mandatory fields: %v", j+2, SubnetMandatoryFields))
			continue
		}

		subnet, pools, reservedPools, reservations, err := parseSubnet4sAndPools(
			tableHeaderFields, fields)
		if err != nil {
			addSubnetFailDataToResponse(response, TableHeaderSubnet4FailLen, localizationSubnet4ToStrSlice(subnet),
				fmt.Sprintf("line %d parse subnet4 %s fields failed: %s", j+2, subnet.Subnet, err.Error()))
		} else if err := subnet.Validate(dhcpConfig, clientClass4s); err != nil {
			addSubnetFailDataToResponse(response, TableHeaderSubnet4FailLen, localizationSubnet4ToStrSlice(subnet),
				fmt.Sprintf("line %d subnet4 %s is invalid: %s", j+2, subnet.Subnet, err.Error()))
		} else if err := checkSubnetNodesValid(subnet.Nodes, sentryNodesForCheck); err != nil {
			addSubnetFailDataToResponse(response, TableHeaderSubnet4FailLen, localizationSubnet4ToStrSlice(subnet),
				fmt.Sprintf("line %d subnet4 %s nodes is invalid: %s", j+2, subnet.Subnet, err.Error()))
		} else if err := checkSubnet4ConflictWithSubnet4s(subnet, append(oldSubnets, subnets...)); err != nil {
			addSubnetFailDataToResponse(response, TableHeaderSubnet4FailLen, localizationSubnet4ToStrSlice(subnet),
				fmt.Sprintf("line %d subnet4 %s is invalid: %s", j+2, subnet.Subnet, err.Error()))
		} else if err := checkReservation4sValid(subnet, reservations); err != nil {
			addSubnetFailDataToResponse(response, TableHeaderSubnet4FailLen, localizationSubnet4ToStrSlice(subnet),
				fmt.Sprintf("line %d subnet4 %s reservations is invalid: %s", j+2, subnet.Subnet, err.Error()))
		} else if err := checkReservedPool4sValid(subnet, reservedPools, reservations); err != nil {
			addSubnetFailDataToResponse(response, TableHeaderSubnet4FailLen, localizationSubnet4ToStrSlice(subnet),
				fmt.Sprintf("line %d subnet4 %s reserved pool4s is invalid: %s", j+2, subnet.Subnet, err.Error()))
		} else if err := checkPool4sValid(subnet, pools, reservedPools, reservations); err != nil {
			addSubnetFailDataToResponse(response, TableHeaderSubnet4FailLen, localizationSubnet4ToStrSlice(subnet),
				fmt.Sprintf("line %d subnet4 %s pool4s is invalid: %s", j+2, subnet.Subnet, err.Error()))
		} else {
			subnet.SubnetId = maxOldSubnetId + uint64(len(subnets)) + 1
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
		}
	}

	if len(subnets) == 0 {
		return nil, nil, nil, nil, nil, nil
	}

	sqls := make([]string, 0, 4)
	reqsForSentryCreate := make(map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest)
	reqForServerCreate := &pbdhcpagent.CreateSubnets4AndPoolsRequest{}
	reqsForSentryDelete := make(map[string]*pbdhcpagent.DeleteSubnets4Request)
	reqForServerDelete := &pbdhcpagent.DeleteSubnets4Request{}
	subnetAndNodes := make(map[uint64][]string)
	sqls = append(sqls, subnet4sToInsertSqlAndRequest(subnets, reqsForSentryCreate,
		reqForServerCreate, reqsForSentryDelete, reqForServerDelete, subnetAndNodes))
	if len(subnetPools) != 0 {
		sqls = append(sqls, pool4sToInsertSqlAndRequest(subnetPools,
			reqForServerCreate, reqsForSentryCreate, subnetAndNodes))
	}

	if len(subnetReservedPools) != 0 {
		sqls = append(sqls, reservedPool4sToInsertSqlAndRequest(subnetReservedPools,
			reqForServerCreate, reqsForSentryCreate, subnetAndNodes))
	}

	if len(subnetReservations) != 0 {
		sqls = append(sqls, reservation4sToInsertSqlAndRequest(subnetReservations,
			reqForServerCreate, reqsForSentryCreate, subnetAndNodes))
	}

	return sqls, reqsForSentryCreate, reqsForSentryDelete, reqForServerCreate, reqForServerDelete, nil
}

func addSubnetFailDataToResponse(response *excel.ImportResult, headerLen int, subnetSlices []string, errStr string) {
	slices := make([]string, headerLen)
	copy(slices, subnetSlices)
	slices[headerLen-1] = errStr
	response.AddFailedData(slices)
}

func parseUint32FromString(field string) (uint32, error) {
	value, err := strconv.ParseUint(field, 10, 32)
	return uint32(value), err
}

func parseSubnet4sAndPools(tableHeaderFields, fields []string) (*resource.Subnet4, []*resource.Pool4, []*resource.ReservedPool4, []*resource.Reservation4, error) {
	subnet := &resource.Subnet4{}
	var pools []*resource.Pool4
	var reservedPools []*resource.ReservedPool4
	var reservations []*resource.Reservation4
	var err error
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		switch tableHeaderFields[i] {
		case FieldNameSubnet:
			subnet.Subnet = strings.TrimSpace(field)
		case FieldNameSubnetName:
			subnet.Tags = field
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
		case FieldNameSubnetMask:
			subnet.SubnetMask = strings.TrimSpace(field)
		case FieldNameRouters:
			subnet.Routers = strings.Split(strings.TrimSpace(field), ",")
		case FieldNameDomainServers:
			subnet.DomainServers = strings.Split(strings.TrimSpace(field), ",")
		case FieldNameIfaceName:
			subnet.IfaceName = strings.TrimSpace(field)
		case FieldNameWhiteClientClasses:
			subnet.WhiteClientClasses = strings.Split(strings.TrimSpace(field), ",")
		case FieldNameBlackClientClasses:
			subnet.BlackClientClasses = strings.Split(strings.TrimSpace(field), ",")
		case FieldNameOption82:
			subnet.RelayAgentAddresses = strings.Split(strings.TrimSpace(field), ",")
		case FieldNameOption66:
			subnet.TftpServer = strings.TrimSpace(field)
		case FieldNameOption67:
			subnet.Bootfile = field
		case FieldNameOption108:
			if subnet.Ipv6OnlyPreferred, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameNodes:
			subnet.Nodes = strings.Split(strings.TrimSpace(field), ",")
		case FieldNamePools:
			if pools, err = parsePool4sFromString(strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameReservedPools:
			if reservedPools, err = parseReservedPool4sFromString(
				strings.TrimSpace(field)); err != nil {
				break
			}
		case FieldNameReservations:
			if reservations, err = parseReservation4sFromString(
				strings.TrimSpace(field)); err != nil {
				break
			}
		}
	}

	return subnet, pools, reservedPools, reservations, err
}

func parsePool4sFromString(field string) ([]*resource.Pool4, error) {
	var pools []*resource.Pool4
	for _, poolStr := range strings.Split(field, ",") {
		if poolSlices := strings.SplitN(poolStr, "-", 3); len(poolSlices) != 3 {
			return nil, fmt.Errorf("parse subnet4 pool4 %s failed with wrong regexp",
				poolStr)
		} else {
			pools = append(pools, &resource.Pool4{
				BeginAddress: poolSlices[0],
				EndAddress:   poolSlices[1],
				Comment:      poolSlices[2],
			})
		}
	}

	return pools, nil
}

func parseReservedPool4sFromString(field string) ([]*resource.ReservedPool4, error) {
	var pools []*resource.ReservedPool4
	for _, poolStr := range strings.Split(field, ",") {
		if poolSlices := strings.SplitN(poolStr, "-", 3); len(poolSlices) != 3 {
			return nil, fmt.Errorf("parse subnet4 reserved pool4 %s failed with wrong regexp",
				poolStr)
		} else {
			pools = append(pools, &resource.ReservedPool4{
				BeginAddress: poolSlices[0],
				EndAddress:   poolSlices[1],
				Comment:      poolSlices[2],
			})
		}
	}

	return pools, nil
}

func parseReservation4sFromString(field string) ([]*resource.Reservation4, error) {
	var reservations []*resource.Reservation4
	for _, reservationStr := range strings.Split(field, ",") {
		if reservationSlices := strings.SplitN(reservationStr,
			"$", 4); len(reservationSlices) != 4 {
			return nil, fmt.Errorf("parse reservation4 %s failed with wrong regexp",
				reservationStr)
		} else {
			reservation := &resource.Reservation4{
				IpAddress: reservationSlices[2],
				Comment:   reservationSlices[3],
			}

			switch reservationSlices[0] {
			case resource.ReservationIdMAC:
				reservation.HwAddress = reservationSlices[1]
			case resource.ReservationIdHostname:
				reservation.Hostname = reservationSlices[1]
			default:
				return nil, fmt.Errorf("parse reservation4 %s failed with wrong prefix %s not in [mac, hostname]",
					reservationStr, reservationSlices[0])
			}

			reservations = append(reservations, reservation)
		}
	}

	return reservations, nil
}

func checkSubnetNodesValid(subnetNodes, sentryNodes []string) error {
	for _, subnetNode := range subnetNodes {
		if slice.SliceIndex(sentryNodes, subnetNode) == -1 {
			return fmt.Errorf("subnet node %s invalid", subnetNode)
		}
	}

	return nil
}

func checkSubnet4ConflictWithSubnet4s(subnet4 *resource.Subnet4, subnets []*resource.Subnet4) error {
	for _, subnet := range subnets {
		if subnet.CheckConflictWithAnother(subnet4) {
			return fmt.Errorf("subnet4 %s conflict with subnet4 %s",
				subnet4.Subnet, subnet.Subnet)
		}
	}

	return nil
}

func checkReservation4sValid(subnet4 *resource.Subnet4, reservations []*resource.Reservation4) error {
	reservationParams := make(map[string]struct{})
	for _, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return err
		}

		if !subnet4.Ipnet.Contains(reservation.Ip) {
			return fmt.Errorf("reservation4 %s not belongs to subnet4 %s",
				reservation.IpAddress, subnet4.Subnet)
		}

		if _, ok := reservationParams[reservation.IpAddress]; ok {
			return fmt.Errorf("duplicate reservation4 with ip %s", reservation.IpAddress)
		} else {
			reservationParams[reservation.IpAddress] = struct{}{}
		}

		if reservation.HwAddress != "" {
			if _, ok := reservationParams[reservation.HwAddress]; ok {
				return fmt.Errorf("duplicate reservation4 with mac %s", reservation.HwAddress)
			} else {
				reservationParams[reservation.HwAddress] = struct{}{}
			}
		} else if reservation.Hostname != "" {
			if _, ok := reservationParams[reservation.Hostname]; ok && reservation.Hostname != "" {
				return fmt.Errorf("duplicate reservation4 with hostname %s", reservation.Hostname)
			} else {
				reservationParams[reservation.Hostname] = struct{}{}
			}
		}
	}

	subnet4.Capacity += uint64(len(reservations))
	return nil
}

func checkReservedPool4sValid(subnet4 *resource.Subnet4, reservedPools []*resource.ReservedPool4, reservations []*resource.Reservation4) error {
	reservedPoolsLen := len(reservedPools)
	for i := 0; i < reservedPoolsLen; i++ {
		if err := reservedPools[i].Validate(); err != nil {
			return err
		}

		if !checkIPsBelongsToIpnet(subnet4.Ipnet, reservedPools[i].BeginIp,
			reservedPools[i].EndIp) {
			return fmt.Errorf("reserved pool4 %s not belongs to subnet4 %s",
				reservedPools[i].String(), subnet4.Subnet)
		}

		for j := i + 1; j < reservedPoolsLen; j++ {
			if reservedPools[i].CheckConflictWithAnother(reservedPools[j]) {
				return fmt.Errorf("reserved pool4 %s conflict with another %s",
					reservedPools[i].String(), reservedPools[j].String())
			}
		}

		for _, reservation := range reservations {
			if reservedPools[i].Contains(reservation.IpAddress) {
				return fmt.Errorf("reserved pool4 %s conflict with reservation4 %s",
					reservedPools[i].String(), reservation.String())
			}
		}
	}

	return nil
}

func checkPool4sValid(subnet4 *resource.Subnet4, pools []*resource.Pool4, reservedPools []*resource.ReservedPool4, reservations []*resource.Reservation4) error {
	poolsLen := len(pools)
	for i := 0; i < poolsLen; i++ {
		if err := pools[i].Validate(); err != nil {
			return err
		}

		if !checkIPsBelongsToIpnet(subnet4.Ipnet,
			pools[i].BeginIp, pools[i].EndIp) {
			return fmt.Errorf("pool4 %s not belongs to subnet4 %s",
				pools[i].String(), subnet4.Subnet)
		}

		for j := i + 1; j < poolsLen; j++ {
			if pools[i].CheckConflictWithAnother(pools[j]) {
				return fmt.Errorf("pool4 %s conflict with another %s",
					pools[i].String(), pools[j].String())
			}
		}

		for _, reservation := range reservations {
			if pools[i].Contains(reservation.IpAddress) {
				pools[i].Capacity -= reservation.Capacity
			}
		}

		for _, reservedPool := range reservedPools {
			if pools[i].CheckConflictWithReservedPool4(reservedPool) {
				pools[i].Capacity -= getPool4ReservedCountWithReservedPool4(pools[i],
					reservedPool)
			}
		}

		subnet4.Capacity += pools[i].Capacity
	}

	return nil
}

func subnet4sToInsertSqlAndRequest(subnets []*resource.Subnet4, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, reqForServerCreate *pbdhcpagent.CreateSubnets4AndPoolsRequest, reqsForSentryDelete map[string]*pbdhcpagent.DeleteSubnets4Request, reqForServerDelete *pbdhcpagent.DeleteSubnets4Request, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_subnet4 VALUES ")
	for _, subnet := range subnets {
		buf.WriteString(subnet4ToInsertDBSqlString(subnet))
		if len(subnet.Nodes) == 0 {
			continue
		}

		subnetAndNodes[subnet.SubnetId] = subnet.Nodes
		pbSubnet := subnet4ToCreateSubnet4Request(subnet)
		reqForServerCreate.Subnets = append(reqForServerCreate.Subnets, pbSubnet)
		reqForServerDelete.Ids = append(reqForServerDelete.Ids, subnet.SubnetId)
		for _, node := range subnet.Nodes {
			createReq, ok := reqsForSentryCreate[node]
			deleteReq := reqsForSentryDelete[node]
			if !ok {
				createReq = &pbdhcpagent.CreateSubnets4AndPoolsRequest{}
				deleteReq = &pbdhcpagent.DeleteSubnets4Request{}
			}
			createReq.Subnets = append(createReq.Subnets, pbSubnet)
			deleteReq.Ids = append(deleteReq.Ids, subnet.SubnetId)
			reqsForSentryCreate[node] = createReq
			reqsForSentryDelete[node] = deleteReq
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func pool4sToInsertSqlAndRequest(subnetPools map[uint64][]*resource.Pool4, reqForServerCreate *pbdhcpagent.CreateSubnets4AndPoolsRequest, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_pool4 VALUES ")
	for subnetId, pools := range subnetPools {
		for _, pool := range pools {
			buf.WriteString(pool4ToInsertDBSqlString(subnetId, pool))
			pbPool := pool4ToCreatePool4Request(subnetId, pool)
			found := false
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqsForSentryCreate[node]; ok {
					found = true
					req.Pools = append(req.Pools, pbPool)
				}
			}

			if found {
				reqForServerCreate.Pools = append(reqForServerCreate.Pools, pbPool)
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func reservedPool4sToInsertSqlAndRequest(subnetReservedPools map[uint64][]*resource.ReservedPool4, reqForServerCreate *pbdhcpagent.CreateSubnets4AndPoolsRequest, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_reserved_pool4 VALUES ")
	for subnetId, pools := range subnetReservedPools {
		for _, pool := range pools {
			buf.WriteString(reservedPool4ToInsertDBSqlString(subnetId, pool))
			pbReservedPool := reservedPool4ToCreateReservedPool4Request(subnetId, pool)
			found := false
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqsForSentryCreate[node]; ok {
					found = true
					req.ReservedPools = append(req.ReservedPools, pbReservedPool)
				}
			}

			if found {
				reqForServerCreate.ReservedPools = append(reqForServerCreate.ReservedPools, pbReservedPool)
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func reservation4sToInsertSqlAndRequest(subnetReservations map[uint64][]*resource.Reservation4, reqForServerCreate *pbdhcpagent.CreateSubnets4AndPoolsRequest, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("INSERT INTO gr_reservation4 VALUES ")
	for subnetId, reservations := range subnetReservations {
		for _, reservation := range reservations {
			buf.WriteString(reservation4ToInsertDBSqlString(subnetId, reservation))
			pbReservation := reservation4ToCreateReservation4Request(subnetId, reservation)
			found := false
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqsForSentryCreate[node]; ok {
					found = true
					req.Reservations = append(req.Reservations, pbReservation)
				}
			}

			if found {
				reqForServerCreate.Reservations = append(reqForServerCreate.Reservations, pbReservation)
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func sendCreateSubnet4sAndPoolsCmdToDHCPAgentWithHA(sentryNodes []string, reqForServerCreate *pbdhcpagent.CreateSubnets4AndPoolsRequest) error {
	if len(sentryNodes) == 0 {
		return nil
	}

	_, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
		sentryNodes, kafka.CreateSubnet4sAndPools, reqForServerCreate)
	return err
}

func sendCreateSubnet4sAndPoolsCmdToDHCPAgent(serverNodes []string, reqsForSentryCreate map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, reqsForSentryDelete map[string]*pbdhcpagent.DeleteSubnets4Request, reqForServerCreate *pbdhcpagent.CreateSubnets4AndPoolsRequest, reqForServerDelete *pbdhcpagent.DeleteSubnets4Request) error {
	if len(reqsForSentryCreate) == 0 {
		return nil
	}

	succeedSentryNodes := make([]string, 0, len(reqsForSentryCreate))
	for node, req := range reqsForSentryCreate {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			[]string{node}, kafka.CreateSubnet4sAndPools, req); err != nil {
			deleteSentrySubnet4s(reqsForSentryDelete, succeedSentryNodes)
			return err
		}

		succeedSentryNodes = append(succeedSentryNodes, node)
	}

	succeedServerNodes := make([]string, 0, len(serverNodes))
	for _, node := range serverNodes {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			[]string{node}, kafka.CreateSubnet4sAndPools,
			reqForServerCreate); err != nil {
			deleteSentrySubnet4s(reqsForSentryDelete, succeedSentryNodes)
			deleteServerSubnet4s(reqForServerDelete, succeedServerNodes)
			return err
		}

		succeedServerNodes = append(succeedServerNodes, node)
	}

	return nil
}

func deleteSentrySubnet4s(reqs map[string]*pbdhcpagent.DeleteSubnets4Request, nodes []string) {
	for _, node := range nodes {
		if req, ok := reqs[node]; ok {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				[]string{node}, kafka.DeleteSubnet4s, req); err != nil {
				log.Errorf("delete sentry subnet4s with node %s when rollback failed: %s",
					node, err.Error())
			}
		}
	}
}

func deleteServerSubnet4s(req *pbdhcpagent.DeleteSubnets4Request, nodes []string) {
	for _, node := range nodes {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			[]string{node}, kafka.DeleteSubnet4s, req); err != nil {
			log.Errorf("delete server subnet4s with node %s when rollback failed: %s",
				node, err.Error())
		}
	}
}

func (s *Subnet4Service) ExportExcel() (interface{}, error) {
	var subnet4s []*resource.Subnet4
	var pools []*resource.Pool4
	var reservedPools []*resource.ReservedPool4
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{resource.SqlOrderBy: resource.SqlColumnSubnetId},
			&subnet4s); err != nil {
			return err
		}

		if err := tx.Fill(nil, &pools); err != nil {
			return err
		}

		if err := tx.Fill(nil, &reservedPools); err != nil {
			return err
		}

		return tx.Fill(nil, &reservations)
	}); err != nil {
		return nil, fmt.Errorf("list subnet4s and pools from db failed: %s", pg.Error(err).Error())
	}

	subnetPools := make(map[string][]string)
	for _, pool := range pools {
		poolSlices := subnetPools[pool.Subnet4]
		poolSlices = append(poolSlices, pool.String()+"-"+pool.Comment)
		subnetPools[pool.Subnet4] = poolSlices
	}

	subnetReservedPools := make(map[string][]string)
	for _, reservedPool := range reservedPools {
		reservedPoolSlices := subnetReservedPools[reservedPool.Subnet4]
		reservedPoolSlices = append(reservedPoolSlices, reservedPool.String()+"-"+reservedPool.Comment)
		subnetReservedPools[reservedPool.Subnet4] = reservedPoolSlices
	}

	subnetReservations := make(map[string][]string)
	for _, reservation := range reservations {
		reservationSlices := subnetReservations[reservation.Subnet4]
		reservationSlices = append(reservationSlices, reservation.String()+"$"+reservation.Comment)
		subnetReservations[reservation.Subnet4] = reservationSlices
	}

	strMatrix := make([][]string, 0, len(subnet4s))
	for _, subnet4 := range subnet4s {
		subnetSlices := localizationSubnet4ToStrSlice(subnet4)
		slices := make([]string, TableHeaderSubnet4Len)
		copy(slices, subnetSlices)
		if poolSlices, ok := subnetPools[subnet4.GetID()]; ok {
			slices[TableHeaderSubnet4Len-3] = strings.Join(poolSlices, ",")
		}

		if reservedPools, ok := subnetReservedPools[subnet4.GetID()]; ok {
			slices[TableHeaderSubnet4Len-2] = strings.Join(reservedPools, ",")
		}

		if reservations, ok := subnetReservations[subnet4.GetID()]; ok {
			slices[TableHeaderSubnet4Len-1] = strings.Join(reservations, ",")
		}

		strMatrix = append(strMatrix, slices)
	}

	if filepath, err := excel.WriteExcelFile(Subnet4FileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderSubnet4, strMatrix); err != nil {
		return nil, fmt.Errorf("export subnet4s failed: %s", err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (s *Subnet4Service) ExportExcelTemplate() (interface{}, error) {
	if filepath, err := excel.WriteExcelFile(Subnet4TemplateFileName,
		TableHeaderSubnet4, TemplateSubnet4); err != nil {
		return nil, fmt.Errorf("export subnet4 template failed: %s", err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (s *Subnet4Service) UpdateNodes(subnetID string, subnetNode *resource.SubnetNode) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, subnetID)
		if err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			resource.SqlColumnNodes: subnetNode.Nodes},
			map[string]interface{}{restdb.IDField: subnetID}); err != nil {
			return pg.Error(err)
		}

		return sendUpdateSubnet4NodesCmdToDHCPAgent(tx, subnet4, subnetNode.Nodes)
	}); err != nil {
		return fmt.Errorf("update subnet4 %s nodes failed: %s", subnetID, err.Error())
	}

	return nil
}

func getChangedNodes(oldNodes, newNodes []string, isv4 bool) ([]string, []string, error) {
	nodesForDelete := make(map[string]struct{})
	nodesForCreate := make(map[string]struct{})
	for _, node := range oldNodes {
		nodesForDelete[node] = struct{}{}
	}

	for _, node := range newNodes {
		if _, ok := nodesForDelete[node]; ok {
			delete(nodesForDelete, node)
		} else {
			nodesForCreate[node] = struct{}{}
		}
	}

	deleteSlices := make([]string, 0, len(nodesForDelete))
	for node := range nodesForDelete {
		deleteSlices = append(deleteSlices, node)
	}

	createSlices := make([]string, 0, len(nodesForCreate))
	for node := range nodesForCreate {
		createSlices = append(createSlices, node)
	}

	if len(deleteSlices) != 0 && len(deleteSlices) == len(oldNodes) &&
		len(createSlices) == 0 {
		if nodes, err := kafka.GetDHCPNodesWithSentryNodes(deleteSlices, isv4); err != nil {
			return nil, nil, err
		} else {
			deleteSlices = nodes
		}
	}

	if len(oldNodes) == 0 && len(createSlices) != 0 {
		if nodes, err := kafka.GetDHCPNodesWithSentryNodes(createSlices, isv4); err != nil {
			return nil, nil, err
		} else {
			createSlices = nodes
		}
	}

	return deleteSlices, createSlices, nil
}

func sendUpdateSubnet4NodesCmdToDHCPAgent(tx restdb.Transaction, subnet4 *resource.Subnet4, newNodes []string) error {
	if len(subnet4.Nodes) == 0 && len(newNodes) == 0 {
		return nil
	}

	if len(subnet4.Nodes) != 0 && len(newNodes) == 0 {
		if err := checkSubnet4CouldBeDelete(tx, subnet4); err != nil {
			return err
		}
	}

	if len(subnet4.Nodes) != 0 && len(newNodes) != 0 {
		if err := checkSubnetCouldBeUpdateNodes(true); err != nil {
			return err
		}
	}

	nodesForDelete, nodesForCreate, err := getChangedNodes(subnet4.Nodes, newNodes, true)
	if err != nil {
		return err
	}

	if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
		nodesForDelete, kafka.DeleteSubnet4,
		&pbdhcpagent.DeleteSubnet4Request{Id: subnet4.SubnetId}); err != nil {
		return err
	}

	if len(nodesForCreate) == 0 {
		return nil
	}

	req, cmd, err := genCreateSubnets4AndPoolsRequestWithSubnet4(tx, subnet4)
	if err != nil {
		return err
	}

	if succeedNodes, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
		nodesForCreate, cmd, req); err != nil {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			succeedNodes, kafka.DeleteSubnet4,
			&pbdhcpagent.DeleteSubnet4Request{Id: subnet4.SubnetId}); err != nil {
			log.Errorf("delete subnet4 %s with node %v when rollback failed: %s",
				subnet4.Subnet, succeedNodes, err.Error())
		}
		return err
	}

	return nil
}

func checkSubnetCouldBeUpdateNodes(isv4 bool) error {
	if isHA, err := IsSentryHA(isv4); err != nil {
		return err
	} else if isHA {
		return fmt.Errorf("ha model can`t update subnet nodes")
	} else {
		return nil
	}
}

func genCreateSubnets4AndPoolsRequestWithSubnet4(tx restdb.Transaction, subnet4 *resource.Subnet4) (proto.Message, kafka.DHCPCmd, error) {
	var pools []*resource.Pool4
	var reservedPools []*resource.ReservedPool4
	var reservations []*resource.Reservation4
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnet4.GetID()}, &pools); err != nil {
		return nil, "", pg.Error(err)
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnet4.GetID()},
		&reservedPools); err != nil {
		return nil, "", pg.Error(err)
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnet4.GetID()},
		&reservations); err != nil {
		return nil, "", pg.Error(err)
	}

	if len(pools) == 0 && len(reservedPools) == 0 && len(reservations) == 0 {
		return subnet4ToCreateSubnet4Request(subnet4), kafka.CreateSubnet4, nil
	}

	req := &pbdhcpagent.CreateSubnets4AndPoolsRequest{
		Subnets: []*pbdhcpagent.CreateSubnet4Request{subnet4ToCreateSubnet4Request(subnet4)},
	}
	for _, pool := range pools {
		req.Pools = append(req.Pools, pool4ToCreatePool4Request(subnet4.SubnetId, pool))
	}

	for _, pool := range reservedPools {
		req.ReservedPools = append(req.ReservedPools,
			reservedPool4ToCreateReservedPool4Request(subnet4.SubnetId, pool))
	}

	for _, reservation := range reservations {
		req.Reservations = append(req.Reservations,
			reservation4ToCreateReservation4Request(subnet4.SubnetId, reservation))
	}

	return req, kafka.CreateSubnet4sAndPools, nil
}

func (s *Subnet4Service) CouldBeCreated(couldBeCreatedSubnet *resource.CouldBeCreatedSubnet) error {
	if _, err := gohelperip.ParseCIDRv4(couldBeCreatedSubnet.Subnet); err != nil {
		return fmt.Errorf("action check subnet4 could be created input subnet %s invalid: %s",
			couldBeCreatedSubnet.Subnet, err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkSubnet4CouldBeCreated(tx, couldBeCreatedSubnet.Subnet)
	}); err != nil {
		return fmt.Errorf("action check subnet4 could be created failed: %s", err.Error())
	}

	return nil
}

func (s *Subnet4Service) ListWithSubnets(subnetListInput *resource.SubnetListInput) (interface{}, error) {
	for _, subnet := range subnetListInput.Subnets {
		if _, err := gohelperip.ParseCIDRv4(subnet); err != nil {
			return nil, resterror.NewAPIError(resterror.InvalidFormat,
				fmt.Sprintf("action list subnet4s input subnet %s invalid: %s",
					subnet, err.Error()))
		}
	}
	subnets, err := ListSubnet4sByPrefixes(subnetListInput.Subnets)
	if err != nil {
		return nil, fmt.Errorf("action list subnet4s failed: %s", err.Error())
	}

	return &resource.Subnet4ListOutput{Subnet4s: subnets}, nil
}

func ListSubnet4sByPrefixes(prefixes []string) ([]*resource.Subnet4, error) {
	var subnet4s []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnet4s, "SELECT * FROM gr_subnet4 WHERE subnet = ANY ($1)", prefixes)
	}); err != nil {
		return nil, fmt.Errorf("get subnet4 from db failed: %s", pg.Error(err).Error())
	}

	if err := SetSubnet4UsedInfo(subnet4s, true); err != nil {
		log.Warnf("set subnet4s leases used info failed: %s", err.Error())
	}
	return subnet4s, nil
}

func GetSubnet4ByIP(ip string) (*resource.Subnet4, error) {
	var subnets []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "SELECT * FROM gr_subnet4 WHERE ipnet >>= $1", ip)
	}); err != nil {
		return nil, pg.Error(err)
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("not found subnet4 with ip %s", ip)
	} else {
		return subnets[0], nil
	}
}

func GetSubnet4ByPrefix(prefix string) (*resource.Subnet4, error) {
	var subnets []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "SELECT * FROM gr_subnet4 WHERE subnet = $1", prefix)
	}); err != nil {
		return nil, pg.Error(err)
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("not found subnet4 with prefix %s", prefix)
	} else {
		return subnets[0], nil
	}
}
