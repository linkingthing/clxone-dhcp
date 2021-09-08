package api

import (
	"context"
	"fmt"
	"net"
	"sort"

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

type Pool6Handler struct {
}

func NewPool6Handler() *Pool6Handler {
	return &Pool6Handler{}
}

func (p *Pool6Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.Pool6)
	if err := pool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool6CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := recalculatePool6Capacity(tx, subnet.GetID(), pool); err != nil {
			return fmt.Errorf("recalculate pool capacity failed: %s", err.Error())
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			"capacity": subnet.Capacity + pool.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s",
				subnet.GetID(), err.Error())
		}

		pool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return err
		}

		return sendCreatePool6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return pool, nil
}

func checkPool6CouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet6, pool *resource.Pool6) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	}

	if err := checkSubnet6IfCanCreateDynamicPool(subnet); err != nil {
		return err
	}

	if pool.Template != "" {
		if err := pool.ParseAddressWithTemplate(tx, subnet); err != nil {
			return err
		}
	}

	if checkIPsBelongsToIpnet(subnet.Ipnet, pool.BeginAddress, pool.EndAddress) == false {
		return fmt.Errorf("pool %s not belongs to subnet %s", pool.String(), subnet.Subnet)
	}

	return checkPool6ConflictWithSubnet6Pools(tx, subnet.GetID(), pool)
}

func checkSubnet6IfCanCreateDynamicPool(subnet *resource.Subnet6) error {
	if ones, _ := subnet.Ipnet.Mask.Size(); ones < 64 {
		return fmt.Errorf("only can create dynamic pool when subnet mask >= 64, current mask is %d",
			ones)
	}

	return nil
}

func checkPool6ConflictWithSubnet6Pools(tx restdb.Transaction, subnetID string, pool *resource.Pool6) error {
	if conflictPool, err := checkPool6ConflictWithSubnet6Pool6s(tx, subnetID, pool); err != nil {
		return err
	} else if conflictPool != nil {
		return fmt.Errorf("pool %s conflict with pool %s", pool.String(), conflictPool.String())
	}

	return nil
}

func checkPool6ConflictWithSubnet6Pool6s(tx restdb.Transaction, subnetID string, pool *resource.Pool6) (*resource.Pool6, error) {
	var pools []*resource.Pool6
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetID}, &pools); err != nil {
		return nil, fmt.Errorf("get pools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, p := range pools {
		if p.CheckConflictWithAnother(pool) {
			return p, nil
		}
	}

	return nil, nil
}

func checkIPConflictWithSubnetPdPool(tx restdb.Transaction, subnetID, ip string) error {
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetID}, &pdpools); err != nil {
		return fmt.Errorf("get pdpools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, pdpool := range pdpools {
		if _, ipnet, _ := net.ParseCIDR(pdpool.String()); ipnet.Contains(net.ParseIP(ip)) {
			return fmt.Errorf("pdpool %s contains ip %s", pdpool.String(), ip)
		}
	}

	return nil
}

func recalculatePool6Capacity(tx restdb.Transaction, subnetID string, pool *resource.Pool6) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetID}, &reservations); err != nil {
		return err
	}

	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			if pool.Contains(ipAddress) {
				pool.Capacity -= reservation.Capacity
			}
		}
	}

	var reservedpools []*resource.ReservedPool6
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetID}, &reservedpools); err != nil {
		return err
	}

	for _, reservedpool := range reservedpools {
		if pool.CheckConflictWithReservedPool6(reservedpool) {
			pool.Capacity -= getPool6ReservedCountWithReservedPool6(pool, reservedpool)
		}
	}

	return nil
}

func sendCreatePool6CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.Pool6) error {
	nodesForSucceed, err := sendDHCPCmdWithNodes(nodes, dhcpservice.CreatePool6,
		pool6ToCreatePool6Request(subnetID, pool))
	if err != nil {
		if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(nodesForSucceed,
			dhcpservice.DeletePool6, pool6ToDeletePool6Request(subnetID, pool)); err != nil {
			log.Errorf("create subnet %d pool6 %s failed, and rollback it failed: %s",
				subnetID, pool.String(), err.Error())
		}
	}

	return err
}

func pool6ToCreatePool6Request(subnetID uint64, pool *resource.Pool6) *dhcpagent.CreatePool6Request {
	return &dhcpagent.CreatePool6Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *Pool6Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	var pools resource.Pool6s
	var reservations resource.Reservation6s
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := tx.Fill(map[string]interface{}{"subnet6": subnet.GetID()}, &pools); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{"subnet6": subnet.GetID()}, &reservations)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pools with subnet %s from db failed: %s", subnet.GetID(), err.Error()))
	}

	poolsLeases := loadPool6sLeases(subnet, pools, reservations)
	for _, pool := range pools {
		setPool6LeasesUsedRatio(pool, poolsLeases[pool.GetID()])
	}

	sort.Sort(pools)
	return pools, nil
}

func loadPool6sLeases(subnet *resource.Subnet6, pools resource.Pool6s, reservations resource.Reservation6s) map[string]uint64 {
	resp, err := getSubnet6Leases(subnet.SubnetId)
	if err != nil {
		log.Warnf("get subnet %s leases failed: %s", subnet.GetID(), err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := make(map[string]struct{})
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress] = struct{}{}
		}
	}

	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if _, ok := reservationMap[lease.GetAddress()]; ok {
			continue
		}

		for _, pool := range pools {
			if pool.Capacity != 0 && pool.Contains(lease.GetAddress()) {
				count := leasesCount[pool.GetID()]
				count += 1
				leasesCount[pool.GetID()] = count
			}
		}
	}

	return leasesCount
}

func getSubnet6Leases(subnetId uint64) (*dhcpagent.GetLeases6Response, error) {
	return grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Leases(context.TODO(),
		&dhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
}

func setPool6LeasesUsedRatio(pool *resource.Pool6, leasesCount uint64) {
	if leasesCount != 0 && pool.Capacity != 0 {
		pool.UsedCount = leasesCount
		pool.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(pool.Capacity))
	}
}

func (p *Pool6Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	poolID := ctx.Resource.GetID()
	var pools resource.Pool6s
	var reservations resource.Reservation6s
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{"subnet6": subnetID}, &reservations)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool %s with subnet %s from db failed: %s", poolID, subnetID, err.Error()))
	}

	if len(pools) != 1 {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("no found pool %s with subnet %s", poolID, subnetID))
	}

	leasesCount, err := getPool6LeasesCount(pools[0], reservations)
	if err != nil {
		log.Warnf("get pool %s with subnet %s from db failed: %s", poolID, subnetID, err.Error())
	}

	setPool6LeasesUsedRatio(pools[0], leasesCount)
	return pools[0], nil
}

func getPool6LeasesCount(pool *resource.Pool6, reservations resource.Reservation6s) (uint64, error) {
	if pool.Capacity == 0 {
		return 0, nil
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetPool6Leases(context.TODO(),
		&dhcpagent.GetPool6LeasesRequest{
			SubnetId:     subnetIDStrToUint64(pool.Subnet6),
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress})

	if err != nil {
		return 0, err
	}

	if len(resp.GetLeases()) == 0 {
		return 0, nil
	}

	if len(reservations) == 0 {
		return uint64(len(resp.GetLeases())), nil
	}

	reservationMap := make(map[string]struct{})
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress] = struct{}{}
		}
	}

	var leasesCount uint64
	for _, lease := range resp.GetLeases() {
		if _, ok := reservationMap[lease.GetAddress()]; ok == false {
			leasesCount += 1
		}
	}

	return leasesCount, nil
}

func (p *Pool6Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.Pool6)
	var reservations resource.Reservation6s
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setPool6FromDB(tx, pool); err != nil {
			return err
		}

		if err := tx.Fill(map[string]interface{}{"subnet6": subnet.GetID()}, &reservations); err != nil {
			return err
		}

		if leasesCount, err := getPool6LeasesCount(pool, reservations); err != nil {
			return fmt.Errorf("get pool %s leases count failed: %s", pool.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete pool with %d ips had been allocated", leasesCount)
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			"capacity": subnet.Capacity - pool.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s",
				subnet.GetID(), err.Error())
		}

		if _, err := tx.Delete(resource.TablePool6, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendDeletePool6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func setPool6FromDB(tx restdb.Transaction, pool *resource.Pool6) error {
	var pools []*resource.Pool6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pool.GetID()}, &pools); err != nil {
		return fmt.Errorf("get pool from db failed: %s", err.Error())
	}

	if len(pools) == 0 {
		return fmt.Errorf("no found pool %s", pool.GetID())
	}

	pool.Subnet6 = pools[0].Subnet6
	pool.BeginAddress = pools[0].BeginAddress
	pool.EndAddress = pools[0].EndAddress
	pool.Capacity = pools[0].Capacity
	return nil
}

func sendDeletePool6CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.Pool6) error {
	_, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(nodes,
		dhcpservice.DeletePool6, pool6ToDeletePool6Request(subnetID, pool))
	return err
}

func pool6ToDeletePool6Request(subnetID uint64, pool *resource.Pool6) *dhcpagent.DeletePool6Request {
	return &dhcpagent.DeletePool6Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (h *Pool6Handler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameValidTemplate:
		return h.validTemplate(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *Pool6Handler) validTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.Pool6)
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("parse action refresh input invalid"))
	}

	pool.Template = templateInfo.Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkPool6CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("template %s invalid: %s", pool.Template, err.Error()))
	}

	return &resource.TemplatePool{BeginAddress: pool.BeginAddress, EndAddress: pool.EndAddress}, nil
}
