package service

import (
	"context"
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
	"net"
	"strconv"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type Pool4Service struct {
}

func NewPool4Service() *Pool4Service {
	return &Pool4Service{}
}

func (p *Pool4Service) Create(subnet *resource.Subnet4, pool *resource.Pool4) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool4CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := recalculatePool4Capacity(tx, subnet.GetID(), pool); err != nil {
			return fmt.Errorf("recalculate pool capacity failed: %s", err.Error())
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			resource.SqlColumnCapacity: subnet.Capacity + pool.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s",
				subnet.GetID(), err.Error())
		}

		pool.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return err
		}

		return sendCreatePool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return nil, err
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

	if checkIPsBelongsToIpnet(subnet.Ipnet, pool.BeginIp, pool.EndIp) == false {
		return fmt.Errorf("pool %s not belongs to subnet %s",
			pool.String(), subnet.Subnet)
	}

	if conflictPool, err := getConflictPool4InSubnet4(tx, subnet.GetID(),
		pool); err != nil {
		return err
	} else if conflictPool != nil {
		return fmt.Errorf("pool %s conflict with pool %s",
			pool.String(), conflictPool.String())
	}

	return nil
}

func getConflictPool4InSubnet4(tx restdb.Transaction, subnetID string, pool *resource.Pool4) (*resource.Pool4, error) {
	var pools []*resource.Pool4
	if err := tx.FillEx(&pools,
		"select * from gr_pool4 where subnet4 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetID, pool.EndIp, pool.BeginIp); err != nil {
		return nil, fmt.Errorf("get pools with subnet %s from db failed: %s",
			subnetID, err.Error())
	}

	if len(pools) != 0 {
		return pools[0], nil
	}

	return nil, nil
}

func checkIPsBelongsToIpnet(ipnet net.IPNet, ips ...net.IP) bool {
	for _, ip := range ips {
		if ipnet.Contains(ip) == false {
			return false
		}
	}

	return true
}

func recalculatePool4Capacity(tx restdb.Transaction, subnetID string, pool *resource.Pool4) error {
	if count, err := tx.CountEx(resource.TableReservation4,
		"select count(*) from gr_reservation4 where subnet4 = $1 and ip >= $2 and ip <= $3",
		subnetID, pool.BeginIp, pool.EndIp); err != nil {
		return fmt.Errorf("get reservation4 from db failed: %s", err.Error())
	} else {
		pool.Capacity -= uint64(count)
	}

	var reservedpools []*resource.ReservedPool4
	if err := tx.FillEx(&reservedpools,
		"select * from gr_reserved_pool4 where subnet4 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetID, pool.EndIp, pool.BeginIp); err != nil {
		return fmt.Errorf("get reserved pool4 from db failed: %s", err.Error())
	}

	for _, reservedpool := range reservedpools {
		pool.Capacity -= getPool4ReservedCountWithReservedPool4(pool, reservedpool)
	}

	return nil
}

func sendCreatePool4CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.Pool4) error {
	nodesForSucceed, err := kafka.SendDHCPCmdWithNodes(true, nodes, kafka.CreatePool4,
		pool4ToCreatePool4Request(subnetID, pool))
	if err != nil {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, kafka.DeletePool4,
			pool4ToDeletePool4Request(subnetID, pool)); err != nil {
			log.Errorf("create subnet %d pool4 %s failed, and rollback it failed: %s",
				subnetID, pool.String(), err.Error())
		}
	}

	return err
}

func pool4ToCreatePool4Request(subnetID uint64, pool *resource.Pool4) *pbdhcpagent.CreatePool4Request {
	return &pbdhcpagent.CreatePool4Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *Pool4Service) List(subnet *resource.Subnet4) (interface{}, error) {
	return GetPool4List(subnet)
}

func GetPool4List(subnet *resource.Subnet4) ([]*resource.Pool4, error) {
	var pools []*resource.Pool4
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet4: subnet.GetID(),
			util.SqlOrderBy:           resource.SqlColumnBeginIp}, &pools); err != nil {
			return err
		}

		return tx.FillEx(&reservations,
			"select * from gr_reservation4 where id in (select distinct r4.id from gr_reservation4 r4, "+
				"gr_pool4 p4 where r4.subnet4 = $1 and r4.subnet4 = p4.subnet4 and "+
				"r4.ip_address >= p4.begin_address and r4.ip_address <= p4.end_address)",
			subnet.GetID())
	}); err != nil {
		return nil, err
	}

	poolsLeases := loadPool4sLeases(subnet, pools, reservations)
	for _, pool := range pools {
		setPool4LeasesUsedRatio(pool, poolsLeases[pool.GetID()])
	}

	return pools, nil
}

func loadPool4sLeases(subnet *resource.Subnet4, pools []*resource.Pool4, reservations []*resource.Reservation4) map[string]uint64 {
	resp, err := getSubnet4Leases(subnet.SubnetId)
	if err != nil {
		log.Warnf("get subnet %s leases failed: %s", subnet.GetID(), err.Error())
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
				count := leasesCount[pool.GetID()]
				count += 1
				leasesCount[pool.GetID()] = count
			}
		}
	}

	return leasesCount
}

func getSubnet4Leases(subnetId uint64) (*pbdhcpagent.GetLeases4Response, error) {
	return grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Leases(context.TODO(),
		&pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
}

func setPool4LeasesUsedRatio(pool *resource.Pool4, leasesCount uint64) {
	if leasesCount != 0 && pool.Capacity != 0 {
		pool.UsedCount = leasesCount
		pool.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(pool.Capacity))
	}
}

func (p *Pool4Service) Get(subnetID, poolID string) (restresource.Resource, error) {
	var pools []*resource.Pool4
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{restdb.IDField: poolID},
			&pools); err != nil {
			return err
		}

		if len(pools) != 1 {
			return fmt.Errorf("no found pool %s with subnet %s", poolID, subnetID)
		}

		return tx.FillEx(&reservations,
			"select * from gr_reservation4 where subnet4 = $1 and ip >= $2 and ip <= $3",
			subnetID, pools[0].BeginIp, pools[0].EndIp)
	}); err != nil {
		return nil, err
	}

	leasesCount, err := getPool4LeasesCount(pools[0], reservations)
	if err != nil {
		log.Warnf("get pool %s with subnet %s from db failed: %s",
			poolID, subnetID, err.Error())
	}

	setPool4LeasesUsedRatio(pools[0], leasesCount)
	return pools[0], nil
}

func getPool4LeasesCount(pool *resource.Pool4, reservations []*resource.Reservation4) (uint64, error) {
	if pool.Capacity == 0 {
		return 0, nil
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetPool4Leases(context.TODO(),
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
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setPool4FromDB(tx, pool); err != nil {
			return err
		}

		if err := tx.FillEx(&reservations,
			"select * from gr_reservation4 where subnet4 = $1 and ip >= $2 and ip <= $3",
			subnet.GetID(), pool.BeginIp, pool.EndIp); err != nil {
			return err
		}

		if leasesCount, err := getPool4LeasesCount(pool, reservations); err != nil {
			return fmt.Errorf("get pool %s leases count failed: %s",
				pool.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete pool with %d ips had been allocated",
				leasesCount)
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			resource.SqlColumnCapacity: subnet.Capacity - pool.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s",
				subnet.GetID(), err.Error())
		}

		if _, err := tx.Delete(resource.TablePool4, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendDeletePool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return err
	}

	return nil
}

func setPool4FromDB(tx restdb.Transaction, pool *resource.Pool4) error {
	var pools []*resource.Pool4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pool.GetID()},
		&pools); err != nil {
		return fmt.Errorf("get pool from db failed: %s", err.Error())
	}

	if len(pools) == 0 {
		return fmt.Errorf("no found pool %s", pool.GetID())
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
	_, err := kafka.SendDHCPCmdWithNodes(true, nodes, kafka.DeletePool4,
		pool4ToDeletePool4Request(subnetID, pool))
	return err
}

func pool4ToDeletePool4Request(subnetID uint64, pool *resource.Pool4) *pbdhcpagent.DeletePool4Request {
	return &pbdhcpagent.DeletePool4Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *Pool4Service) Update(pool *resource.Pool4) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePool4, map[string]interface{}{
			util.SqlColumnsComment: pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found pool4 %s", pool.GetID())
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return pool, nil
}

func (p *Pool4Service) ActionValidTemplate(ctx *restresource.Context) (interface{}, error) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.Pool4)
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if ok == false {
		return nil, fmt.Errorf("parse action refresh input invalid")
	}

	pool.Template = templateInfo.Template

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkPool4CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, fmt.Errorf("template %s invalid: %s", pool.Template, err.Error())
	}

	return &resource.TemplatePool{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress}, nil
}
