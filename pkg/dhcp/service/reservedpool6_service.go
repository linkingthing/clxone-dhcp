package service

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	"github.com/linkingthing/clxone-dhcp/pkg/util"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type ReservedPool6Service struct {
}

func NewReservedPool6Service() *ReservedPool6Service {
	return &ReservedPool6Service{}
}

func (p *ReservedPool6Service) Create(subnet *resource.Subnet6, pool *resource.ReservedPool6) (restresource.Resource, error) {
	return CreateReservedPool6(subnet, pool)
}

func CreateReservedPool6(subnet *resource.Subnet6, pool *resource.ReservedPool6) (restresource.Resource, error) {
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
			resource.SqlColumnCapacity: subnet.Capacity - affectPoolsCapacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s",
				subnet.GetID(), err.Error())
		}

		for affectPoolID, capacity := range affectPools {
			if _, err := tx.Update(resource.TablePool6, map[string]interface{}{
				resource.SqlColumnCapacity: capacity,
			}, map[string]interface{}{restdb.IDField: affectPoolID}); err != nil {
				return fmt.Errorf("update subnet %s pool %s capacity to db failed: %s",
					subnet.GetID(), affectPoolID, err.Error())
			}
		}

		pool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return err
		}

		return sendCreateReservedPool6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return nil, err
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

	if checkIPsBelongsToIpnet(subnet.Ipnet, pool.BeginIp, pool.EndIp) == false {
		return fmt.Errorf("pool %s not belongs to subnet %s", pool.String(), subnet.Subnet)
	}

	return checkReservedPool6ConflictWithSubnet6Pools(tx, subnet.GetID(), pool)
}

func checkReservedPool6ConflictWithSubnet6Pools(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool6) error {
	if err := checkReservedPool6ConflictWithSubnet6ReservedPool6s(tx,
		subnetID, pool); err != nil {
		return err
	}

	return checkReservedPool6ConflictWithSubnet6Reservation6s(tx, subnetID, pool)
}

func checkReservedPool6ConflictWithSubnet6ReservedPool6s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool6) error {
	var pools []*resource.ReservedPool6
	if err := tx.FillEx(&pools,
		"select * from gr_reserved_pool6 where subnet6 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetID, pool.EndIp, pool.BeginIp); err != nil {
		return fmt.Errorf("get reserved pool6s with subnet %s from db failed: %s",
			subnetID, err.Error())
	}

	if len(pools) != 0 {
		return fmt.Errorf("reserved pool6 %s conflict with reserved pool6 %s",
			pool.String(), pools[0].String())
	}

	return nil
}

func checkReservedPool6ConflictWithSubnet6Reservation6s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool6) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetID},
		&reservations); err != nil {
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
	if err := tx.FillEx(&pools,
		"select * from gr_pool6 where subnet6 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetID, reservedPool.EndIp, reservedPool.BeginIp); err != nil {
		return nil, 0, fmt.Errorf("get pools with subnet %s from db failed: %s",
			subnetID, err.Error())
	}

	var affectedCount uint64
	affectedPool6s := make(map[string]uint64)
	for _, pool := range pools {
		reservedCount := getPool6ReservedCountWithReservedPool6(pool, reservedPool)
		affectedCount += reservedCount
		if isCreate {
			affectedPool6s[pool.GetID()] = pool.Capacity - reservedCount
		} else {
			affectedPool6s[pool.GetID()] = pool.Capacity + reservedCount
		}
	}

	return affectedPool6s, affectedCount, nil
}

func getPool6ReservedCountWithReservedPool6(pool *resource.Pool6, reservedPool *resource.ReservedPool6) uint64 {
	begin := gohelperip.IPv6ToBigInt(pool.BeginIp)
	if reservedPoolBegin := gohelperip.IPv6ToBigInt(
		reservedPool.BeginIp); reservedPoolBegin.Cmp(begin) == 1 {
		begin = reservedPoolBegin
	}

	end := gohelperip.IPv6ToBigInt(pool.EndIp)
	if reservedPoolEnd := gohelperip.IPv6ToBigInt(
		reservedPool.EndIp); reservedPoolEnd.Cmp(end) == -1 {
		end = reservedPoolEnd
	}

	count, _ := resource.CalculateIpv6Pool6CapacityWithBigInt(begin, end)
	return count
}

func sendCreateReservedPool6CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.ReservedPool6) error {
	nodesForSucceed, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreateReservedPool6,
		reservedPool6ToCreateReservedPool6Request(subnetID, pool))
	if err != nil {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, kafka.DeleteReservedPool6,
			reservedPool6ToDeleteReservedPool6Request(subnetID, pool)); err != nil {
			log.Errorf("create subnet %d reserved pool6 %s failed, and rollback it failed: %s",
				subnetID, pool.String(), err.Error())
		}
	}
	return err
}

func reservedPool6ToCreateReservedPool6Request(subnetID uint64, pool *resource.ReservedPool6) *pbdhcpagent.CreateReservedPool6Request {
	return &pbdhcpagent.CreateReservedPool6Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *ReservedPool6Service) List(subnet *resource.Subnet6) (interface{}, error) {
	return GetReservedPool6List(subnet)
}

func GetReservedPool6List(subnet *resource.Subnet6) ([]*resource.ReservedPool6, error) {
	var pools []*resource.ReservedPool6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet6: subnet.GetID(),
			util.SqlOrderBy:           resource.SqlColumnBeginIp}, &pools)
	}); err != nil {
		return nil, err
	}

	return pools, nil
}

func (p *ReservedPool6Service) Get(subnetID, poolID string) (restresource.Resource, error) {
	var pools []*resource.ReservedPool6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools)
	}); err != nil {
		return nil, err
	}

	if len(pools) != 1 {
		return nil, fmt.Errorf("no found pool %s with subnet %s", poolID, subnetID)
	}

	return pools[0], nil
}

func (p *ReservedPool6Service) Delete(subnet *resource.Subnet6, pool *resource.ReservedPool6) error {
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
			resource.SqlColumnCapacity: subnet.Capacity + affectPoolsCapacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s",
				subnet.GetID(), err.Error())
		}

		for affectPoolID, capacity := range affectPools {
			if _, err := tx.Update(resource.TablePool6, map[string]interface{}{
				resource.SqlColumnCapacity: capacity,
			}, map[string]interface{}{restdb.IDField: affectPoolID}); err != nil {
				return fmt.Errorf("update subnet %s pool %s capacity to db failed: %s",
					subnet.GetID(), affectPoolID, err.Error())
			}
		}

		if _, err := tx.Delete(resource.TableReservedPool6, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservedPool6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return err
	}

	return nil
}

func setReservedPool6FromDB(tx restdb.Transaction, pool *resource.ReservedPool6) error {
	var pools []*resource.ReservedPool6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pool.GetID()},
		&pools); err != nil {
		return fmt.Errorf("get pool from db failed: %s", err.Error())
	}

	if len(pools) == 0 {
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

func sendDeleteReservedPool6CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.ReservedPool6) error {
	_, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeleteReservedPool6,
		reservedPool6ToDeleteReservedPool6Request(subnetID, pool))
	return err
}

func reservedPool6ToDeleteReservedPool6Request(subnetID uint64, pool *resource.ReservedPool6) *pbdhcpagent.DeleteReservedPool6Request {
	return &pbdhcpagent.DeleteReservedPool6Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *ReservedPool6Service) ActionValidTemplate(subnet *resource.Subnet6, pool *resource.ReservedPool6,
	templateInfo *resource.TemplateInfo) (interface{}, error) {
	pool.Template = templateInfo.Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkReservedPool6CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, fmt.Errorf("template %s invalid: %s", pool.Template, err.Error())
	}

	return &resource.TemplatePool{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress}, nil
}

func (p *ReservedPool6Service) Update(pool *resource.ReservedPool6) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservedPool6, map[string]interface{}{
			util.SqlColumnsComment: pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found reserved pool6 %s", pool.GetID())
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return pool, nil
}
