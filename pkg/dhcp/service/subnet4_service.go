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
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	Subnet4FileNamePrefix       = "subnet4-"
	Subnet4TemplateFileName     = "subnet4-template"
	Subnet4ImportFileNamePrefix = "subnet4-import"

	FilterNameExcludeShared  = "exclude_shared"
	FilterNameSharedNetwork4 = "shared_network4"

	ExcludeSharedState = "subnet_id not in (select subnet_id from gr_shared_network4 where subnet_id=any(subnet_ids))"
	SharedNetworkState = "subnet_id = any((select subnet_ids from gr_shared_network4 where name = $"
)

type Subnet4Service struct {
}

func NewSubnet4Service() *Subnet4Service {
	return &Subnet4Service{}
}

func (s *Subnet4Service) Create(subnet *resource.Subnet4) error {
	if err := subnet.Validate(nil, nil); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkSubnet4CouldBeCreated(tx, subnet.Subnet); err != nil {
			return err
		}

		if err := setSubnet4ID(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Insert(subnet); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameNetwork, subnet.Subnet, err)
		}

		return sendCreateSubnet4CmdToDHCPAgent(subnet)
	})
}

func checkSubnet4CouldBeCreated(tx restdb.Transaction, subnet string) error {
	if count, err := tx.Count(resource.TableSubnet4, nil); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameCount, string(errorno.ErrNameNetworkV4),
			pg.Error(err).Error())
	} else if count >= int64(config.GetMaxSubnetsCount()) {
		return errorno.ErrExceedMaxCount(errorno.ErrNameNetworkV4,
			config.GetMaxSubnetsCount())
	}

	var subnets []*resource.Subnet4
	if err := tx.FillEx(&subnets,
		"SELECT * FROM gr_subnet4 WHERE network($1::inet) >>= network(ipnet::inet) OR network(ipnet::inet) >>= network($2::inet)",
		subnet, subnet); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, subnet, pg.Error(err).Error())
	} else if len(subnets) != 0 {
		return errorno.ErrExistIntersection(subnet, subnets[0].Subnet)
	}

	return nil
}

func setSubnet4ID(tx restdb.Transaction, subnet *resource.Subnet4) error {
	var subnets []*resource.Subnet4
	if err := tx.Fill(map[string]interface{}{
		resource.SqlOrderBy: "subnet_id desc", resource.SqlOffset: 0, resource.SqlLimit: 1},
		&subnets); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV4),
			pg.Error(err).Error())
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
				log.Errorf("create subnet4 %s failed, and rollback %v failed: %s",
					subnet.Subnet, nodesForSucceed, err.Error())
			}
		})
}

func subnet4ToCreateSubnet4Request(subnet *resource.Subnet4) *pbdhcpagent.CreateSubnet4Request {
	return &pbdhcpagent.CreateSubnet4Request{
		Id:                       subnet.SubnetId,
		Subnet:                   subnet.Subnet,
		ValidLifetime:            subnet.ValidLifetime,
		MaxValidLifetime:         subnet.MaxValidLifetime,
		MinValidLifetime:         subnet.MinValidLifetime,
		RenewTime:                subnet.ValidLifetime / 2,
		RebindTime:               subnet.ValidLifetime * 7 / 8,
		WhiteClientClassStrategy: subnet.WhiteClientClassStrategy,
		WhiteClientClasses:       subnet.WhiteClientClasses,
		BlackClientClassStrategy: subnet.BlackClientClassStrategy,
		BlackClientClasses:       subnet.BlackClientClasses,
		IfaceName:                subnet.IfaceName,
		RelayAgentCircuitId:      subnet.RelayAgentCircuitId,
		RelayAgentRemoteId:       subnet.RelayAgentRemoteId,
		RelayAgentAddresses:      subnet.RelayAgentAddresses,
		NextServer:               subnet.NextServer,
		SubnetOptions:            pbSubnetOptionsFromSubnet4(subnet),
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

	if len(subnet.CapWapACAddresses) != 0 {
		subnetOptions = append(subnetOptions, &pbdhcpagent.SubnetOption{
			Name: "cap-wap-access-controller-addresses",
			Code: 138,
			Data: strings.Join(subnet.CapWapACAddresses, ","),
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
				return errorno.ErrDBError(errorno.ErrDBNameCount,
					string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
			} else {
				subnetsCount = int(count)
			}
		}

		if err := tx.FillEx(&subnets, listCtx.sql, listCtx.params...); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery,
				string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
		}
		return nil
	}); err != nil {
		return nil, err
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
				excludeSharedState = ExcludeSharedState
			}
		case FilterNameSharedNetwork4:
			if value, ok := util.GetFilterValueWithEqModifierFromFilter(filter); ok {
				listCtx.hasShared = true
				sharedNetworkState = SharedNetworkState + strconv.Itoa(seq) + ")::numeric[])"
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
			listCtx.params = append(listCtx.params,
				(pagination.PageNum-1)*pagination.PageSize)
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

		if len(ids) == 0 {
			return
		}

		err = transport.CallDhcpAgentGrpc4(func(ctx context.Context,
			client pbdhcpagent.DHCPManagerClient) error {
			resp, err = client.GetSubnets4LeasesCountWithIds(
				ctx, &pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: ids})
			return err
		})
	} else {
		err = transport.CallDhcpAgentGrpc4(func(ctx context.Context,
			client pbdhcpagent.DHCPManagerClient) error {
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
		pagination.PageTotal = int(math.Ceil(float64(pageTotal) / float64(pagination.PageSize)))
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, id, pg.Error(err).Error())
	} else if len(subnets) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV4, id)
	}

	setSubnet4LeasesUsedInfo(subnets[0])
	if nodeNames, err := GetAgentInfo(false, kafka.AgentRoleSentry4); err != nil {
		log.Warnf("get node names failed: %s", err.Error())
	} else {
		subnets[0].NodeNames, subnets[0].NodeIds = getSubnetNodeNamesAndIds(subnets[0].Nodes,
			nodeNames)
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
	err = transport.CallDhcpAgentGrpc4(func(ctx context.Context,
		client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet4LeasesCount(ctx,
			&pbdhcpagent.GetSubnet4LeasesCountRequest{Id: subnet.SubnetId})
		return err
	})

	return resp.GetLeasesCount(), err
}

func (s *Subnet4Service) Update(subnet *resource.Subnet4) error {
	if err := subnet.ValidateParams(nil); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			resource.SqlColumnValidLifetime:            subnet.ValidLifetime,
			resource.SqlColumnMaxValidLifetime:         subnet.MaxValidLifetime,
			resource.SqlColumnMinValidLifetime:         subnet.MinValidLifetime,
			resource.SqlColumnSubnetMask:               subnet.SubnetMask,
			resource.SqlColumnDomainServers:            subnet.DomainServers,
			resource.SqlColumnRouters:                  subnet.Routers,
			resource.SqlColumnWhiteClientClassStrategy: subnet.WhiteClientClassStrategy,
			resource.SqlColumnWhiteClientClasses:       subnet.WhiteClientClasses,
			resource.SqlColumnBlackClientClassStrategy: subnet.BlackClientClassStrategy,
			resource.SqlColumnBlackClientClasses:       subnet.BlackClientClasses,
			resource.SqlColumnIfaceName:                subnet.IfaceName,
			resource.SqlColumnRelayAgentCircuitId:      subnet.RelayAgentCircuitId,
			resource.SqlColumnRelayAgentRemoteId:       subnet.RelayAgentRemoteId,
			resource.SqlColumnRelayAgentAddresses:      subnet.RelayAgentAddresses,
			resource.SqlColumnNextServer:               subnet.NextServer,
			resource.SqlColumnTftpServer:               subnet.TftpServer,
			resource.SqlColumnBootfile:                 subnet.Bootfile,
			resource.SqlColumnIpv6OnlyPreferred:        subnet.Ipv6OnlyPreferred,
			resource.SqlColumnCapWapACAddresses:        subnet.CapWapACAddresses,
			resource.SqlColumnTags:                     subnet.Tags,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, subnet.GetID(),
				pg.Error(err).Error())
		}

		return sendUpdateSubnet4CmdToDHCPAgent(subnet)
	})
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, subnetId,
			pg.Error(err).Error())
	} else if len(subnets) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV4, subnetId)
	}

	return subnets[0], nil
}

func sendUpdateSubnet4CmdToDHCPAgent(subnet *resource.Subnet4) error {
	return kafka.SendDHCPCmdWithNodes(true, subnet.Nodes, kafka.UpdateSubnet4,
		&pbdhcpagent.UpdateSubnet4Request{
			Id:                       subnet.SubnetId,
			Subnet:                   subnet.Subnet,
			ValidLifetime:            subnet.ValidLifetime,
			MaxValidLifetime:         subnet.MaxValidLifetime,
			MinValidLifetime:         subnet.MinValidLifetime,
			RenewTime:                subnet.ValidLifetime / 2,
			RebindTime:               subnet.ValidLifetime * 7 / 8,
			WhiteClientClassStrategy: subnet.WhiteClientClassStrategy,
			WhiteClientClasses:       subnet.WhiteClientClasses,
			BlackClientClassStrategy: subnet.BlackClientClassStrategy,
			BlackClientClasses:       subnet.BlackClientClasses,
			IfaceName:                subnet.IfaceName,
			RelayAgentCircuitId:      subnet.RelayAgentCircuitId,
			RelayAgentRemoteId:       subnet.RelayAgentRemoteId,
			RelayAgentAddresses:      subnet.RelayAgentAddresses,
			NextServer:               subnet.NextServer,
			SubnetOptions:            pbSubnetOptionsFromSubnet4(subnet),
		}, nil)
}

func (s *Subnet4Service) Delete(subnet *resource.Subnet4) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkSubnet4CouldBeDelete(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableSubnet4,
			map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, subnet.Subnet,
				pg.Error(err).Error())
		}

		return sendDeleteSubnet4CmdToDHCPAgent(subnet, subnet.Nodes)
	})
}

func checkSubnet4CouldBeDelete(tx restdb.Transaction, subnet4 *resource.Subnet4) error {
	if err := checkUsedBySharedNetwork(tx, subnet4); err != nil {
		return err
	}

	if leasesCount, err := getSubnet4LeasesCount(subnet4); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameCount, subnet4.Subnet,
			pg.Error(err).Error())
	} else if leasesCount != 0 {
		return errorno.ErrIPHasBeenAllocated(errorno.ErrNameNetworkV4, subnet4.Subnet)
	}

	return nil
}

func sendDeleteSubnet4CmdToDHCPAgent(subnet *resource.Subnet4, nodes []string) error {
	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.DeleteSubnet4,
		&pbdhcpagent.DeleteSubnet4Request{Id: subnet.SubnetId}, nil)
}

func (s *Subnet4Service) ImportExcel(file *excel.ImportFile) (interface{}, error) {
	if len(file.Name) == 0 {
		return nil, nil
	}

	var oldSubnet4s []*resource.Subnet4
	if err := db.GetResources(map[string]interface{}{resource.SqlOrderBy: "subnet_id desc"},
		&oldSubnet4s); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameNetworkV4), err.Error())
	}

	if len(oldSubnet4s) >= config.GetMaxSubnetsCount() {
		return nil, errorno.ErrExceedMaxCount(errorno.ErrNameNetworkV4,
			config.GetMaxSubnetsCount())
	}

	sentryNodes, serverNodes, sentryVip, err := kafka.GetDHCPNodes(kafka.AgentStack4)
	if err != nil {
		return nil, err
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(Subnet4ImportFileNamePrefix, TableHeaderSubnet4Fail,
		response)
	validSqls, reqsForSentryCreate, reqsForSentryDelete,
		reqForServerCreate, reqForServerDelete, err := parseSubnet4sFromFile(file.Name,
		oldSubnet4s, sentryNodes, sentryVip, response)
	if err != nil {
		return response, err
	}

	if len(validSqls) == 0 {
		return response, nil
	}

	if err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, validSql := range validSqls {
			if _, err = tx.Exec(validSql); err != nil {
				return errorno.ErrDBError(errorno.ErrDBNameInsert,
					string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
			}
		}

		if reqForServerCreate == nil || len(reqForServerCreate.Subnets) == 0 {
			return nil
		}

		if sentryVip != "" {
			return sendCreateSubnet4sAndPoolsCmdToDHCPAgentWithHA(sentryNodes,
				reqForServerCreate)
		} else {
			return sendCreateSubnet4sAndPoolsCmdToDHCPAgent(serverNodes, reqsForSentryCreate,
				reqsForSentryDelete, reqForServerCreate, reqForServerDelete)
		}
	}); err != nil {
		return response, err
	}

	return response, nil
}

func sendImportFieldResponse(fileName string, tableHeader []string, response *excel.ImportResult) {
	if response.Failed != 0 {
		if err := response.FlushResult(fmt.Sprintf("%s-error-%s", fileName,
			time.Now().Format(excel.TimeFormat)), tableHeader); err != nil {
			log.Warnf("write error excel file failed: %s", err.Error())
		}
	}
}

func parseSubnet4sFromFile(fileName string, oldSubnets []*resource.Subnet4, sentryNodes []string, sentryVip string, response *excel.ImportResult) ([]string, map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, map[string]*pbdhcpagent.DeleteSubnets4Request, *pbdhcpagent.CreateSubnets4AndPoolsRequest, *pbdhcpagent.DeleteSubnets4Request, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return nil, nil, nil, nil, nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return nil, nil, nil, nil, nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0],
		TableHeaderSubnet4, SubnetMandatoryFields)
	if err != nil {
		return nil, nil, nil, nil, nil, errorno.ErrInvalidTableHeader()
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

	subnets := make([]*resource.Subnet4, 0, len(contents))
	subnetPools := make(map[uint64][]*resource.Pool4, len(contents))
	subnetReservedPools := make(map[uint64][]*resource.ReservedPool4, len(contents))
	subnetReservations := make(map[uint64][]*resource.Reservation4, len(contents))
	fieldcontents := contents[1:]
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, SubnetMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderSubnet4FailLen,
				localizationSubnet4ToStrSlice(&resource.Subnet4{}),
				errorno.ErrMissingMandatory(j+2, SubnetMandatoryFields).ErrorCN())
			continue
		}

		subnet, pools, reservedPools, reservations, err := parseSubnet4sAndPools(
			tableHeaderFields, fields)
		if err != nil {
			addFailDataToResponse(response, TableHeaderSubnet4FailLen,
				localizationSubnet4ToStrSlice(subnet), errorno.TryGetErrorCNMsg(err))
		} else if err := subnet.Validate(dhcpConfig, clientClass4s); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet4FailLen,
				localizationSubnet4ToStrSlice(subnet), errorno.TryGetErrorCNMsg(err))
		} else if err := checkSubnetNodesValid(subnet.Nodes,
			sentryNodesForCheck); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet4FailLen,
				localizationSubnet4ToStrSlice(subnet), errorno.TryGetErrorCNMsg(err))
		} else if err := checkSubnet4ConflictWithSubnet4s(subnet,
			append(oldSubnets, subnets...)); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet4FailLen,
				localizationSubnet4ToStrSlice(subnet), errorno.TryGetErrorCNMsg(err))
		} else if err := checkReservation4sValid(subnet, reservations); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet4FailLen,
				localizationSubnet4ToStrSlice(subnet), errorno.TryGetErrorCNMsg(err))
		} else if err := checkReservedPool4sValid(subnet, reservedPools,
			reservations); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet4FailLen,
				localizationSubnet4ToStrSlice(subnet), errorno.TryGetErrorCNMsg(err))
		} else if err := checkPool4sValid(subnet, pools, reservedPools,
			reservations); err != nil {
			addFailDataToResponse(response, TableHeaderSubnet4FailLen,
				localizationSubnet4ToStrSlice(subnet), errorno.TryGetErrorCNMsg(err))
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
	subnetAndNodes := make(map[uint64][]string, len(subnets))
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

	return sqls, reqsForSentryCreate, reqsForSentryDelete, reqForServerCreate,
		reqForServerDelete, nil
}

func addFailDataToResponse(response *excel.ImportResult, headerLen int, resourceSlices []string, errStr string) {
	if response != nil {
		errSlices := make([]string, headerLen)
		copy(errSlices, resourceSlices)
		errSlices[headerLen-1] = errStr
		response.AddFailedData(errSlices)
	}
}

func parseUint32FromString(field string) (uint32, error) {
	value, err := strconv.ParseUint(field, 10, 32)
	if err != nil {
		return 0, errorno.ErrInvalidParams(errorno.ErrNameNumber, field)
	}
	return uint32(value), nil
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
			subnet.Tags = strings.TrimSpace(field)
		case FieldNameValidLifetime:
			if subnet.ValidLifetime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations,
					errorno.ErrInvalidParams(errorno.ErrNameLifetime, field)
			}
		case FieldNameMaxValidLifetime:
			if subnet.MaxValidLifetime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations,
					errorno.ErrInvalidParams(errorno.ErrNameMaxLifetime, field)
			}
		case FieldNameMinValidLifetime:
			if subnet.MinValidLifetime, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations,
					errorno.ErrInvalidParams(errorno.ErrNameMinLifetime, field)
			}
		case FieldNameSubnetMask:
			subnet.SubnetMask = strings.TrimSpace(field)
			if !util.IsSubnetMask(subnet.SubnetMask) {
				return subnet, pools, reservedPools, reservations,
					errorno.ErrInvalidParams(errorno.ErrNameNetworkMask, field)
			}
		case FieldNameRouters:
			subnet.Routers = splitFieldWithoutSpace(field)
		case FieldNameDomainServers:
			subnet.DomainServers = splitFieldWithoutSpace(field)
		case FieldNameIfaceName:
			subnet.IfaceName = strings.TrimSpace(field)
		case FieldNameWhiteClientClassStrategy:
			subnet.WhiteClientClassStrategy = internationalizationClientClassStrategy(
				strings.TrimSpace(field))
		case FieldNameWhiteClientClasses:
			subnet.WhiteClientClasses = splitFieldWithoutSpace(field)
		case FieldNameBlackClientClassStrategy:
			subnet.BlackClientClassStrategy = internationalizationClientClassStrategy(
				strings.TrimSpace(field))
		case FieldNameBlackClientClasses:
			subnet.BlackClientClasses = splitFieldWithoutSpace(field)
		case FieldNameRelayCircuitId:
			subnet.RelayAgentCircuitId = strings.TrimSpace(field)
		case FieldNameRelayRemoteId:
			subnet.RelayAgentRemoteId = strings.TrimSpace(field)
		case FieldNameRelayAddresses:
			subnet.RelayAgentAddresses = splitFieldWithoutSpace(field)
		case FieldNameOption66:
			subnet.TftpServer = strings.TrimSpace(field)
		case FieldNameOption67:
			subnet.Bootfile = strings.TrimSpace(field)
		case FieldNameOption108:
			if subnet.Ipv6OnlyPreferred, err = parseUint32FromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations,
					errorno.ErrInvalidParams(FieldNameOption108, field)
			}
		case FieldNameOption138:
			subnet.CapWapACAddresses = splitFieldWithoutSpace(field)
		case FieldNameNodes:
			subnet.Nodes = splitFieldWithoutSpace(field)
		case FieldNameNextServer:
			subnet.NextServer = strings.TrimSpace(field)
		case FieldNamePools:
			if pools, err = parsePool4sFromString(strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, err
			}
		case FieldNameReservedPools:
			if reservedPools, err = parseReservedPool4sFromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, err
			}
		case FieldNameReservations:
			if reservations, err = parseReservation4sFromString(
				strings.TrimSpace(field)); err != nil {
				return subnet, pools, reservedPools, reservations, err
			}
		}
	}

	return subnet, pools, reservedPools, reservations, nil
}

func parsePool4sFromString(field string) ([]*resource.Pool4, error) {
	var pools []*resource.Pool4
	for _, poolStr := range strings.Split(field, resource.CommonDelimiter) {
		poolStr = strings.TrimSpace(poolStr)
		if poolSlices := strings.SplitN(poolStr, resource.PoolDelimiter,
			3); len(poolSlices) != 3 {
			return nil, errorno.ErrInvalidParams(errorno.ErrNameDhcpPool, poolStr)
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
	for _, poolStr := range strings.Split(field, resource.CommonDelimiter) {
		poolStr = strings.TrimSpace(poolStr)
		if poolSlices := strings.SplitN(poolStr, resource.PoolDelimiter,
			3); len(poolSlices) != 3 {
			return nil, errorno.ErrInvalidParams(errorno.ErrNameDhcpReservedPool, poolStr)
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
	for _, reservationStr := range strings.Split(field, resource.CommonDelimiter) {
		reservationStr = strings.TrimSpace(reservationStr)
		if reservationSlices := strings.SplitN(reservationStr,
			resource.ReservationDelimiter, 4); len(reservationSlices) != 4 {
			return nil, errorno.ErrInvalidParams(errorno.ErrNameDhcpReservation,
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
				return nil, errorno.ErrOnlySupport(errorno.ErrName(reservationSlices[0]),
					[]string{resource.ReservationIdMAC, resource.ReservationIdHostname})
			}

			reservations = append(reservations, reservation)
		}
	}

	return reservations, nil
}

func checkSubnetNodesValid(subnetNodes, sentryNodes []string) error {
	for _, subnetNode := range subnetNodes {
		if slice.SliceIndex(sentryNodes, subnetNode) == -1 {
			return errorno.ErrInvalidParams(errorno.ErrNameDhcpSentryNode, subnetNode)
		}
	}

	return nil
}

func checkSubnet4ConflictWithSubnet4s(subnet4 *resource.Subnet4, subnets []*resource.Subnet4) error {
	for _, subnet := range subnets {
		if subnet.CheckConflictWithAnother(subnet4) {
			return errorno.ErrConflict(errorno.ErrNameNetworkV4, errorno.ErrNameNetworkV4,
				subnet4.Subnet, subnet.Subnet)
		}
	}

	return nil
}

func checkReservation4sValid(subnet4 *resource.Subnet4, reservations []*resource.Reservation4) error {
	reservation4Identifier := Reservation4IdentifierFromReservations(nil)
	for _, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return err
		}

		if !subnet4.Ipnet.Contains(reservation.Ip) {
			return errorno.ErrNotBelongTo(errorno.ErrNameIp, errorno.ErrNameNetworkV4,
				reservation.IpAddress, subnet4.Subnet)
		}

		if err := reservation4Identifier.Add(reservation); err != nil {
			return err
		}
	}

	subnet4.Capacity += uint64(len(reservations))
	return nil
}

func checkReservedPool4sValid(subnet4 *resource.Subnet4, reservedPools []*resource.ReservedPool4, reservations []*resource.Reservation4) error {
	for i, reservedPool := range reservedPools {
		if err := reservedPool.Validate(); err != nil {
			return err
		}

		if !checkIPsBelongsToIpnet(subnet4.Ipnet, reservedPool.BeginIp, reservedPool.EndIp) {
			return errorno.ErrNotBelongTo(errorno.ErrNameDhcpReservedPool,
				errorno.ErrNameNetworkV4, reservedPool.String(), subnet4.Subnet)
		}

		if err := checkReservedPool4ConflictWithReservedPool4s(reservedPool,
			reservedPools[i+1:]); err != nil {
			return err
		}

		if err := checkReservedPool4ConflictWithReservation4s(reservedPool,
			reservations); err != nil {
			return err
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
			return errorno.ErrNotBelongTo(errorno.ErrNameDhcpPool, errorno.ErrNameNetworkV4,
				pools[i].String(), subnet4.Subnet)
		}

		for j := i + 1; j < poolsLen; j++ {
			if pools[i].CheckConflictWithAnother(pools[j]) {
				return errorno.ErrConflict(errorno.ErrNameDhcpPool, errorno.ErrNameDhcpPool,
					pools[i].String(), pools[j].String())
			}
		}

		for _, reservation := range reservations {
			if pools[i].ContainsIpstr(reservation.IpAddress) {
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
				reqForServerCreate.ReservedPools = append(reqForServerCreate.ReservedPools,
					pbReservedPool)
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
				reqForServerCreate.Reservations = append(reqForServerCreate.Reservations,
					pbReservation)
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
		if err := tx.Fill(map[string]interface{}{
			resource.SqlOrderBy: resource.SqlColumnSubnetId}, &subnet4s); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery,
				string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
		}

		if err := tx.Fill(nil, &pools); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery,
				string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
		}

		if err := tx.Fill(nil, &reservedPools); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery,
				string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
		}

		if err := tx.Fill(nil, &reservations); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery,
				string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}

		return nil
	}); err != nil {
		return nil, err
	}

	subnetPools := make(map[string][]string, len(subnet4s))
	for _, pool := range pools {
		poolSlices := subnetPools[pool.Subnet4]
		poolSlices = append(poolSlices, pool.String()+resource.PoolDelimiter+pool.Comment)
		subnetPools[pool.Subnet4] = poolSlices
	}

	subnetReservedPools := make(map[string][]string, len(subnet4s))
	for _, reservedPool := range reservedPools {
		reservedPoolSlices := subnetReservedPools[reservedPool.Subnet4]
		reservedPoolSlices = append(reservedPoolSlices,
			reservedPool.String()+resource.PoolDelimiter+reservedPool.Comment)
		subnetReservedPools[reservedPool.Subnet4] = reservedPoolSlices
	}

	subnetReservations := make(map[string][]string, len(subnet4s))
	for _, reservation := range reservations {
		reservationSlices := subnetReservations[reservation.Subnet4]
		reservationSlices = append(reservationSlices,
			reservation.String()+resource.ReservationDelimiter+reservation.Comment)
		subnetReservations[reservation.Subnet4] = reservationSlices
	}

	strMatrix := make([][]string, 0, len(subnet4s))
	for _, subnet4 := range subnet4s {
		subnetSlices := localizationSubnet4ToStrSlice(subnet4)
		slices := make([]string, TableHeaderSubnet4Len)
		copy(slices, subnetSlices)
		if poolSlices, ok := subnetPools[subnet4.GetID()]; ok {
			slices[TableHeaderSubnet4Len-3] = strings.Join(poolSlices,
				resource.CommonDelimiter)
		}

		if reservedPools, ok := subnetReservedPools[subnet4.GetID()]; ok {
			slices[TableHeaderSubnet4Len-2] = strings.Join(reservedPools,
				resource.CommonDelimiter)
		}

		if reservations, ok := subnetReservations[subnet4.GetID()]; ok {
			slices[TableHeaderSubnet4Len-1] = strings.Join(reservations,
				resource.CommonDelimiter)
		}

		strMatrix = append(strMatrix, slices)
	}

	if filepath, err := excel.WriteExcelFile(Subnet4FileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderSubnet4, strMatrix); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameNetworkV4), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (s *Subnet4Service) ExportExcelTemplate() (interface{}, error) {
	if filepath, err := excel.WriteExcelFile(Subnet4TemplateFileName,
		TableHeaderSubnet4, TemplateSubnet4); err != nil {
		return nil, errorno.ErrOperateResource(errorno.ErrNameExport,
			string(errorno.ErrNameTemplate), err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (s *Subnet4Service) UpdateNodes(subnetID string, subnetNode *resource.SubnetNode) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, subnetID)
		if err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			resource.SqlColumnNodes: subnetNode.Nodes},
			map[string]interface{}{restdb.IDField: subnetID}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, subnetID,
				pg.Error(err).Error())
		}

		return sendUpdateSubnet4NodesCmdToDHCPAgent(tx, subnet4, subnetNode.Nodes)
	})
}

func getChangedNodes(oldNodes, newNodes []string, isv4 bool) ([]string, []string, error) {
	nodesForDelete := make(map[string]struct{}, len(oldNodes))
	nodesForCreate := make(map[string]struct{}, len(newNodes))
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
	if checkSlicesEqual(subnet4.Nodes, newNodes) {
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

func checkSlicesEqual(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}

	return true
}

func checkSubnetCouldBeUpdateNodes(isv4 bool) error {
	if isHA, err := IsSentryHA(isv4); err != nil {
		return err
	} else if isHA {
		return errorno.ErrHaMode()
	} else {
		return nil
	}
}

func genCreateSubnets4AndPoolsRequestWithSubnet4(tx restdb.Transaction, subnet4 *resource.Subnet4) (proto.Message, kafka.DHCPCmd, error) {
	var pools []*resource.Pool4
	var reservedPools []*resource.ReservedPool4
	var reservations []*resource.Reservation4
	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnSubnet4: subnet4.GetID()}, &pools); err != nil {
		return nil, "", errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnet4.GetID()},
		&reservedPools); err != nil {
		return nil, "", errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnet4.GetID()},
		&reservations); err != nil {
		return nil, "", errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
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
		return errorno.ErrParseCIDR(couldBeCreatedSubnet.Subnet)
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkSubnet4CouldBeCreated(tx, couldBeCreatedSubnet.Subnet)
	})
}

func (s *Subnet4Service) ListWithSubnets(subnetListInput *resource.SubnetListInput) (interface{}, error) {
	for _, subnet := range subnetListInput.Subnets {
		if _, err := gohelperip.ParseCIDRv4(subnet); err != nil {
			return nil, errorno.ErrParseCIDR(subnet)
		}
	}
	subnets, err := ListSubnet4sByPrefixes(subnetListInput.Subnets)
	if err != nil {
		return nil, err
	}

	return &resource.Subnet4ListOutput{Subnet4s: subnets}, nil
}

func ListSubnet4sByPrefixes(prefixes []string) ([]*resource.Subnet4, error) {
	var subnet4s []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnet4s,
			"SELECT * FROM gr_subnet4 WHERE subnet = ANY ($1::TEXT[])", prefixes)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
	}

	if len(subnets) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV4, ip)
	} else {
		return subnets[0], nil
	}
}

func GetSubnet4ByPrefix(prefix string) (subnet *resource.Subnet4, err error) {
	restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet, err = getSubnet4WithPrefix(tx, prefix)
		return err
	})
	return
}

func getSubnet4WithPrefix(tx restdb.Transaction, prefix string) (*resource.Subnet4, error) {
	var subnets []*resource.Subnet4
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet: prefix},
		&subnets); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
	} else if len(subnets) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV4, prefix)
	} else {
		return subnets[0], nil
	}
}
