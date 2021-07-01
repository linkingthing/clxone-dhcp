package api

import (
	"fmt"
	"sort"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	dhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type ReservedPool6Handler struct {
}

func NewReservedPool6Handler() *ReservedPool6Handler {
	return &ReservedPool6Handler{}
}

func (p *ReservedPool6Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.ReservedPool6)
	if err := pool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservedPool6CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		affectPools, affectPoolsCapacity, err := recalculatePool6sCapacityWithReservedPool6(
			tx, subnet.GetID(), pool, true)
		if err != nil {
			return fmt.Errorf("recalculate pool6s capacity failed: %s", err.Error())
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			"capacity": subnet.Capacity - affectPoolsCapacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
		}

		for affectPoolID, capacity := range affectPools {
			if _, err := tx.Update(resource.TablePool6, map[string]interface{}{
				"capacity": capacity,
			}, map[string]interface{}{restdb.IDField: affectPoolID}); err != nil {
				return fmt.Errorf("update subnet %s pool %s capacity to db failed: %s",
					subnet.GetID(), affectPoolID, err.Error())
			}
		}

		pool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return err
		}

		return sendCreateReservedPool6CmdToDHCPAgent(subnet.SubnetId, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return pool, nil
}

func checkReservedPool6CouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet6, pool *resource.ReservedPool6) error {
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

	return checkReservedPool6ConflictWithSubnet6Pools(tx, subnet.GetID(), pool)
}

func checkReservedPool6ConflictWithSubnet6Pools(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool6) error {
	if err := checkReservedPool6ConflictWithSubnet6ReservedPool6s(tx, subnetID, pool); err != nil {
		return err
	}

	return checkReservedPool6ConflictWithSubnet6Reservation6s(tx, subnetID, pool)
}

func checkReservedPool6ConflictWithSubnet6ReservedPool6s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool6) error {
	var pools []*resource.ReservedPool6
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetID}, &pools); err != nil {
		return fmt.Errorf("get pools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, p := range pools {
		if p.CheckConflictWithAnother(pool) {
			return fmt.Errorf("reserved pool6 %s conflict with reserved pool6 %s",
				pool.String(), p.String())
		}
	}

	return nil
}

func checkReservedPool6ConflictWithSubnet6Reservation6s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool6) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetID}, &reservations); err != nil {
		return err
	}

	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			if pool.Contains(ipAddress) {
				return fmt.Errorf("reserved pool6 %s conflict with reservation6 %s ip %s",
					pool.String(), reservation.String(), ipAddress)
			}
		}
	}

	return nil
}

func recalculatePool6sCapacityWithReservedPool6(tx restdb.Transaction, subnetID string, reservedPool *resource.ReservedPool6, isCreate bool) (map[string]uint64, uint64, error) {
	var pools []*resource.Pool6
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetID}, &pools); err != nil {
		return nil, 0, err
	}

	var affectedCount uint64
	affectedPool6s := make(map[string]uint64)
	for _, pool := range pools {
		if pool.CheckConflictWithReservedPool6(reservedPool) {
			reservedCount := getPool6ReservedCountWithReservedPool6(pool, reservedPool)
			affectedCount += reservedCount
			if isCreate {
				affectedPool6s[pool.GetID()] = pool.Capacity - reservedCount
			} else {
				affectedPool6s[pool.GetID()] = pool.Capacity + reservedCount
			}
		}
	}

	return affectedPool6s, affectedCount, nil
}

func getPool6ReservedCountWithReservedPool6(pool *resource.Pool6, reservedPool *resource.ReservedPool6) uint64 {
	begin, _ := util.Ipv6StringToBigInt(pool.BeginAddress)
	if reservedPoolBegin, _ := util.Ipv6StringToBigInt(reservedPool.BeginAddress); reservedPoolBegin.Cmp(begin) == 1 {
		begin = reservedPoolBegin
	}

	end, _ := util.Ipv6StringToBigInt(pool.EndAddress)
	if reservedPoolEnd, _ := util.Ipv6StringToBigInt(reservedPool.EndAddress); reservedPoolEnd.Cmp(end) == -1 {
		end = reservedPoolEnd
	}

	return resource.Ipv6Pool6CapacityWithBigInt(begin, end)
}

func sendCreateReservedPool6CmdToDHCPAgent(subnetID uint64, pool *resource.ReservedPool6) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateReservedPool6,
		&dhcpagent.CreateReservedPool6Request{
			SubnetId:     subnetID,
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
		})
}

func (p *ReservedPool6Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	var pools resource.ReservedPool6s
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{"subnet6": subnet.GetID()}, &pools)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pools with subnet %s from db failed: %s", subnet.GetID(), err.Error()))
	}

	sort.Sort(pools)
	return pools, nil
}

func (p *ReservedPool6Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	poolID := ctx.Resource.GetID()
	var pools resource.ReservedPool6s
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool %s with subnet %s from db failed: %s", poolID, subnetID, err.Error()))
	}

	if len(pools) != 1 {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("no found pool %s with subnet %s", poolID, subnetID))
	}

	return pools[0], nil
}

func (p *ReservedPool6Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.ReservedPool6)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservedPool6FromDB(tx, pool); err != nil {
			return err
		}

		affectPools, affectPoolsCapacity, err := recalculatePool6sCapacityWithReservedPool6(
			tx, subnet.GetID(), pool, false)
		if err != nil {
			return fmt.Errorf("recalculate pool6s capacity failed: %s", err.Error())
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			"capacity": subnet.Capacity + affectPoolsCapacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
		}

		for affectPoolID, capacity := range affectPools {
			if _, err := tx.Update(resource.TablePool6, map[string]interface{}{
				"capacity": capacity,
			}, map[string]interface{}{restdb.IDField: affectPoolID}); err != nil {
				return fmt.Errorf("update subnet %s pool %s capacity to db failed: %s",
					subnet.GetID(), affectPoolID, err.Error())
			}
		}

		if _, err := tx.Delete(resource.TableReservedPool6, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservedPool6CmdToDHCPAgent(subnet.SubnetId, pool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func setReservedPool6FromDB(tx restdb.Transaction, pool *resource.ReservedPool6) error {
	var pools []*resource.ReservedPool6
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

func sendDeleteReservedPool6CmdToDHCPAgent(subnetID uint64, pool *resource.ReservedPool6) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteReservedPool6,
		&dhcpagent.DeleteReservedPool6Request{
			SubnetId:     subnetID,
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
		})
}

func (h *ReservedPool6Handler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameValidTemplate:
		return h.validTemplate(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *ReservedPool6Handler) validTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.ReservedPool6)
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("parse action refresh input invalid"))
	}

	pool.Template = templateInfo.Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkReservedPool6CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("template %s invalid: %s", pool.Template, err.Error()))
	}

	return &resource.TemplatePool{BeginAddress: pool.BeginAddress, EndAddress: pool.EndAddress}, nil
}
