package api

import (
	"context"
	"fmt"
	"net"
	"sort"
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

type Pool4Handler struct {
}

func NewPool4Handler() *Pool4Handler {
	return &Pool4Handler{}
}

func (p *Pool4Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.Pool4)
	if err := pool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool4CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := recalculatePool4Capacity(tx, subnet.GetID(), pool); err != nil {
			return fmt.Errorf("recalculate pool capacity failed: %s", err.Error())
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			"capacity": subnet.Capacity + pool.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s",
				subnet.GetID(), err.Error())
		}

		pool.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return err
		}

		return sendCreatePool4CmdToDHCPAgent(subnet.SubnetId, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return pool, nil
}

func checkPool4CouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet4, pool *resource.Pool4) error {
	if err := setSubnet4FromDB(tx, subnet); err != nil {
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

	if conflictPool, err := checkPool4ConflictWithSubnet4Pool4s(tx, subnet.GetID(), pool); err != nil {
		return err
	} else if conflictPool != nil {
		return fmt.Errorf("pool %s conflict with pool %s", pool.String(), conflictPool.String())
	}

	return nil
}

func checkPool4ConflictWithSubnet4Pool4s(tx restdb.Transaction, subnetID string, pool *resource.Pool4) (*resource.Pool4, error) {
	var pools []*resource.Pool4
	if err := tx.Fill(map[string]interface{}{"subnet4": subnetID}, &pools); err != nil {
		return nil, fmt.Errorf("get pools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, p := range pools {
		if p.CheckConflictWithAnother(pool) {
			return p, nil
		}
	}

	return nil, nil
}

func checkIPsBelongsToIpnet(ipnet net.IPNet, ips ...string) bool {
	for _, ip := range ips {
		if ipnet.Contains(net.ParseIP(ip)) == false {
			return false
		}
	}

	return true
}

func recalculatePool4Capacity(tx restdb.Transaction, subnetID string, pool *resource.Pool4) error {
	var reservations []*resource.Reservation4
	if err := tx.Fill(map[string]interface{}{"subnet4": subnetID}, &reservations); err != nil {
		return err
	}

	for _, reservation := range reservations {
		if pool.Contains(reservation.IpAddress) {
			pool.Capacity -= reservation.Capacity
		}
	}

	var reservedpools []*resource.ReservedPool4
	if err := tx.Fill(map[string]interface{}{"subnet4": subnetID}, &reservedpools); err != nil {
		return err
	}

	for _, reservedpool := range reservedpools {
		if pool.CheckConflictWithReservedPool4(reservedpool) {
			pool.Capacity -= getPool4ReservedCountWithReservedPool4(pool, reservedpool)
		}
	}

	return nil
}

func sendCreatePool4CmdToDHCPAgent(subnetID uint64, pool *resource.Pool4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreatePool4,
		&dhcpagent.CreatePool4Request{
			SubnetId:     subnetID,
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
		})
}

func (p *Pool4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	var pools resource.Pool4s
	var reservations resource.Reservation4s
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := tx.Fill(map[string]interface{}{"subnet4": subnet.GetID()}, &pools); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{"subnet4": subnet.GetID()}, &reservations)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pools with subnet %s from db failed: %s", subnet.GetID(), err.Error()))
	}

	poolsLeases := loadPool4sLeases(subnet, pools, reservations)
	for _, pool := range pools {
		setPool4LeasesUsedRatio(pool, poolsLeases[pool.GetID()])
	}

	sort.Sort(pools)
	return pools, nil
}

func loadPool4sLeases(subnet *resource.Subnet4, pools resource.Pool4s, reservations resource.Reservation4s) map[string]uint64 {
	resp, err := getSubnet4Leases(subnet.SubnetId)
	if err != nil {
		log.Warnf("get subnet %s leases failed: %s", subnet.GetID(), err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := make(map[string]struct{})
	for _, reservation := range reservations {
		reservationMap[reservation.IpAddress] = struct{}{}
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

func getSubnet4Leases(subnetId uint64) (*dhcpagent.GetLeases4Response, error) {
	return grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Leases(context.TODO(),
		&dhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
}

func setPool4LeasesUsedRatio(pool *resource.Pool4, leasesCount uint64) {
	if leasesCount != 0 && pool.Capacity != 0 {
		pool.UsedCount = leasesCount
		pool.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(pool.Capacity))
	}
}

func (p *Pool4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	poolID := ctx.Resource.GetID()
	var pools resource.Pool4s
	var reservations resource.Reservation4s
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{"subnet4": subnetID}, &reservations)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool %s with subnet %s from db failed: %s", poolID, subnetID, err.Error()))
	}

	if len(pools) != 1 {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("no found pool %s with subnet %s", poolID, subnetID))
	}

	leasesCount, err := getPool4LeasesCount(pools[0], reservations)
	if err != nil {
		log.Warnf("get pool %s with subnet %s from db failed: %s", poolID, subnetID, err.Error())
	}

	setPool4LeasesUsedRatio(pools[0], leasesCount)
	return pools[0], nil
}

func getPool4LeasesCount(pool *resource.Pool4, reservations resource.Reservation4s) (uint64, error) {
	if pool.Capacity == 0 {
		return 0, nil
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetPool4Leases(context.TODO(),
		&dhcpagent.GetPool4LeasesRequest{
			SubnetId:     subnetIDStrToUint64(pool.Subnet4),
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
		reservationMap[reservation.IpAddress] = struct{}{}
	}

	var leasesCount uint64
	for _, lease := range resp.GetLeases() {
		if _, ok := reservationMap[lease.GetAddress()]; ok == false {
			leasesCount += 1
		}
	}

	return leasesCount, nil
}

func subnetIDStrToUint64(subnetID string) uint64 {
	id, _ := strconv.ParseUint(subnetID, 10, 64)
	return id
}

func (p *Pool4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.Pool4)
	var reservations resource.Reservation4s
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setPool4FromDB(tx, pool); err != nil {
			return err
		}

		if err := tx.Fill(map[string]interface{}{"subnet4": subnet.GetID()}, &reservations); err != nil {
			return err
		}

		if leasesCount, err := getPool4LeasesCount(pool, reservations); err != nil {
			return fmt.Errorf("get pool %s leases count failed: %s", pool.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete pool with %d ips had been allocated", leasesCount)
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			"capacity": subnet.Capacity - pool.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
		}

		if _, err := tx.Delete(resource.TablePool4, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendDeletePool4CmdToDHCPAgent(subnet.SubnetId, pool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func setPool4FromDB(tx restdb.Transaction, pool *resource.Pool4) error {
	var pools []*resource.Pool4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pool.GetID()}, &pools); err != nil {
		return fmt.Errorf("get pool from db failed: %s", err.Error())
	}

	if len(pools) == 0 {
		return fmt.Errorf("no found pool %s", pool.GetID())
	}

	pool.Subnet4 = pools[0].Subnet4
	pool.BeginAddress = pools[0].BeginAddress
	pool.EndAddress = pools[0].EndAddress
	pool.Capacity = pools[0].Capacity
	return nil
}

func sendDeletePool4CmdToDHCPAgent(subnetID uint64, pool *resource.Pool4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeletePool4,
		&dhcpagent.DeletePool4Request{
			SubnetId:     subnetID,
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
		})
}

func (h *Pool4Handler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameValidTemplate:
		return h.validTemplate(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *Pool4Handler) validTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.Pool4)
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("parse action refresh input invalid"))
	}

	pool.Template = templateInfo.Template

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkPool4CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("template %s invalid: %s", pool.Template, err.Error()))
	}

	return &resource.TemplatePool{BeginAddress: pool.BeginAddress, EndAddress: pool.EndAddress}, nil
}
