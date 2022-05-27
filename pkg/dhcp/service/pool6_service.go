package service

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type Pool6Service struct {
}

func NewPool6Service() *Pool6Service {
	return &Pool6Service{}
}

func (p *Pool6Service) Create(subnet *resource.Subnet6, pool *resource.Pool6) error {
	if err := pool.Validate(); err != nil {
		return fmt.Errorf("validate pool6 params invalid: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool6CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := recalculatePool6Capacity(tx, subnet.GetID(), pool); err != nil {
			return fmt.Errorf("recalculate pool6 capacity failed: %s", err.Error())
		}

		if err := updateSubnet6CapacityWithPool6(tx, subnet.GetID(),
			subnet.AddCapacityWithString(pool.Capacity)); err != nil {
			return err
		}

		pool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return pg.Error(err)
		}

		return sendCreatePool6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return fmt.Errorf("create pool6 %s with subnet6 %s failed: %s",
			pool.String(), subnet.GetID(), err.Error())
	}

	return nil
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
			subnetID, pg.Error(err).Error())
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
	return reservations, pg.Error(err)
}

func recalculatePool6CapacityWithReservations(pool *resource.Pool6, reservations []*resource.Reservation6) {
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			if pool.Contains(ipAddress) {
				pool.SubCapacityWithBigInt(big.NewInt(1))
			}
		}
	}
}

func recalculatePool6CapacityWithReservedPools(pool *resource.Pool6, reservedPools []*resource.ReservedPool6) {
	for _, reservedPool := range reservedPools {
		if reservedCount := getPool6ReservedCountWithReservedPool6(pool, reservedPool); reservedCount != nil {
			pool.SubCapacityWithBigInt(reservedCount)
		}
	}
}

func updateSubnet6CapacityWithPool6(tx restdb.Transaction, subnetID string, capacity string) error {
	if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
		resource.SqlColumnCapacity: capacity,
	}, map[string]interface{}{restdb.IDField: subnetID}); err != nil {
		return fmt.Errorf("update subnet6 %s capacity to db failed: %s",
			subnetID, pg.Error(err).Error())
	} else {
		return nil
	}
}

func sendCreatePool6CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.Pool6) error {
	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreatePool6,
		pool6ToCreatePool6Request(subnetID, pool), func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeletePool6,
				pool6ToDeletePool6Request(subnetID, pool)); err != nil {
				log.Errorf("create subnet6 %d pool6 %s failed, rollback with nodes %v failed: %s",
					subnetID, pool.String(), nodesForSucceed, err.Error())
			}
		})
}

func pool6ToCreatePool6Request(subnetID uint64, pool *resource.Pool6) *pbdhcpagent.CreatePool6Request {
	return &pbdhcpagent.CreatePool6Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *Pool6Service) List(subnetId string) ([]*resource.Pool6, error) {
	return listPool6s(subnetId)
}

func listPool6s(subnetId string) ([]*resource.Pool6, error) {
	var pools []*resource.Pool6
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet6: subnetId,
			resource.SqlOrderBy:       resource.SqlColumnBeginIp}, &pools)
		if err != nil {
			return err
		}

		reservations, err = getReservation6sWithIpsExists(tx, subnetId)
		return err
	}); err != nil {
		return nil, fmt.Errorf("list pool6s with subnet6 %s from db failed: %s",
			subnetId, pg.Error(err).Error())
	}

	poolsLeases := loadPool6sLeases(subnetId, pools, reservations)
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
			if resource.IsCapacityZero(pool.Capacity) == false && pool.Contains(lease.GetAddress()) {
				leasesCount[pool.GetID()] += 1
				break
			}
		}
	}

	return leasesCount
}

func getSubnet6Leases(subnetId uint64) (*pbdhcpagent.GetLeases6Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Leases(ctx,
		&pbdhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
}

func setPool6LeasesUsedRatio(pool *resource.Pool6, leasesCount uint64) {
	if resource.IsCapacityZero(pool.Capacity) == false && leasesCount != 0 {
		pool.UsedCount = leasesCount
		pool.UsedRatio = fmt.Sprintf("%.4f", calculateUsedRatio(pool.Capacity, leasesCount))
	}
}

func (p *Pool6Service) Get(subnet *resource.Subnet6, poolID string) (*resource.Pool6, error) {
	var pools []*resource.Pool6
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools)
		if err != nil {
			return pg.Error(err)
		} else if len(pools) != 1 {
			return fmt.Errorf("no found pool6 %s with subnet6 %s", poolID, subnet.GetID())
		}

		reservations, err = getReservation6sWithIpsExists(tx, subnet.GetID())
		return err
	}); err != nil {
		return nil, fmt.Errorf("get pool6 %s with subnet6 %s from db failed: %s",
			poolID, subnet.GetID(), err.Error())
	}

	leasesCount, err := getPool6LeasesCount(pools[0], reservations)
	if err != nil {
		log.Warnf("get pool6 %s with subnet6 %s from db failed: %s",
			poolID, subnet.GetID(), err.Error())
	}

	setPool6LeasesUsedRatio(pools[0], leasesCount)
	return pools[0], nil
}

func getPool6LeasesCount(pool *resource.Pool6, reservations []*resource.Reservation6) (uint64, error) {
	if resource.IsCapacityZero(pool.Capacity) {
		return 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetPool6Leases(ctx,
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

func (p *Pool6Service) Delete(subnet *resource.Subnet6, pool *resource.Pool6) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool6CouldBeDeleted(tx, subnet, pool); err != nil {
			return err
		}

		if err := updateSubnet6CapacityWithPool6(tx, subnet.GetID(),
			subnet.SubCapacityWithString(pool.Capacity)); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TablePool6, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return pg.Error(err)
		}

		return sendDeletePool6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return fmt.Errorf("delete pool6 %s with subnet6 %s failed: %s",
			pool.String(), subnet.GetID(), err.Error())
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
		return fmt.Errorf("get pool6 from db failed: %s", pg.Error(err).Error())
	} else if len(pools) == 0 {
		return fmt.Errorf("no found pool6 %s", pool.GetID())
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
	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeletePool6,
		pool6ToDeletePool6Request(subnetID, pool), nil)
}

func pool6ToDeletePool6Request(subnetID uint64, pool *resource.Pool6) *pbdhcpagent.DeletePool6Request {
	return &pbdhcpagent.DeletePool6Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *Pool6Service) Update(subnetId string, pool *resource.Pool6) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePool6, map[string]interface{}{
			resource.SqlColumnComment: pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found pool6 %s", pool.GetID())
		}
		return nil
	}); err != nil {
		return fmt.Errorf("update pool6 %s with subnet6 %s failed: %s",
			pool.String(), subnetId, err.Error())
	}

	return nil
}

func (p *Pool6Service) ActionValidTemplate(subnet *resource.Subnet6, pool *resource.Pool6, templateInfo *resource.TemplateInfo) (*resource.TemplatePool, error) {
	pool.Template = templateInfo.Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkPool6CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, fmt.Errorf("template6 %s invalid: %s", pool.Template, err.Error())
	}

	return &resource.TemplatePool{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress}, nil
}

func GetPool6sByPrefix(prefix string) ([]*resource.Pool6, error) {
	subnet6, err := GetSubnet6ByPrefix(prefix)
	if err != nil {
		return nil, err
	}

	if pools, err := listPool6s(subnet6.GetID()); err != nil {
		return nil, err
	} else {
		return pools, nil
	}
}
