package service

import (
	"context"
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Pool6Service struct {
}

func NewPool6Service() *Pool6Service {
	return &Pool6Service{}
}

func (p *Pool6Service) Create(subnet *resource.Subnet6, pool *resource.Pool6) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool6CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := recalculatePool6Capacity(tx, subnet.GetID(), pool); err != nil {
			return fmt.Errorf("recalculate pool capacity failed: %s", err.Error())
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			resource.SqlColumnCapacity: subnet.Capacity + pool.Capacity,
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
		return nil, err
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
		return fmt.Errorf("pool %s not belongs to subnet %s",
			pool.String(), subnet.Subnet)
	}

	if conflictPool, err := getConflictPool6InSubnet6(tx, subnet.GetID(), pool); err != nil {
		return err
	} else if conflictPool != nil {
		return fmt.Errorf("pool %s conflict with pool %s",
			pool.String(), conflictPool.String())
	} else {
		return nil
	}
}

func checkSubnet6IfCanCreateDynamicPool(subnet *resource.Subnet6) error {
	if subnet.UseEui64 {
		return fmt.Errorf("subnet use EUI64, can not create dynamic pool")
	}

	if ones, _ := subnet.Ipnet.Mask.Size(); ones < 64 {
		return fmt.Errorf(
			"only can create dynamic pool when subnet mask >= 64, current mask is %d",
			ones)
	}

	return nil
}

func getConflictPool6InSubnet6(tx restdb.Transaction, subnetID string, pool *resource.Pool6) (*resource.Pool6, error) {
	var pools []*resource.Pool6
	if err := tx.FillEx(&pools,
		"select * from gr_pool6 where subnet6 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetID, pool.EndIp, pool.BeginIp); err != nil {
		return nil, fmt.Errorf("get pools with subnet %s from db failed: %s",
			subnetID, err.Error())
	}

	if len(pools) != 0 {
		return pools[0], nil
	}

	return nil, nil
}

func recalculatePool6Capacity(tx restdb.Transaction, subnetID string, pool *resource.Pool6) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetID},
		&reservations); err != nil {
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
	if err := tx.FillEx(&reservedpools,
		"select * from gr_reserved_pool6 where subnet6 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetID, pool.EndIp, pool.BeginIp); err != nil {
		return err
	}

	for _, reservedpool := range reservedpools {
		pool.Capacity -= getPool6ReservedCountWithReservedPool6(pool, reservedpool)
	}

	return nil
}

func sendCreatePool6CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.Pool6) error {
	nodesForSucceed, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreatePool6,
		pool6ToCreatePool6Request(subnetID, pool))
	if err != nil {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, kafka.DeletePool6,
			pool6ToDeletePool6Request(subnetID, pool)); err != nil {
			log.Errorf("create subnet %d pool6 %s failed, and rollback it failed: %s",
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

func (p *Pool6Service) List(subnet *resource.Subnet6) (interface{}, error) {
	return GetPool6List(subnet)
}

func GetPool6List(subnet *resource.Subnet6) ([]*resource.Pool6, error) {
	var pools []*resource.Pool6
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet6: subnet.GetID(),
			util.SqlOrderBy:           resource.SqlColumnBeginIp}, &pools); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet.GetID()}, &reservations)
	}); err != nil {
		return nil, err
	}

	poolsLeases := loadPool6sLeases(subnet, pools, reservations)
	for _, pool := range pools {
		setPool6LeasesUsedRatio(pool, poolsLeases[pool.GetID()])
	}

	return pools, nil
}

func loadPool6sLeases(subnet *resource.Subnet6, pools []*resource.Pool6, reservations []*resource.Reservation6) map[string]uint64 {
	resp, err := getSubnet6Leases(subnet.SubnetId)
	if err != nil {
		log.Warnf("get subnet %s leases failed: %s", subnet.GetID(), err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := reservationMapFromReservation6s(reservations)
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

func (p *Pool6Service) Get(subnetID, poolID string) (restresource.Resource, error) {
	var pools []*resource.Pool6
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{restdb.IDField: poolID},
			&pools); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetID}, &reservations)
	}); err != nil {
		return nil, err
	}

	if len(pools) != 1 {
		return nil, fmt.Errorf("no found pool %s with subnet %s", poolID, subnetID)
	}

	leasesCount, err := getPool6LeasesCount(pools[0], reservations)
	if err != nil {
		log.Warnf("get pool %s with subnet %s from db failed: %s",
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

	reservationMap := reservationMapFromReservation6s(reservations)
	var leasesCount uint64
	for _, lease := range resp.GetLeases() {
		if _, ok := reservationMap[lease.GetAddress()]; ok == false {
			leasesCount += 1
		}
	}

	return leasesCount, nil
}

func (p *Pool6Service) Delete(subnet *resource.Subnet6, pool *resource.Pool6) error {
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setPool6FromDB(tx, pool); err != nil {
			return err
		}

		if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet.GetID()},
			&reservations); err != nil {
			return err
		}

		if leasesCount, err := getPool6LeasesCount(pool, reservations); err != nil {
			return fmt.Errorf("get pool %s leases count failed: %s",
				pool.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete pool with %d ips had been allocated",
				leasesCount)
		}

		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			resource.SqlColumnCapacity: subnet.Capacity - pool.Capacity,
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
		return err
	}

	return nil
}

func setPool6FromDB(tx restdb.Transaction, pool *resource.Pool6) error {
	var pools []*resource.Pool6
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

func sendDeletePool6CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.Pool6) error {
	_, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeletePool6,
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

func (p *Pool6Service) Update(pool *resource.Pool6) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePool6, map[string]interface{}{
			util.SqlColumnsComment: pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found pool6 %s", pool.GetID())
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return pool, nil
}

func (p *Pool6Service) ActionValidTemplate(ctx *restresource.Context) (interface{}, error) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pool := ctx.Resource.(*resource.Pool6)
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if ok == false {
		return nil, fmt.Errorf("parse action refresh input invalid")
	}

	pool.Template = templateInfo.Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkPool6CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, err
	}

	return &resource.TemplatePool{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress}, nil
}
