package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	dhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type Subnet6Handler struct {
}

func NewSubnet6Handler() *Subnet6Handler {
	return &Subnet6Handler{}
}

func (s *Subnet6Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet6)
	if err := subnet.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create subnet params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var subnets []*resource.Subnet6
		if err := tx.Fill(map[string]interface{}{"orderby": "subnet_id desc"}, &subnets); err != nil {
			return fmt.Errorf("get subnets from db failed: %s\n", err.Error())
		}

		if len(subnets) >= MaxSubnetsCount {
			return fmt.Errorf("subnet6s count has reached maximum (1w)")
		}

		subnet.SubnetId = 1
		if len(subnets) > 0 {
			subnet.SubnetId = subnets[0].SubnetId + 1
		}

		subnet.SetID(strconv.FormatUint(subnet.SubnetId, 10))
		if err := checkSubnet6ConflictWithSubnet6s(subnet, subnets); err != nil {
			return err
		}

		if _, err := tx.Insert(subnet); err != nil {
			return err
		}

		return sendCreateSubnet6CmdToDHCPAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create subnet %s failed: %s", subnet.Subnet, err.Error()))
	}

	return subnet, nil
}

func checkSubnet6ConflictWithSubnet6s(subnet6 *resource.Subnet6, subnets []*resource.Subnet6) error {
	for _, subnet := range subnets {
		if subnet.CheckConflictWithAnother(subnet6) {
			return fmt.Errorf("subnet6 %s conflict with subnet6 %s", subnet6.Subnet, subnet.Subnet)
		}
	}

	return nil
}

func sendCreateSubnet6CmdToDHCPAgent(subnet *resource.Subnet6) error {
	nodesForSucceed, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(subnet.Nodes,
		dhcpservice.CreateSubnet6, subnet6ToCreateSubnet6Request(subnet))
	if err != nil {
		if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, dhcpservice.DeleteSubnet6,
			&dhcpagent.DeleteSubnet6Request{Id: subnet.SubnetId}); err != nil {
			log.Errorf("create subnet6 %s failed, and rollback it failed: %s",
				subnet.Subnet, err.Error())
		}
	}

	return err
}

func subnet6ToCreateSubnet6Request(subnet *resource.Subnet6) *dhcpagent.CreateSubnet6Request {
	return &dhcpagent.CreateSubnet6Request{
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
		DnsServers:            subnet.DomainServers,
		ClientClass:           subnet.ClientClass,
		IfaceName:             subnet.IfaceName,
		RelayAgentAddresses:   subnet.RelayAgentAddresses,
		RelayAgentInterfaceId: subnet.RelayAgentInterfaceId,
	}
}

func (s *Subnet6Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	listCtx := genGetSubnetsContext(ctx, resource.TableSubnet6)
	var subnets []*resource.Subnet6
	var subnetsCount int
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if listCtx.hasPagination {
			if count, err := tx.CountEx(resource.TableSubnet6,
				listCtx.countSql, listCtx.params[:len(listCtx.params)-2]...); err != nil {
				return err
			} else {
				subnetsCount = int(count)
			}
		}

		return tx.FillEx(&subnets, listCtx.sql, listCtx.params...)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnet6s from db failed: %s", err.Error()))
	}

	if err := setSubnet6sLeasesUsedInfo(subnets, listCtx.isUseIds()); err != nil {
		log.Warnf("set subnet6s leases used info failed: %s", err.Error())
	}

	setPagination(ctx, listCtx.hasPagination, subnetsCount)
	return subnets, nil
}

func setSubnet6sLeasesUsedInfo(subnets []*resource.Subnet6, useIds bool) error {
	if len(subnets) == 0 {
		return nil
	}

	var resp *dhcpagent.GetSubnetsLeasesCountResponse
	var err error
	if useIds {
		var ids []uint64
		for _, subnet := range subnets {
			ids = append(ids, subnet.SubnetId)
		}

		resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets6LeasesCountWithIds(context.TODO(),
			&dhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: ids})
	} else {
		resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets6LeasesCount(context.TODO(),
			&dhcpagent.GetSubnetsLeasesCountRequest{})
	}

	if err != nil {
		return err
	}

	subnetsLeasesCount := resp.GetSubnetsLeasesCount()
	for _, subnet := range subnets {
		if subnet.Capacity != 0 {
			if leasesCount, ok := subnetsLeasesCount[subnet.SubnetId]; ok {
				subnet.UsedCount = leasesCount
				subnet.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(subnet.Capacity))
			}
		}
	}

	return nil
}

func (s *Subnet6Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	var subnets []*resource.Subnet6
	subnetInterface, err := restdb.GetResourceWithID(db.GetDB(), subnetID, &subnets)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet %s from db failed: %s", subnetID, err.Error()))
	}

	subnet := subnetInterface.(*resource.Subnet6)
	if err := setSubnet6LeasesUsedRatio(subnet); err != nil {
		log.Warnf("get subnet %s leases used ratio failed: %s", subnetID, err.Error())
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
		subnet.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(subnet.Capacity))
	}
	return nil
}

func getSubnet6LeasesCount(subnet *resource.Subnet6) (uint64, error) {
	if subnet.Capacity == 0 {
		return 0, nil
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6LeasesCount(context.TODO(),
		&dhcpagent.GetSubnet6LeasesCountRequest{Id: subnet.SubnetId})
	return resp.GetLeasesCount(), err
}

func (s *Subnet6Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet6)
	if err := subnet.ValidateParams(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update subnet params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			"valid_lifetime":           subnet.ValidLifetime,
			"max_valid_lifetime":       subnet.MaxValidLifetime,
			"min_valid_lifetime":       subnet.MinValidLifetime,
			"preferred_lifetime":       subnet.PreferredLifetime,
			"domain_servers":           subnet.DomainServers,
			"client_class":             subnet.ClientClass,
			"iface_name":               subnet.IfaceName,
			"relay_agent_addresses":    subnet.RelayAgentAddresses,
			"relay_agent_interface_id": subnet.RelayAgentInterfaceId,
			"tags":                     subnet.Tags,
			"network_type":             subnet.NetworkType,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendUpdateSubnet6CmdToDHCPAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return subnet, nil
}

func setSubnet6FromDB(tx restdb.Transaction, subnet *resource.Subnet6) error {
	var subnets []*resource.Subnet6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: subnet.GetID()}, &subnets); err != nil {
		return fmt.Errorf("get subnet %s from db failed: %s", subnet.GetID(), err.Error())
	}

	if len(subnets) == 0 {
		return fmt.Errorf("no found subnet %s", subnet.GetID())
	}

	subnet.SubnetId = subnets[0].SubnetId
	subnet.Capacity = subnets[0].Capacity
	subnet.Subnet = subnets[0].Subnet
	subnet.Ipnet = subnets[0].Ipnet
	subnet.Nodes = subnets[0].Nodes
	return nil
}

func sendUpdateSubnet6CmdToDHCPAgent(subnet *resource.Subnet6) error {
	_, err := sendDHCPCmdWithNodes(subnet.Nodes, dhcpservice.UpdateSubnet6,
		&dhcpagent.UpdateSubnet6Request{
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
			DnsServers:            subnet.DomainServers,
			ClientClass:           subnet.ClientClass,
			IfaceName:             subnet.IfaceName,
			RelayAgentAddresses:   subnet.RelayAgentAddresses,
			RelayAgentInterfaceId: subnet.RelayAgentInterfaceId,
		})
	return err
}

func (s *Subnet6Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.(*resource.Subnet6)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if leasesCount, err := getSubnet6LeasesCount(subnet); err != nil {
			return fmt.Errorf("get subnet %s leases count failed: %s", subnet.Subnet, err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete subnet with %d ips had been allocated", leasesCount)
		}

		if _, err := tx.Delete(resource.TableSubnet6,
			map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendDeleteSubnet6CmdToDHCPAgent(subnet, subnet.Nodes)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return nil
}

func sendDeleteSubnet6CmdToDHCPAgent(subnet *resource.Subnet6, nodes []string) error {
	_, err := sendDHCPCmdWithNodes(nodes, dhcpservice.DeleteSubnet6,
		&dhcpagent.DeleteSubnet6Request{Id: subnet.SubnetId})
	return err
}

func (h *Subnet6Handler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameUpdateNodes:
		return h.updateNodes(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *Subnet6Handler) updateNodes(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	subnetNode, ok := ctx.Resource.GetAction().Input.(*resource.SubnetNode)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action update subnet6 %s nodes input invalid", subnetID))
	}

	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{restdb.IDField: subnetID}, &subnets); err != nil {
			return err
		}

		if len(subnets) == 0 {
			return fmt.Errorf("no found subnet6 %s", subnetID)
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			"nodes": subnetNode.Nodes},
			map[string]interface{}{restdb.IDField: subnetID}); err != nil {
			return err
		}

		return sendUpdateSubnet6NodesCmdToDHCPAgent(tx, subnets[0], subnetNode.Nodes)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet6 %s nodes failed: %s", subnetID, err.Error()))
	}

	return nil, nil
}

func sendUpdateSubnet6NodesCmdToDHCPAgent(tx restdb.Transaction, subnet6 *resource.Subnet6, newNodes []string) error {
	if len(subnet6.Nodes) == 0 && len(newNodes) == 0 {
		return nil
	}

	nodesForDelete, nodesForCreate, err := getChangedNodes(subnet6.Nodes, newNodes)
	if err != nil {
		return err
	}

	if _, err := sendDHCPCmdWithNodes(nodesForDelete, dhcpservice.DeleteSubnet6,
		&dhcpagent.DeleteSubnet6Request{Id: subnet6.SubnetId}); err != nil {
		return err
	}

	if len(nodesForCreate) == 0 {
		return nil
	}

	var pools []*resource.Pool6
	var reservedPools []*resource.ReservedPool6
	var reservations []*resource.Reservation6
	var pdpools []*resource.PdPool
	var reservedPdPools []*resource.ReservedPdPool
	if err := tx.Fill(map[string]interface{}{"subnet6": subnet6.GetID()}, &pools); err != nil {
		return err
	}

	if err := tx.Fill(map[string]interface{}{"subnet6": subnet6.GetID()}, &reservedPools); err != nil {
		return err
	}

	if err := tx.Fill(map[string]interface{}{"subnet6": subnet6.GetID()}, &reservations); err != nil {
		return err
	}

	if err := tx.Fill(map[string]interface{}{"subnet6": subnet6.GetID()}, &pdpools); err != nil {
		return err
	}

	if err := tx.Fill(map[string]interface{}{"subnet6": subnet6.GetID()}, &reservedPdPools); err != nil {
		return err
	}

	req := &dhcpagent.CreateSubnets6AndPoolsRequest{
		Subnets: []*dhcpagent.CreateSubnet6Request{subnet6ToCreateSubnet6Request(subnet6)},
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

	if succeedNodes, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForCreate,
		dhcpservice.CreateSubnet6sAndPools, req); err != nil {
		if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
			succeedNodes, dhcpservice.DeleteSubnet6,
			&dhcpagent.DeleteSubnet6Request{Id: subnet6.SubnetId}); err != nil {
			log.Errorf("delete subnet %s with node %v when rollback failed: %s",
				subnet6.Subnet, succeedNodes, err.Error())
		}
		return err
	}

	return nil
}
