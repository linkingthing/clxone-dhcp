package service

import (
	"context"
	"fmt"
	"net"
	"strconv"
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

type Pool4Service struct {
}

func NewPool4Service() *Pool4Service {
	return &Pool4Service{}
}

func (p *Pool4Service) Create(subnet *resource.Subnet4, pool *resource.Pool4) error {
	if err := pool.Validate(); err != nil {
		return fmt.Errorf("validate pool4 params invalid: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool4CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := recalculatePool4Capacity(tx, subnet.GetID(), pool); err != nil {
			return fmt.Errorf("recalculate pool4 capacity failed: %s", err.Error())
		}

		if err := updateSubnet4CapacityWithPool4(tx, subnet.GetID(),
			subnet.Capacity+pool.Capacity); err != nil {
			return err
		}

		pool.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return pg.Error(err)
		}

		return sendCreatePool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return fmt.Errorf("create pool4 %s with subnet4 %s failed: %s",
			pool.String(), subnet.GetID(), err.Error())
	}

	return nil
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

	if checkIPsBelongsToIpnet(subnet.Ipnet, pool.BeginIp, pool.EndIp) == false {
		return fmt.Errorf("pool4 %s not belongs to subnet4 %s",
			pool.String(), subnet.Subnet)
	}

	if conflictPools, err := getPool4sWithBeginAndEndIp(tx, subnet.GetID(),
		pool.BeginIp, pool.EndIp); err != nil {
		return err
	} else if len(conflictPools) != 0 {
		return fmt.Errorf("pool4 %s conflict with pool4 %s",
			pool.String(), conflictPools[0].String())
	}

	return nil
}

func checkIPsBelongsToIpnet(ipnet net.IPNet, ips ...net.IP) bool {
	for _, ip := range ips {
		if ipnet.Contains(ip) == false {
			return false
		}
	}

	return true
}

func getPool4sWithBeginAndEndIp(tx restdb.Transaction, subnetID string, begin, end net.IP) ([]*resource.Pool4, error) {
	var pools []*resource.Pool4
	if err := tx.FillEx(&pools,
		"select * from gr_pool4 where subnet4 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetID, end, begin); err != nil {
		return nil, fmt.Errorf("get pool4s with subnet4 %s from db failed: %s",
			subnetID, pg.Error(err).Error())
	} else {
		return pools, nil
	}
}

func recalculatePool4Capacity(tx restdb.Transaction, subnetID string, pool *resource.Pool4) error {
	if count, err := tx.CountEx(resource.TableReservation4,
		"select count(*) from gr_reservation4 where subnet4 = $1 and ip >= $2 and ip <= $3",
		subnetID, pool.BeginIp, pool.EndIp); err != nil {
		return fmt.Errorf("get reservation4 from db failed: %s", pg.Error(err).Error())
	} else {
		pool.Capacity -= uint64(count)
	}

	reservedpools, err := getReservedPool4sWithBeginAndEndIp(tx, subnetID,
		pool.BeginIp, pool.EndIp)
	if err != nil {
		return err
	}

	for _, reservedpool := range reservedpools {
		pool.Capacity -= getPool4ReservedCountWithReservedPool4(pool, reservedpool)
	}

	return nil
}

func updateSubnet4CapacityWithPool4(tx restdb.Transaction, subnetID string, capacity uint64) error {
	if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
		resource.SqlColumnCapacity: capacity,
	}, map[string]interface{}{restdb.IDField: subnetID}); err != nil {
		return fmt.Errorf("update subnet4 %s capacity to db failed: %s",
			subnetID, pg.Error(err).Error())
	} else {
		return nil
	}
}

func sendCreatePool4CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.Pool4) error {
	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.CreatePool4,
		pool4ToCreatePool4Request(subnetID, pool), func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeletePool4,
				pool4ToDeletePool4Request(subnetID, pool)); err != nil {
				log.Errorf("create subnet4 %d pool4 %s failed, rollback with nodes %v failed: %s",
					subnetID, pool.String(), nodesForSucceed, err.Error())
			}
		})
}

func pool4ToCreatePool4Request(subnetID uint64, pool *resource.Pool4) *pbdhcpagent.CreatePool4Request {
	return &pbdhcpagent.CreatePool4Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *Pool4Service) List(subnetID string) ([]*resource.Pool4, error) {
	return listPool4s(subnetID)
}

func listPool4s(subnetID string) ([]*resource.Pool4, error) {
	var pools []*resource.Pool4
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet4: subnetID,
			resource.SqlOrderBy:       resource.SqlColumnBeginIp}, &pools); err != nil {
			return err
		}

		return tx.FillEx(&reservations, `
		select * from gr_reservation4 where id in 
			(select distinct r4.id from gr_reservation4 r4, gr_pool4 p4 where 
				r4.subnet4 = $1 and 
				r4.subnet4 = p4.subnet4 and 
				r4.ip_address >= p4.begin_address and 
				r4.ip_address <= p4.end_address
			)`, subnetID)
	}); err != nil {
		return nil, fmt.Errorf("list pool4s with subnet4 %s from db failed: %s",
			subnetID, pg.Error(err).Error())
	}

	poolsLeases := loadPool4sLeases(subnetID, pools, reservations)
	for _, pool := range pools {
		setPool4LeasesUsedRatio(pool, poolsLeases[pool.GetID()])
	}

	return pools, nil
}

func loadPool4sLeases(subnetID string, pools []*resource.Pool4, reservations []*resource.Reservation4) map[string]uint64 {
	resp, err := getSubnet4Leases(subnetIDStrToUint64(subnetID))
	if err != nil {
		log.Warnf("get subnet4 %s leases failed: %s", subnetID, err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := reservationMapFromReservation4s(reservations)
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

func getSubnet4Leases(subnetId uint64) (*pbdhcpagent.GetLeases4Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Leases(ctx,
		&pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
}

func setPool4LeasesUsedRatio(pool *resource.Pool4, leasesCount uint64) {
	if leasesCount != 0 && pool.Capacity != 0 {
		pool.UsedCount = leasesCount
		pool.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(pool.Capacity))
	}
}

func (p *Pool4Service) Get(subnet *resource.Subnet4, poolID string) (*resource.Pool4, error) {
	var pools []*resource.Pool4
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools)
		if err != nil {
			return pg.Error(err)
		} else if len(pools) != 1 {
			return fmt.Errorf("no found pool4 %s with subnet4 %s", poolID, subnet.GetID())
		}

		reservations, err = getReservation4sWithBeginAndEndIp(tx, subnet.GetID(), pools[0].BeginIp, pools[0].EndIp)
		return err
	}); err != nil {
		return nil, fmt.Errorf("get pool4 %s with subnet4 %s failed: %s",
			poolID, subnet.GetID(), err.Error())
	}

	leasesCount, err := getPool4LeasesCount(pools[0], reservations)
	if err != nil {
		log.Warnf("get pool4 %s with subnet4 %s from db failed: %s",
			poolID, subnet.GetID(), err.Error())
	}

	setPool4LeasesUsedRatio(pools[0], leasesCount)
	return pools[0], nil
}

func getReservation4sWithBeginAndEndIp(tx restdb.Transaction, subnetID string, begin, end net.IP) ([]*resource.Reservation4, error) {
	var reservations []*resource.Reservation4
	if err := tx.FillEx(&reservations,
		"select * from gr_reservation4 where subnet4 = $1 and ip >= $2 and ip <= $3",
		subnetID, begin, end); err != nil {
		return nil, fmt.Errorf("get reservation4s with subnet4 %s failed: %s",
			subnetID, pg.Error(err).Error())
	} else {
		return reservations, nil
	}
}

func getPool4LeasesCount(pool *resource.Pool4, reservations []*resource.Reservation4) (uint64, error) {
	if pool.Capacity == 0 {
		return 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetPool4Leases(ctx,
		&pbdhcpagent.GetPool4LeasesRequest{
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

	reservationMap := reservationMapFromReservation4s(reservations)
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

func (p *Pool4Service) Delete(subnet *resource.Subnet4, pool *resource.Pool4) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool4CouldBeDeleted(tx, subnet, pool); err != nil {
			return err
		}

		if err := updateSubnet4CapacityWithPool4(tx, subnet.GetID(),
			subnet.Capacity-pool.Capacity); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TablePool4, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return pg.Error(err)
		}

		return sendDeletePool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return fmt.Errorf("delete pool4 %s with subnet4 %s failed: %s",
			pool.String(), subnet.GetID(), err.Error())
	}

	return nil
}

func checkPool4CouldBeDeleted(tx restdb.Transaction, subnet *resource.Subnet4, pool *resource.Pool4) error {
	if err := setSubnet4FromDB(tx, subnet); err != nil {
		return err
	}

	if err := setPool4FromDB(tx, pool); err != nil {
		return err
	}

	reservations, err := getReservation4sWithBeginAndEndIp(tx, subnet.GetID(),
		pool.BeginIp, pool.EndIp)
	if err != nil {
		return err
	}

	if leasesCount, err := getPool4LeasesCount(pool, reservations); err != nil {
		return fmt.Errorf("get pool4 %s leases count failed: %s",
			pool.String(), err.Error())
	} else if leasesCount != 0 {
		return fmt.Errorf("can not delete pool4 with %d ips had been allocated",
			leasesCount)
	}

	return nil
}

func setPool4FromDB(tx restdb.Transaction, pool *resource.Pool4) error {
	var pools []*resource.Pool4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pool.GetID()},
		&pools); err != nil {
		return fmt.Errorf("get pool4 from db failed: %s", pg.Error(err).Error())
	} else if len(pools) == 0 {
		return fmt.Errorf("no found pool4 %s", pool.GetID())
	}

	pool.Subnet4 = pools[0].Subnet4
	pool.BeginAddress = pools[0].BeginAddress
	pool.BeginIp = pools[0].BeginIp
	pool.EndAddress = pools[0].EndAddress
	pool.EndIp = pools[0].EndIp
	pool.Capacity = pools[0].Capacity
	return nil
}

func sendDeletePool4CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.Pool4) error {
	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.DeletePool4,
		pool4ToDeletePool4Request(subnetID, pool), nil)
}

func pool4ToDeletePool4Request(subnetID uint64, pool *resource.Pool4) *pbdhcpagent.DeletePool4Request {
	return &pbdhcpagent.DeletePool4Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *Pool4Service) Update(subnetId string, pool *resource.Pool4) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePool4, map[string]interface{}{
			resource.SqlColumnComment: pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found pool4 %s", pool.GetID())
		}

		return nil
	}); err != nil {
		return fmt.Errorf("update pool4 %s with subnet4 %s failed: %s",
			pool.String(), subnetId, err.Error())
	}

	return nil
}

func (p *Pool4Service) ActionValidTemplate(subnet *resource.Subnet4, pool *resource.Pool4, templateInfo *resource.TemplateInfo) (*resource.TemplatePool, error) {
	pool.Template = templateInfo.Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkPool4CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, fmt.Errorf("template4 %s invalid: %s", pool.Template, err.Error())
	}

	return &resource.TemplatePool{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress}, nil
}

func GetPool4sByPrefix(prefix string) ([]*resource.Pool4, error) {
	subnet4, err := GetSubnet4ByPrefix(prefix)
	if err != nil {
		return nil, err
	}

	if pools, err := listPool4s(subnet4.GetID()); err != nil {
		return nil, err
	} else {
		return pools, nil
	}
}
