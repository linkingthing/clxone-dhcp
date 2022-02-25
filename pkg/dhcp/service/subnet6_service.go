package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/golang/protobuf/proto"
	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Subnet6Service struct {
}

func NewSubnet6Service() *Subnet6Service {
	return &Subnet6Service{}
}

func (s *Subnet6Service) Create(subnet *resource.Subnet6) (restresource.Resource, error) {
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
		return nil, err
	}

	return subnet, nil
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
	nodesForSucceed, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
		subnet.Nodes, kafka.CreateSubnet6, subnet6ToCreateSubnet6Request(subnet))
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

func (s *Subnet6Service) List(ctx *restresource.Context) (interface{}, error) {
	listCtx := genGetSubnetsContext(ctx, resource.TableSubnet6)
	subnets, subnetsCount, err := GetSubnet6List(listCtx)
	if err != nil {
		return nil, err
	}
	setPagination(ctx, listCtx.hasPagination, subnetsCount)
	return subnets, nil
}

func GetSubnet6List(listCtx listSubnetContext) ([]*resource.Subnet6, int, error) {
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
		return nil, -1, err
	}

	if err := setSubnet6sLeasesUsedInfo(subnets, listCtx.isUseIds()); err != nil {
		log.Warnf("set subnet6s leases used info failed: %s", err.Error())
	}

	if nodeNames, err := GetNodeNames(false); err != nil {
		log.Warnf("get node names failed: %s", err.Error())
	} else {
		setSubnet6sNodeNames(subnets, nodeNames)
	}
	return subnets, subnetsCount, nil
}

func setSubnet6sLeasesUsedInfo(subnets []*resource.Subnet6, useIds bool) error {
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

func (s *Subnet6Service) Get(subnetID string) (restresource.Resource, error) {
	var subnets []*resource.Subnet6
	subnetInterface, err := restdb.GetResourceWithID(db.GetDB(), subnetID, &subnets)
	if err != nil {
		return nil, fmt.Errorf("get subnet %s from db failed: %s", subnetID, err.Error())
	}

	subnet := subnetInterface.(*resource.Subnet6)
	if err := setSubnet6LeasesUsedRatio(subnet); err != nil {
		log.Warnf("get subnet %s leases used ratio failed: %s", subnetID, err.Error())
	}

	if nodeNames, err := GetNodeNames(false); err != nil {
		log.Warnf("get node names failed: %s", err.Error())
	} else {
		subnet.NodeNames = getSubnetNodeNames(subnet.Nodes, nodeNames)
	}

	return subnet, nil
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
	return resp.GetLeasesCount(), err
}

func (s *Subnet6Service) Update(subnet *resource.Subnet6) (restresource.Resource, error) {
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
		return nil, err
	}

	return subnet, nil
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
		return nil, fmt.Errorf("get subnet %s from db failed: %s", subnetId, err.Error())
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet %s", subnetId)
	}

	return subnets[0], nil
}

func checkUseEUI64(tx restdb.Transaction, subnet *resource.Subnet6, newUseEUI64 bool) error {
	if newUseEUI64 {
		if ones, _ := subnet.Ipnet.Mask.Size(); ones != 64 {
			return fmt.Errorf("subnet use EUI64, mask size %d is not 64", ones)
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
		return err
	}

	return nil
}

func checkSubnet6CouldBeDelete(subnet6 *resource.Subnet6) error {
	if leasesCount, err := getSubnet6LeasesCount(subnet6); err != nil {
		return fmt.Errorf("get subnet %s leases count failed: %s",
			subnet6.Subnet, err.Error())
	} else if leasesCount != 0 {
		return fmt.Errorf("can not delete subnet with %d ips had been allocated",
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

func (s *Subnet6Service) UpdateNodes(subnetID string, subnetNode *resource.SubnetNode) (interface{}, error) {
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
		return nil, err
	}

	return nil, nil
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
			log.Errorf("delete subnet %s with node %v when rollback failed: %s",
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
			pdpoolToCreatePdPoolRequest(subnet6.SubnetId, pdpool))
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
		return nil, err
	}

	return nil, nil
}

func (s *Subnet6Service) ListWithSubnets(subnetListInput *resource.SubnetListInput) (interface{}, error) {
	for _, subnet := range subnetListInput.Subnets {
		if _, err := gohelperip.ParseCIDRv6(subnet); err != nil {
			return nil, fmt.Errorf("action check subnet could be created input subnet %s invalid: %s",
				subnet, err.Error())
		}
	}

	return GetListWithSubnet6s(subnetListInput.Subnets)
}

func GetListWithSubnet6s(prefixes []string) (*resource.Subnet6ListOutput, error) {
	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets,
			fmt.Sprintf("select * from gr_subnet6 where subnet in ('%s')",
				strings.Join(prefixes, "','")))
	}); err != nil {
		return nil, err
	}

	if err := setSubnet6sLeasesUsedInfo(subnets, true); err != nil {
		log.Warnf("set subnet6s leases used info failed: %s", err.Error())
	}

	return &resource.Subnet6ListOutput{Subnet6s: subnets}, nil
}
