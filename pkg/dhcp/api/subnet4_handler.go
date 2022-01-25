package api

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
	csvutil "github.com/linkingthing/clxone-utils/csv"
	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	MaxSubnetsCount = 10000

	Subnet4FileNamePrefix   = "subnet4-"
	Subnet4TemplateFileName = "subnet4-template"

	FilterNameExcludeShared  = "exclude_shared"
	FilterNameSharedNetwork4 = "shared_network4"
)

type Subnet4Handler struct {
}

func NewSubnet4Handler() *Subnet4Handler {
	return &Subnet4Handler{}
}

func (s *Subnet4Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := subnet.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create subnet params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkSubnet4CouldBeCreated(tx, subnet.Subnet); err != nil {
			return err
		}

		if err := setSubnet4ID(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Insert(subnet); err != nil {
			return err
		}

		return sendCreateSubnet4CmdToDHCPAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create subnet %s failed: %s", subnet.Subnet, err.Error()))
	}

	return subnet, nil
}

func checkSubnet4CouldBeCreated(tx restdb.Transaction, subnet string) error {
	if count, err := tx.Count(resource.TableSubnet4, nil); err != nil {
		return fmt.Errorf("get subnet4s count failed: %s", err.Error())
	} else if count >= MaxSubnetsCount {
		return fmt.Errorf("subnet4s count has reached maximum (1w)")
	}

	var subnets []*resource.Subnet4
	if err := tx.FillEx(&subnets,
		"select * from gr_subnet4 where $1 && ipnet", subnet); err != nil {
		return fmt.Errorf("check subnet4 conflict failed: %s", err.Error())
	} else if len(subnets) != 0 {
		return fmt.Errorf("conflict with subnet4 %s", subnets[0].Subnet)
	}

	return nil
}

func setSubnet4ID(tx restdb.Transaction, subnet *resource.Subnet4) error {
	var subnets []*resource.Subnet4
	if err := tx.Fill(map[string]interface{}{
		"orderby": "subnet_id desc", "offset": 0, "limit": 1},
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

func sendCreateSubnet4CmdToDHCPAgent(subnet *resource.Subnet4) error {
	nodesForSucceed, err := sendDHCPCmdWithNodes(true, subnet.Nodes, dhcpservice.CreateSubnet4,
		subnet4ToCreateSubnet4Request(subnet))
	if err != nil {
		if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, dhcpservice.DeleteSubnet4,
			&pbdhcpagent.DeleteSubnet4Request{Id: subnet.SubnetId}); err != nil {
			log.Errorf("create subnet4 %s failed, and rollback it failed: %s",
				subnet.Subnet, err.Error())
		}
	}

	return err
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
		ClientClass:         subnet.ClientClass,
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

	return subnetOptions
}

func (s *Subnet4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
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
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnet4s from db failed: %s", err.Error()))
	}

	if err := setSubnet4sLeasesUsedInfo(subnets, listCtx); err != nil {
		log.Warnf("set subnet4s leases used info failed: %s", err.Error())
	}

	if nodeNames, err := GetNodeNames(true); err != nil {
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
	return l.hasExclude == false && l.hasShared == false
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

	sqls := []string{"select * from gr_" + string(table)}
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
	if listCtx.hasFilterSubnet == false {
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

func setSubnet4sLeasesUsedInfo(subnets []*resource.Subnet4, ctx listSubnetContext) error {
	if ctx.needSetSubnetsLeasesUsedInfo() == false || len(subnets) == 0 {
		return nil
	}

	var resp *pbdhcpagent.GetSubnetsLeasesCountResponse
	var err error
	if ctx.isUseIds() {
		var ids []uint64
		for _, subnet := range subnets {
			if subnet.Capacity != 0 {
				ids = append(ids, subnet.SubnetId)
			}
		}

		if len(ids) != 0 {
			resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets4LeasesCountWithIds(
				context.TODO(), &pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: ids})
		}
	} else {
		resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets4LeasesCount(
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

func setPagination(ctx *restresource.Context, hasPagination bool, pageTotal int) {
	if hasPagination && pageTotal != 0 {
		pagination := ctx.GetPagination()
		pagination.Total = pageTotal
		pagination.PageTotal = int(math.Ceil(float64(pageTotal) /
			float64(pagination.PageSize)))
		ctx.SetPagination(pagination)
	}
}

func setSubnet4sNodeNames(subnets []*resource.Subnet4, nodeNames map[string]string) {
	for _, subnet := range subnets {
		subnet.NodeNames = getSubnetNodeNames(subnet.Nodes, nodeNames)
	}
}

func getSubnetNodeNames(nodes []string, nodeNames map[string]string) []string {
	var names []string
	for _, node := range nodes {
		if name, ok := nodeNames[node]; ok {
			names = append(names, name)
		}
	}
	return names
}

func (s *Subnet4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	var subnets []*resource.Subnet4
	subnetInterface, err := restdb.GetResourceWithID(db.GetDB(), subnetID, &subnets)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet %s from db failed: %s", subnetID, err.Error()))
	}

	subnet := subnetInterface.(*resource.Subnet4)
	if err := setSubnet4LeasesUsedRatio(subnet); err != nil {
		log.Warnf("get subnet %s leases used ratio failed: %s", subnetID, err.Error())
	}

	if nodeNames, err := GetNodeNames(true); err != nil {
		log.Warnf("get node names failed: %s", err.Error())
	} else {
		subnet.NodeNames = getSubnetNodeNames(subnet.Nodes, nodeNames)
	}

	return subnets[0], nil
}

func setSubnet4LeasesUsedRatio(subnet *resource.Subnet4) error {
	leasesCount, err := getSubnet4LeasesCount(subnet)
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

func getSubnet4LeasesCount(subnet *resource.Subnet4) (uint64, error) {
	if subnet.Capacity == 0 {
		return 0, nil
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4LeasesCount(context.TODO(),
		&pbdhcpagent.GetSubnet4LeasesCountRequest{Id: subnet.SubnetId})
	return resp.GetLeasesCount(), err
}

func (s *Subnet4Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := subnet.ValidateParams(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update subnet params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			"valid_lifetime":        subnet.ValidLifetime,
			"max_valid_lifetime":    subnet.MaxValidLifetime,
			"min_valid_lifetime":    subnet.MinValidLifetime,
			"subnet_mask":           subnet.SubnetMask,
			"domain_servers":        subnet.DomainServers,
			"routers":               subnet.Routers,
			"client_class":          subnet.ClientClass,
			"iface_name":            subnet.IfaceName,
			"relay_agent_addresses": subnet.RelayAgentAddresses,
			"next_server":           subnet.NextServer,
			"tftp_server":           subnet.TftpServer,
			"bootfile":              subnet.Bootfile,
			"tags":                  subnet.Tags,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendUpdateSubnet4CmdToDHCPAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return subnet, nil
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
		return nil, fmt.Errorf("get subnet %s from db failed: %s",
			subnetId, err.Error())
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet %s", subnetId)
	}

	return subnets[0], nil
}

func sendUpdateSubnet4CmdToDHCPAgent(subnet *resource.Subnet4) error {
	_, err := sendDHCPCmdWithNodes(true, subnet.Nodes, dhcpservice.UpdateSubnet4,
		&pbdhcpagent.UpdateSubnet4Request{
			Id:                  subnet.SubnetId,
			Subnet:              subnet.Subnet,
			ValidLifetime:       subnet.ValidLifetime,
			MaxValidLifetime:    subnet.MaxValidLifetime,
			MinValidLifetime:    subnet.MinValidLifetime,
			RenewTime:           subnet.ValidLifetime / 2,
			RebindTime:          subnet.ValidLifetime * 3 / 4,
			ClientClass:         subnet.ClientClass,
			IfaceName:           subnet.IfaceName,
			RelayAgentAddresses: subnet.RelayAgentAddresses,
			NextServer:          subnet.NextServer,
			SubnetOptions:       pbSubnetOptionsFromSubnet4(subnet),
		})
	return err
}

func (s *Subnet4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkSubnet4CouldBeDelete(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableSubnet4,
			map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendDeleteSubnet4CmdToDHCPAgent(subnet, subnet.Nodes)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return nil
}

func checkSubnet4CouldBeDelete(tx restdb.Transaction, subnet4 *resource.Subnet4) error {
	if err := checkUsedBySharedNetwork(tx, subnet4.SubnetId); err != nil {
		return err
	}

	if leasesCount, err := getSubnet4LeasesCount(subnet4); err != nil {
		return fmt.Errorf("get subnet %s leases count failed: %s",
			subnet4.Subnet, err.Error())
	} else if leasesCount != 0 {
		return fmt.Errorf("can not delete subnet with %d ips had been allocated",
			leasesCount)
	}

	return nil
}

func sendDeleteSubnet4CmdToDHCPAgent(subnet *resource.Subnet4, nodes []string) error {
	_, err := sendDHCPCmdWithNodes(true, nodes, dhcpservice.DeleteSubnet4,
		&pbdhcpagent.DeleteSubnet4Request{Id: subnet.SubnetId})
	return err
}

func (h *Subnet4Handler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case csvutil.ActionNameImportCSV:
		return h.importCSV(ctx)
	case csvutil.ActionNameExportCSV:
		return h.exportCSV(ctx)
	case csvutil.ActionNameExportCSVTemplate:
		return h.exportCSVTemplate(ctx)
	case resource.ActionNameUpdateNodes:
		return h.updateNodes(ctx)
	case resource.ActionNameCouldBeCreated:
		return h.couldBeCreated(ctx)
	case resource.ActionNameListWithSubnets:
		return h.listWithSubnets(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *Subnet4Handler) importCSV(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var oldSubnet4s []*resource.Subnet4
	if err := db.GetResources(map[string]interface{}{"orderby": "subnet_id desc"},
		&oldSubnet4s); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet4s from db failed: %s", err.Error()))
	}

	if len(oldSubnet4s) >= MaxSubnetsCount {
		return nil, resterror.NewAPIError(resterror.ServerError,
			"subnet4s count has reached maximum (1w)")
	}

	file, ok := ctx.Resource.GetAction().Input.(*csvutil.ImportFile)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action importcsv input invalid"))
	}

	validSqls, createSubnets4AndPoolsReq, deleteSubnets4Req, err := parseSubnet4sFromFile(
		file.Name, oldSubnet4s)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("parse subnet4s from file %s failed: %s",
				file.Name, err.Error()))
	}

	if len(validSqls) == 0 {
		return nil, nil
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, validSql := range validSqls {
			if _, err := tx.Exec(validSql); err != nil {
				return fmt.Errorf("batch insert subnet4s to db failed: %s",
					err.Error())
			}
		}

		return sendCreateSubnet4sAndPoolsCmdToDHCPAgent(createSubnets4AndPoolsReq,
			deleteSubnets4Req)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("import subnet4s from file %s failed: %s",
				file.Name, err.Error()))
	}

	return nil, nil
}

func parseSubnet4sFromFile(fileName string, oldSubnets []*resource.Subnet4) ([]string, map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, map[string]*pbdhcpagent.DeleteSubnets4Request, error) {
	contents, err := csvutil.ReadCSVFile(fileName)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(contents) < 2 {
		return nil, nil, nil, nil
	}

	tableHeaderFields, err := csvutil.ParseTableHeader(contents[0],
		TableHeaderSubnet4, SubnetMandatoryFields)
	if err != nil {
		return nil, nil, nil, err
	}

	oldSubnetsLen := len(oldSubnets)
	subnets := make([]*resource.Subnet4, 0)
	subnetPools := make(map[uint64][]*resource.Pool4)
	subnetReservedPools := make(map[uint64][]*resource.ReservedPool4)
	subnetReservations := make(map[uint64][]*resource.Reservation4)
	fieldcontents := contents[1:]
	for _, fieldcontent := range fieldcontents {
		fields, missingMandatory, emptyLine := csvutil.ParseTableFields(fieldcontent,
			tableHeaderFields, SubnetMandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			log.Warnf("subnet4 missing mandatory fields subnet")
			continue
		}

		subnet, pools, reservedPools, reservations, err := parseSubnet4sAndPools(
			tableHeaderFields, fields)
		if err != nil {
			log.Warnf("parse subnet4 %s fields failed: %s", subnet.Subnet, err.Error())
		} else if err := subnet.Validate(); err != nil {
			log.Warnf("subnet %s is invalid: %s", subnet.Subnet, err.Error())
		} else if err := checkSubnet4ConflictWithSubnet4s(subnet,
			append(oldSubnets, subnets...)); err != nil {
			log.Warnf(err.Error())
		} else if err := checkReservationsValid(subnet, reservations); err != nil {
			log.Warnf("subnet %s reservations is invalid: %s", subnet.Subnet, err.Error())
		} else if err := checkReservedPool4sValid(subnet, reservedPools,
			reservations); err != nil {
			log.Warnf("subnet %s reserved pool4s is invalid: %s",
				subnet.Subnet, err.Error())
		} else if err := checkPool4sValid(subnet, pools, reservedPools,
			reservations); err != nil {
			log.Warnf("subnet %s pool4s is invalid: %s", subnet.Subnet, err.Error())
		} else {
			subnet.SubnetId = uint64(oldSubnetsLen + len(subnets) + 1)
			subnet.SetID(strconv.Itoa(int(subnet.SubnetId)))
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
		return nil, nil, nil, nil
	}

	var sqls []string
	reqsForCreate := make(map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest)
	reqsForDelete := make(map[string]*pbdhcpagent.DeleteSubnets4Request)
	subnetAndNodes := make(map[uint64][]string)
	sqls = append(sqls,
		subnet4sToInsertSqlAndRequest(subnets, reqsForCreate,
			reqsForDelete, subnetAndNodes))
	if len(subnetPools) != 0 {
		sqls = append(sqls, pool4sToInsertSqlAndRequest(subnetPools,
			reqsForCreate, subnetAndNodes))
	}

	if len(subnetReservedPools) != 0 {
		sqls = append(sqls, reservedPool4sToInsertSqlAndRequest(subnetReservedPools,
			reqsForCreate, subnetAndNodes))
	}

	if len(subnetReservations) != 0 {
		sqls = append(sqls, reservation4sToInsertSqlAndRequest(subnetReservations,
			reqsForCreate, subnetAndNodes))
	}

	return sqls, reqsForCreate, reqsForDelete, nil
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
		if csvutil.IsSpaceField(field) {
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
		case FieldNameOption60:
			subnet.ClientClass = field
		case FieldNameOption82:
			subnet.RelayAgentAddresses = strings.Split(strings.TrimSpace(field), ",")
		case FieldNameOption66:
			subnet.TftpServer = strings.TrimSpace(field)
		case FieldNameOption67:
			subnet.Bootfile = field
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
		if poolSlices := strings.Split(poolStr, "-"); len(poolSlices) != 3 {
			return nil, fmt.Errorf("parse subnet4 pool %s failed with wrong regexp",
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
		if poolSlices := strings.Split(poolStr, "-"); len(poolSlices) != 3 {
			return nil, fmt.Errorf("parse subnet4 reserved pool %s failed with wrong regexp",
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
		if reservationSlices := strings.Split(reservationStr,
			"-"); len(reservationSlices) != 3 {
			return nil, fmt.Errorf("parse subnet4 reservation %s failed with wrong regexp",
				reservationStr)
		} else {
			reservations = append(reservations, &resource.Reservation4{
				HwAddress: reservationSlices[0],
				IpAddress: reservationSlices[1],
				Comment:   reservationSlices[2],
			})
		}
	}

	return reservations, nil
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

func checkReservationsValid(subnet4 *resource.Subnet4, reservations []*resource.Reservation4) error {
	ipMacs := make(map[string]struct{})
	for _, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return err
		}

		if checkIPsBelongsToIpnet(subnet4.Ipnet, reservation.Ip) == false {
			return fmt.Errorf("reservation %s not belongs to subnet %s",
				reservation.IpAddress, subnet4.Subnet)
		}

		if _, ok := ipMacs[reservation.IpAddress]; ok {
			return fmt.Errorf("duplicate reservation with ip %s", reservation.IpAddress)
		} else if _, ok := ipMacs[reservation.HwAddress]; ok {
			return fmt.Errorf("duplicate reservation with mac %s", reservation.HwAddress)
		} else {
			ipMacs[reservation.IpAddress] = struct{}{}
			ipMacs[reservation.HwAddress] = struct{}{}
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

		if checkIPsBelongsToIpnet(subnet4.Ipnet, reservedPools[i].BeginIp,
			reservedPools[i].EndIp) == false {
			return fmt.Errorf("reserved pool %s not belongs to subnet %s",
				reservedPools[i].String(), subnet4.Subnet)
		}

		for j := i + 1; j < reservedPoolsLen; j++ {
			if reservedPools[i].CheckConflictWithAnother(reservedPools[j]) {
				return fmt.Errorf("reserved pool %s conflict with another %s",
					reservedPools[i].String(), reservedPools[j].String())
			}
		}

		for _, reservation := range reservations {
			if reservedPools[i].Contains(reservation.IpAddress) {
				return fmt.Errorf("reserved pool %s conflict with reservation %s",
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

		if checkIPsBelongsToIpnet(subnet4.Ipnet,
			pools[i].BeginIp, pools[i].EndIp) == false {
			return fmt.Errorf("pool %s not belongs to subnet %s",
				pools[i].String(), subnet4.Subnet)
		}

		for j := i + 1; j < poolsLen; j++ {
			if pools[i].CheckConflictWithAnother(pools[j]) {
				return fmt.Errorf("pool %s conflict with another %s",
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

func subnet4sToInsertSqlAndRequest(subnets []*resource.Subnet4, reqsForCreate map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, reqsForDelete map[string]*pbdhcpagent.DeleteSubnets4Request, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_subnet4 values ")
	for _, subnet := range subnets {
		buf.WriteString(subnet4ToInsertDBSqlString(subnet))
		nodes := subnetAndNodes[subnet.SubnetId]
		nodes = append(nodes, subnet.Nodes...)
		subnetAndNodes[subnet.SubnetId] = nodes
		pbSubnet := subnet4ToCreateSubnet4Request(subnet)
		for _, node := range subnet.Nodes {
			createReq, ok := reqsForCreate[node]
			deleteReq := reqsForDelete[node]
			if ok == false {
				createReq = &pbdhcpagent.CreateSubnets4AndPoolsRequest{}
				deleteReq = &pbdhcpagent.DeleteSubnets4Request{}
			}
			createReq.Subnets = append(createReq.Subnets, pbSubnet)
			deleteReq.Ids = append(deleteReq.Ids, subnet.SubnetId)
			reqsForCreate[node] = createReq
			reqsForDelete[node] = deleteReq
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func pool4sToInsertSqlAndRequest(subnetPools map[uint64][]*resource.Pool4, reqs map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_pool4 values ")
	for subnetId, pools := range subnetPools {
		for _, pool := range pools {
			buf.WriteString(pool4ToInsertDBSqlString(subnetId, pool))
			pbPool := pool4ToCreatePool4Request(subnetId, pool)
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqs[node]; ok {
					req.Pools = append(req.Pools, pbPool)
				}
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func reservedPool4sToInsertSqlAndRequest(subnetReservedPools map[uint64][]*resource.ReservedPool4, reqs map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_reserved_pool4 values ")
	for subnetId, pools := range subnetReservedPools {
		for _, pool := range pools {
			buf.WriteString(reservedPool4ToInsertDBSqlString(subnetId, pool))
			pbReservedPool := reservedPool4ToCreateReservedPool4Request(subnetId, pool)
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqs[node]; ok {
					req.ReservedPools = append(req.ReservedPools, pbReservedPool)
				}
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func reservation4sToInsertSqlAndRequest(subnetReservations map[uint64][]*resource.Reservation4, reqs map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, subnetAndNodes map[uint64][]string) string {
	var buf bytes.Buffer
	buf.WriteString("insert into gr_reservation4 values ")
	for subnetId, reservations := range subnetReservations {
		for _, reservation := range reservations {
			buf.WriteString(reservation4ToInsertDBSqlString(subnetId, reservation))
			pbReservation := reservation4ToCreateReservation4Request(subnetId, reservation)
			for _, node := range subnetAndNodes[subnetId] {
				if req, ok := reqs[node]; ok {
					req.Reservations = append(req.Reservations, pbReservation)
				}
			}
		}
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func sendCreateSubnet4sAndPoolsCmdToDHCPAgent(reqsForCreate map[string]*pbdhcpagent.CreateSubnets4AndPoolsRequest, reqsForDelete map[string]*pbdhcpagent.DeleteSubnets4Request) error {
	if len(reqsForCreate) == 0 {
		return nil
	}

	var sentryNodes []string
	for node := range reqsForCreate {
		sentryNodes = append(sentryNodes, node)
	}

	nodes, err := getDHCPNodes(sentryNodes, true)
	if err != nil {
		return err
	}

	serverNodes := nodes[len(sentryNodes):]
	var succeedNodes []string
	for node, req := range reqsForCreate {
		if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
			append(serverNodes, node), dhcpservice.CreateSubnet4sAndPools,
			req); err != nil {
			deleteSubnets(reqsForDelete, succeedNodes, serverNodes)
			return err
		}

		succeedNodes = append(succeedNodes, node)
	}

	return nil
}

func deleteSubnets(reqs map[string]*pbdhcpagent.DeleteSubnets4Request, sentryNodes, serverNodes []string) {
	for _, node := range sentryNodes {
		if req, ok := reqs[node]; ok {
			if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
				append(serverNodes, node), dhcpservice.DeleteSubnet4s, req); err != nil {
				log.Errorf("delete subnets with node %s when rollback failed: %s",
					node, err.Error())
			}
		}
	}
}

func (h *Subnet4Handler) exportCSV(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var subnet4s []*resource.Subnet4
	var pools []*resource.Pool4
	var reservedPools []*resource.ReservedPool4
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{"orderby": "subnet_id"},
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
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export subnet4 failed: %s", err.Error()))
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
		reservationSlices = append(reservationSlices, reservation.String()+"-"+reservation.Comment)
		subnetReservations[reservation.Subnet4] = reservationSlices
	}

	var strMatrix [][]string
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

	if filepath, err := csvutil.WriteCSVFile(Subnet4FileNamePrefix+
		time.Now().Format(csvutil.TimeFormat), TableHeaderSubnet4, strMatrix); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export device failed: %s", err.Error()))
	} else {
		return &csvutil.ExportFile{Path: filepath}, nil
	}
}

func (h *Subnet4Handler) exportCSVTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if filepath, err := csvutil.WriteCSVFile(Subnet4TemplateFileName,
		TableHeaderSubnet4, nil); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export subnet4 template failed: %s", err.Error()))
	} else {
		return &csvutil.ExportFile{Path: filepath}, nil
	}
}

func (h *Subnet4Handler) updateNodes(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	subnetNode, ok := ctx.Resource.GetAction().Input.(*resource.SubnetNode)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action update subnet4 %s nodes input invalid", subnetID))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, subnetID)
		if err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			"nodes": subnetNode.Nodes},
			map[string]interface{}{restdb.IDField: subnetID}); err != nil {
			return err
		}

		return sendUpdateSubnet4NodesCmdToDHCPAgent(tx, subnet4, subnetNode.Nodes)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet4 %s nodes failed: %s", subnetID, err.Error()))
	}

	return nil, nil
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
		if nodes, err := getDHCPNodes(deleteSlices, isv4); err != nil {
			return nil, nil, err
		} else {
			deleteSlices = nodes
		}
	}

	if len(oldNodes) == 0 && len(createSlices) != 0 {
		if nodes, err := getDHCPNodes(createSlices, isv4); err != nil {
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

	nodesForDelete, nodesForCreate, err := getChangedNodes(subnet4.Nodes, newNodes, true)
	if err != nil {
		return err
	}

	if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
		nodesForDelete, dhcpservice.DeleteSubnet4,
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

	if succeedNodes, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
		nodesForCreate, cmd, req); err != nil {
		if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
			succeedNodes, dhcpservice.DeleteSubnet4,
			&pbdhcpagent.DeleteSubnet4Request{Id: subnet4.SubnetId}); err != nil {
			log.Errorf("delete subnet %s with node %v when rollback failed: %s",
				subnet4.Subnet, succeedNodes, err.Error())
		}
		return err
	}

	return nil
}

func genCreateSubnets4AndPoolsRequestWithSubnet4(tx restdb.Transaction, subnet4 *resource.Subnet4) (proto.Message, dhcpservice.DHCPCmd, error) {
	var pools []*resource.Pool4
	var reservedPools []*resource.ReservedPool4
	var reservations []*resource.Reservation4
	if err := tx.Fill(map[string]interface{}{"subnet4": subnet4.GetID()}, &pools); err != nil {
		return nil, "", err
	}

	if err := tx.Fill(map[string]interface{}{"subnet4": subnet4.GetID()},
		&reservedPools); err != nil {
		return nil, "", err
	}

	if err := tx.Fill(map[string]interface{}{"subnet4": subnet4.GetID()},
		&reservations); err != nil {
		return nil, "", err
	}

	if len(pools) == 0 && len(reservedPools) == 0 && len(reservations) == 0 {
		return subnet4ToCreateSubnet4Request(subnet4), dhcpservice.CreateSubnet4, nil
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

	return req, dhcpservice.CreateSubnet4sAndPools, nil
}

func (h *Subnet4Handler) couldBeCreated(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	couldBeCreatedSubnet, ok := ctx.Resource.GetAction().Input.(*resource.CouldBeCreatedSubnet)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action check subnet could be created input invalid"))
	}

	if _, err := gohelperip.ParseCIDRv4(couldBeCreatedSubnet.Subnet); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action check subnet could be created input invalid: %s",
				err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkSubnet4CouldBeCreated(tx, couldBeCreatedSubnet.Subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("action check subnet could be created: %s", err.Error()))
	}

	return nil, nil
}

func (h *Subnet4Handler) listWithSubnets(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetListInput, ok := ctx.Resource.GetAction().Input.(*resource.SubnetListInput)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action list subnet input invalid"))
	}

	for _, subnet := range subnetListInput.Subnets {
		if _, err := gohelperip.ParseCIDRv4(subnet); err != nil {

			return nil, resterror.NewAPIError(resterror.InvalidFormat,
				fmt.Sprintf("action check subnet could be created input invalid: %s",
					err.Error()))
		}
	}

	var subnets []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets,
			fmt.Sprintf("select * from gr_subnet4 where subnet in ('%s')",
				strings.Join(subnetListInput.Subnets, "','")))
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("action list subnet failed: %s", err.Error()))
	}

	if err := setSubnet4sLeasesUsedInfo(subnets,
		listSubnetContext{hasFilterSubnet: true}); err != nil {
		log.Warnf("set subnet4s leases used info failed: %s", err.Error())
	}

	return &resource.Subnet4ListOutput{Subnet4s: subnets}, nil
}
