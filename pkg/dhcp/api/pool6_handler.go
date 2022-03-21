package api

import (
	"context"
	"fmt"
	"net"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
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
			fmt.Sprintf("create pool6 params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool6CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := recalculatePool6Capacity(tx, subnet.GetID(), pool); err != nil {
			return fmt.Errorf("recalculate pool6 capacity failed: %s", err.Error())
		}

		if err := updateSubnet6CapacityWithPool6(tx, subnet.GetID(),
			subnet.Capacity+pool.Capacity); err != nil {
			return err
		}

		pool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return err
		}

		return sendCreatePool6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool6 %s with subnet6 %s failed: %s",
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

	if checkIPsBelongsToIpnet(subnet.Ipnet, pool.BeginIp, pool.EndIp) == false {
		return fmt.Errorf("pool6 %s not belongs to subnet6 %s",
			pool.String(), subnet.Subnet)
	}

	if conflictPools, err := getPool6sWithBeginAndEndIp(tx, subnet.GetID(),
		pool.BeginIp, pool.EndIp); err != nil {
		return err
	} else if len(conflictPools) != 0 {
		return fmt.Errorf("pool6 %s conflict with pool6 %s",
			pool.String(), conflictPools[0].String())
	}

	return nil
}

func checkSubnet6IfCanCreateDynamicPool(subnet *resource.Subnet6) error {
	if subnet.UseEui64 {
		return fmt.Errorf("subnet6 use EUI64, can not create dynamic pool")
	}

	if ones, _ := subnet.Ipnet.Mask.Size(); ones < 64 {
		return fmt.Errorf(
			"only can create dynamic pool6 when subnet mask >= 64, current is %d", ones)
	}

	return nil
}

func getPool6sWithBeginAndEndIp(tx restdb.Transaction, subnetID string, begin, end net.IP) ([]*resource.Pool6, error) {
	var pools []*resource.Pool6
	if err := tx.FillEx(&pools,
		"select * from gr_pool6 where subnet6 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetID, end, begin); err != nil {
		return nil, fmt.Errorf("get pool6s with subnet6 %s from db failed: %s",
			subnetID, err.Error())
	} else {
		return pools, nil
	}
}

func recalculatePool6Capacity(tx restdb.Transaction, subnetID string, pool *resource.Pool6) error {
	reservations, err := getReservation6sWithIpsExists(tx, subnetID)
	if err != nil {
		return err
	}

	reservedpools, err := getReservedPool6sWithBeginAndEndIp(tx, subnetID,
		pool.BeginIp, pool.EndIp)
	if err != nil {
		return err
	}

	recalculatePool6CapacityWithReservations(pool, reservations)
	recalculatePool6CapacityWithReservedPools(pool, reservedpools)
	return nil
}

func getReservation6sWithIpsExists(tx restdb.Transaction, subnetID string) ([]*resource.Reservation6, error) {
	var reservations []*resource.Reservation6
	err := tx.FillEx(&reservations,
		"select * from gr_reservation6 where subnet6 = $1 and ip_addresses != '{}'",
		subnetID)
	return reservations, err
}

func recalculatePool6CapacityWithReservations(pool *resource.Pool6, reservations []*resource.Reservation6) {
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			if pool.Contains(ipAddress) {
				pool.Capacity -= 1
			}
		}
	}
}

func recalculatePool6CapacityWithReservedPools(pool *resource.Pool6, reservedpools []*resource.ReservedPool6) {
	for _, reservedpool := range reservedpools {
		pool.Capacity -= getPool6ReservedCountWithReservedPool6(pool, reservedpool)
	}
}

func updateSubnet6CapacityWithPool6(tx restdb.Transaction, subnetID string, capacity uint64) error {
	if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
		"capacity": capacity,
	}, map[string]interface{}{restdb.IDField: subnetID}); err != nil {
		return fmt.Errorf("update subnet6 %s capacity to db failed: %s",
			subnetID, err.Error())
	} else {
		return nil
	}
}

func sendCreatePool6CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.Pool6) error {
	nodesForSucceed, err := sendDHCPCmdWithNodes(false, nodes, dhcpservice.CreatePool6,
		pool6ToCreatePool6Request(subnetID, pool))
	if err != nil {
		if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, dhcpservice.DeletePool6,
			pool6ToDeletePool6Request(subnetID, pool)); err != nil {
			log.Errorf("create subnet6 %d pool6 %s failed, and rollback it failed: %s",
				subnetID, pool.String(), err.Error())
		}
	}

	return err
}

func pool6ToCreatePool6Request(subnetID uint64, pool *resource.Pool6) *pbdhcpagent.CreatePool6Request {
	return &pbdhcpagent.CreatePool6Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *Pool6Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var pools []*resource.Pool6
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{"subnet6": subnetID,
			"orderby": "begin_ip"}, &pools)
		if err != nil {
			return err
		}

		reservations, err = getReservation6sWithIpsExists(tx, subnetID)
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pool6s with subnet6 %s from db failed: %s",
				subnetID, err.Error()))
	}

	poolsLeases := loadPool6sLeases(subnetID, pools, reservations)
	for _, pool := range pools {
		setPool6LeasesUsedRatio(pool, poolsLeases[pool.GetID()])
	}

	return pools, nil
}

func loadPool6sLeases(subnetID string, pools []*resource.Pool6, reservations []*resource.Reservation6) map[string]uint64 {
	resp, err := getSubnet6Leases(subnetIDStrToUint64(subnetID))
	if err != nil {
		log.Warnf("get subnet6 %s leases failed: %s", subnetID, err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := reservationIpMapFromReservation6s(reservations)
	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if _, ok := reservationMap[lease.GetAddress()]; ok {
			continue
		}

		for _, pool := range pools {
			if pool.Capacity != 0 && pool.Contains(lease.GetAddress()) {
				leasesCount[pool.GetID()] += 1
				break
			}
		}
	}

	return leasesCount
}

func getSubnet6Leases(subnetId uint64) (*pbdhcpagent.GetLeases6Response, error) {
	return grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Leases(context.TODO(),
		&pbdhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
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
	var pools []*resource.Pool6
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools)
		if err != nil {
			return err
		} else if len(pools) != 1 {
			return fmt.Errorf("no found pool6 %s with subnet6 %s", poolID, subnetID)
		}

		reservations, err = getReservation6sWithIpsExists(tx, subnetID)
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool6 %s with subnet6 %s from db failed: %s",
				poolID, subnetID, err.Error()))
	}

	leasesCount, err := getPool6LeasesCount(pools[0], reservations)
	if err != nil {
		log.Warnf("get pool6 %s with subnet6 %s from db failed: %s",
			poolID, subnetID, err.Error())
	}

	setPool6LeasesUsedRatio(pools[0], leasesCount)
	return pools[0], nil
}

func getPool6LeasesCount(pool *resource.Pool6, reservations []*resource.Reservation6) (uint64, error) {
	if pool.Capacity == 0 {
		return 0, nil
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetPool6Leases(context.TODO(),
		&pbdhcpagent.GetPool6LeasesRequest{
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

	reservationMap := reservationIpMapFromReservation6s(reservations)
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
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool6CouldBeDeleted(tx, subnet, pool); err != nil {
			return err
		}

		if err := updateSubnet6CapacityWithPool6(tx, subnet.GetID(),
			subnet.Capacity-pool.Capacity); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TablePool6, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendDeletePool6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool6 %s with subnet6 %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func checkPool6CouldBeDeleted(tx restdb.Transaction, subnet *resource.Subnet6, pool *resource.Pool6) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	}

	if err := setPool6FromDB(tx, pool); err != nil {
		return err
	}

	reservations, err := getReservation6sWithIpsExists(tx, subnet.GetID())
	if err != nil {
		return err
	}

	if leasesCount, err := getPool6LeasesCount(pool, reservations); err != nil {
		return fmt.Errorf("get pool6 %s leases count failed: %s",
			pool.String(), err.Error())
	} else if leasesCount != 0 {
		return fmt.Errorf("can not delete pool6 with %d ips had been allocated",
			leasesCount)
	}

	return nil
}

func setPool6FromDB(tx restdb.Transaction, pool *resource.Pool6) error {
	var pools []*resource.Pool6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pool.GetID()},
		&pools); err != nil {
		return fmt.Errorf("get pool6 from db failed: %s", err.Error())
	} else if len(pools) == 0 {
		return fmt.Errorf("no found pool %s", pool.GetID())
	}

	pool.Subnet6 = pools[0].Subnet6
	pool.BeginAddress = pools[0].BeginAddress
	pool.BeginIp = pools[0].BeginIp
	pool.EndAddress = pools[0].EndAddress
	pool.EndIp = pools[0].EndIp
	pool.Capacity = pools[0].Capacity
	return nil
}

func sendDeletePool6CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.Pool6) error {
	_, err := sendDHCPCmdWithNodes(false, nodes, dhcpservice.DeletePool6,
		pool6ToDeletePool6Request(subnetID, pool))
	return err
}

func pool6ToDeletePool6Request(subnetID uint64, pool *resource.Pool6) *pbdhcpagent.DeletePool6Request {
	return &pbdhcpagent.DeletePool6Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *Pool6Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.Pool6)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePool6, map[string]interface{}{
			"comment": pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found pool6 %s", pool.GetID())
		}

		return nil
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update pool6 %s with subnet6 %s failed: %s",
				pool.String(), ctx.Resource.GetParent().GetID(), err.Error()))
	}

	return pool, nil
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
			fmt.Sprintf("template6 %s invalid: %s", pool.Template, err.Error()))
	}

	return &resource.TemplatePool{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress}, nil
}
