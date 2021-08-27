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

type ReservedPool4Handler struct {
}

func NewReservedPool4Handler() *ReservedPool4Handler {
	return &ReservedPool4Handler{}
}

func (p *ReservedPool4Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.ReservedPool4)
	if err := pool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create reserved pool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservedPool4CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		affectPools, affectPoolsCapacity, err := recalculatePool4sCapacityWithReservedPool4(
			tx, subnet.GetID(), pool, true)
		if err != nil {
			return fmt.Errorf("recalculate pools capacity failed: %s", err.Error())
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			"capacity": subnet.Capacity - affectPoolsCapacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
		}

		for affectPoolID, capacity := range affectPools {
			if _, err := tx.Update(resource.TablePool4, map[string]interface{}{
				"capacity": capacity,
			}, map[string]interface{}{restdb.IDField: affectPoolID}); err != nil {
				return fmt.Errorf("update subnet %s pool %s capacity to db failed: %s",
					subnet.GetID(), affectPoolID, err.Error())
			}
		}

		pool.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return err
		}

		return sendCreateReservedPool4CmdToDHCPAgent(subnet.SubnetId, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return pool, nil
}

func checkReservedPool4CouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet4, pool *resource.ReservedPool4) error {
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

	if err := checkReservedPool4ConflictWithSubnet4Pools(tx, subnet.GetID(), pool); err != nil {
		return err
	}

	return nil
}

func checkReservedPool4ConflictWithSubnet4Pools(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool4) error {
	if err := checkReservedPool4ConflictWithSubnet4ReservedPool4s(tx, subnetID, pool); err != nil {
		return err
	}

	return checkReservedPool4ConflictWithSubnet4Reservation4s(tx, subnetID, pool)
}

func checkReservedPool4ConflictWithSubnet4ReservedPool4s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool4) error {
	var pools []*resource.ReservedPool4
	if err := tx.Fill(map[string]interface{}{"subnet4": subnetID}, &pools); err != nil {
		return fmt.Errorf("get reserved pools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, p := range pools {
		if p.CheckConflictWithAnother(pool) {
			return fmt.Errorf("reserved pool %s conflict with exists reserved pool %s",
				pool.String(), p.String())
		}
	}

	return nil
}

func checkReservedPool4ConflictWithSubnet4Reservation4s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool4) error {
	var reservations []*resource.Reservation4
	if err := tx.Fill(map[string]interface{}{"subnet4": subnetID}, &reservations); err != nil {
		return err
	}

	for _, reservation := range reservations {
		if pool.Contains(reservation.IpAddress) {
			return fmt.Errorf("reserved pool %s conflict with reservation %s",
				pool.String(), reservation.String())
		}
	}

	return nil
}

func recalculatePool4sCapacityWithReservedPool4(tx restdb.Transaction, subnetID string, reservedPool *resource.ReservedPool4, isCreate bool) (map[string]uint64, uint64, error) {
	var pools []*resource.Pool4
	if err := tx.Fill(map[string]interface{}{"subnet4": subnetID}, &pools); err != nil {
		return nil, 0, err
	}

	var affectedCount uint64
	affectedPool4s := make(map[string]uint64)
	for _, pool := range pools {
		if pool.CheckConflictWithReservedPool4(reservedPool) {
			reservedCount := getPool4ReservedCountWithReservedPool4(pool, reservedPool)
			affectedCount += reservedCount
			if isCreate {
				affectedPool4s[pool.GetID()] = pool.Capacity - reservedCount
			} else {
				affectedPool4s[pool.GetID()] = pool.Capacity + reservedCount
			}
		}
	}

	return affectedPool4s, affectedCount, nil
}

func getPool4ReservedCountWithReservedPool4(pool *resource.Pool4, reservedPool *resource.ReservedPool4) uint64 {
	begin, _ := util.Ipv4StringToUint32(pool.BeginAddress)
	if reservedPoolBegin, _ := util.Ipv4StringToUint32(reservedPool.BeginAddress); reservedPoolBegin > begin {
		begin = reservedPoolBegin
	}

	end, _ := util.Ipv4StringToUint32(pool.EndAddress)
	if reservedPoolEnd, _ := util.Ipv4StringToUint32(reservedPool.EndAddress); reservedPoolEnd < end {
		end = reservedPoolEnd
	}

	return uint64(end - begin + 1)
}

func sendCreateReservedPool4CmdToDHCPAgent(subnetID uint64, pool *resource.ReservedPool4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateReservedPool4,
		reservedPool4ToPbCreateReservedPool4Request(subnetID, pool))
}

func reservedPool4ToPbCreateReservedPool4Request(subnetID uint64, pool *resource.ReservedPool4) *dhcpagent.CreateReservedPool4Request {
	return &dhcpagent.CreateReservedPool4Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *ReservedPool4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var pools resource.ReservedPool4s
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{"subnet4": subnetID}, &pools)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list reserved pools with subnet %s from db failed: %s",
				subnetID, err.Error()))
	}

	sort.Sort(pools)
	return pools, nil
}

func (p *ReservedPool4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	poolID := ctx.Resource.GetID()
	var pools resource.ReservedPool4s
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool %s with subnet %s from db failed: %s",
				poolID, subnetID, err.Error()))
	}

	if len(pools) != 1 {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("no found pool %s with subnet %s", poolID, subnetID))
	}

	return pools[0], nil
}

func (p *ReservedPool4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.ReservedPool4)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservedPool4FromDB(tx, pool); err != nil {
			return err
		}

		affectPools, affectPoolsCapacity, err := recalculatePool4sCapacityWithReservedPool4(
			tx, subnet.GetID(), pool, false)
		if err != nil {
			return fmt.Errorf("recalculate pools capacity failed: %s", err.Error())
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			"capacity": subnet.Capacity + affectPoolsCapacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
		}

		for affectPoolID, capacity := range affectPools {
			if _, err := tx.Update(resource.TablePool4, map[string]interface{}{
				"capacity": capacity,
			}, map[string]interface{}{restdb.IDField: affectPoolID}); err != nil {
				return fmt.Errorf("update subnet %s pool %s capacity to db failed: %s",
					subnet.GetID(), affectPoolID, err.Error())
			}
		}

		if _, err := tx.Delete(resource.TableReservedPool4, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservedPool4CmdToDHCPAgent(subnet.SubnetId, pool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pool %s with subnet %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func setReservedPool4FromDB(tx restdb.Transaction, pool *resource.ReservedPool4) error {
	var pools []*resource.ReservedPool4
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

func sendDeleteReservedPool4CmdToDHCPAgent(subnetID uint64, pool *resource.ReservedPool4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteReservedPool4,
		&dhcpagent.DeleteReservedPool4Request{
			SubnetId:     subnetID,
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
		})
}

func (h *ReservedPool4Handler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameValidTemplate:
		return h.validTemplate(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *ReservedPool4Handler) validTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.ReservedPool4)
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("parse action valid pool template input invalid"))
	}

	pool.Template = templateInfo.Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkReservedPool4CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("template %s invalid: %s", pool.Template, err.Error()))
	}

	return &resource.TemplatePool{BeginAddress: pool.BeginAddress, EndAddress: pool.EndAddress}, nil
}
