package service

import (
	"context"
	"fmt"
	"net"

	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Pool4Service struct {
}

func NewPool4Service() *Pool4Service {
	return &Pool4Service{}
}

func (p *Pool4Service) Create(subnet *resource.Subnet4, pool *resource.Pool4) error {
	if err := pool.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool4CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := recalculatePool4Capacity(tx, subnet.GetID(), pool); err != nil {
			return err
		}

		pool.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return util.FormatDbInsertError(errorno.ErrNameDhcpPool, pool.GetID(), err)
		}

		if pool.Capacity != 0 {
			if err := updateResourceCapacity(tx, resource.TableSubnet4, subnet.GetID(),
				subnet.Capacity+pool.Capacity, errorno.ErrNameNetworkV4); err != nil {
				return err
			}
		}

		return sendCreatePool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	})
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

	if !checkIPsBelongsToIpnet(subnet.Ipnet, pool.BeginIp, pool.EndIp) {
		return errorno.ErrNotBelongTo(errorno.ErrNameDhcpPool, errorno.ErrNameNetworkV4,
			pool.String(), subnet.Subnet)
	}

	if conflictPools, err := getPool4sWithBeginAndEndIp(tx, subnet.GetID(),
		pool.BeginIp, pool.EndIp); err != nil {
		return err
	} else if len(conflictPools) != 0 {
		return errorno.ErrConflict(errorno.ErrNameDhcpPool, errorno.ErrNameDhcpPool,
			pool.String(), conflictPools[0].String())
	}

	return nil
}

func checkIPsBelongsToIpnet(ipnet net.IPNet, ips ...net.IP) bool {
	for _, ip := range ips {
		if !ipnet.Contains(ip) {
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
	} else {
		return pools, nil
	}
}

func recalculatePool4Capacity(tx restdb.Transaction, subnetID string, pool *resource.Pool4) error {
	if count, err := tx.CountEx(resource.TableReservation4,
		"select count(*) from gr_reservation4 where subnet4 = $1 and ip >= $2 and ip <= $3",
		subnetID, pool.BeginIp, pool.EndIp); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	} else {
		pool.Capacity -= uint64(count)
	}

	if pool.Capacity == 0 {
		return nil
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

func sendCreatePool4CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.Pool4) error {
	if len(nodes) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.CreatePool4,
		pool4ToCreatePool4Request(subnetID, pool), func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeletePool4,
				pool4ToDeletePool4Request(subnetID, pool)); err != nil {
				log.Errorf("create subnet4 %d pool4 %s failed, rollback nodes %v failed: %s",
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

func pool4ToDeletePool4Request(subnetID uint64, pool *resource.Pool4) *pbdhcpagent.DeletePool4Request {
	return &pbdhcpagent.DeletePool4Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *Pool4Service) List(subnet *resource.Subnet4) ([]*resource.Pool4, error) {
	return listPool4s(subnet, ListResourceModeAPI)
}

func listPool4s(subnet *resource.Subnet4, mode ListResourceMode) ([]*resource.Pool4, error) {
	var pools []*resource.Pool4
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		if mode == ListResourceModeAPI {
			if err = setSubnet4FromDB(tx, subnet); err != nil {
				return
			}
		}

		if pools, err = getPool4sWithCondition(tx, map[string]interface{}{
			resource.SqlColumnSubnet4: subnet.GetID(),
			resource.SqlOrderBy:       resource.SqlColumnBeginIp}); err != nil {
			return
		}

		if len(subnet.Nodes) == 0 {
			return
		}

		if err = tx.FillEx(&reservations, `
		select * from gr_reservation4 where id in
			(select distinct r4.id from gr_reservation4 r4, gr_pool4 p4 where
				r4.subnet4 = $1 and
				r4.subnet4 = p4.subnet4 and
				r4.ip >= p4.begin_ip and
				r4.ip <= p4.end_ip
			)`, subnet.GetID()); err != nil {
			err = errorno.ErrDBError(errorno.ErrDBNameQuery,
				string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}

		return
	}); err != nil {
		return nil, err
	}

	if len(subnet.Nodes) != 0 {
		poolsLeases := loadPool4sLeases(subnet.SubnetId, pools, reservations)
		for _, pool := range pools {
			setPool4LeasesUsedRatio(pool, poolsLeases[pool.GetID()])
		}
	}

	return pools, nil
}

func getPool4sWithCondition(tx restdb.Transaction, condition map[string]interface{}) ([]*resource.Pool4, error) {
	var pools []*resource.Pool4
	if err := tx.Fill(condition, &pools); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
	}

	return pools, nil
}

func loadPool4sLeases(subnetID uint64, pools []*resource.Pool4, reservations []*resource.Reservation4) map[string]uint64 {
	resp, err := getSubnet4Leases(subnetID)
	if err != nil {
		log.Warnf("get subnet4 %d leases failed: %s", subnetID, err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := reservationMapFromReservation4s(reservations)
	leasesCount := make(map[string]uint64, len(pools))
	for _, lease := range resp.GetLeases() {
		if _, ok := reservationMap[lease.GetAddress()]; ok {
			continue
		}

		for _, pool := range pools {
			if pool.Capacity != 0 && pool.ContainsIpstr(lease.GetAddress()) {
				leasesCount[pool.GetID()] += 1
				break
			}
		}
	}

	return leasesCount
}

func getSubnet4Leases(subnetId uint64) (response *pbdhcpagent.GetLeases4Response, err error) {
	transport.CallDhcpAgentGrpc4(func(ctx context.Context,
		client pbdhcpagent.DHCPManagerClient) error {
		response, err = client.GetSubnet4Leases(ctx,
			&pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
		return err
	})
	return
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
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		if err = setSubnet4FromDB(tx, subnet); err != nil {
			return
		}

		if pools, err = getPool4sWithCondition(tx, map[string]interface{}{
			restdb.IDField: poolID}); err != nil {
			return err
		} else if len(pools) != 1 {
			return errorno.ErrNotFound(errorno.ErrNameDhcpPool, poolID)
		}

		if len(subnet.Nodes) != 0 {
			reservations, err = getReservation4sWithBeginAndEndIp(tx, subnet.GetID(),
				pools[0].BeginIp, pools[0].EndIp)
		}

		return
	}); err != nil {
		return nil, err
	}

	leasesCount, err := getPool4LeasesCount(subnet, pools[0], reservations)
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	} else {
		return reservations, nil
	}
}

func getPool4LeasesCount(subnet *resource.Subnet4, pool *resource.Pool4, reservations []*resource.Reservation4) (uint64, error) {
	if pool.Capacity == 0 || len(subnet.Nodes) == 0 {
		return 0, nil
	}

	var resp *pbdhcpagent.GetLeases4Response
	var err error
	if err = transport.CallDhcpAgentGrpc4(func(ctx context.Context,
		client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetPool4Leases(ctx,
			&pbdhcpagent.GetPool4LeasesRequest{
				SubnetId:     subnet.SubnetId,
				BeginAddress: pool.BeginAddress,
				EndAddress:   pool.EndAddress})
		return err
	}); err != nil {
		return 0, errorno.ErrNetworkError(errorno.ErrNameLease, err.Error())
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
		if _, ok := reservationMap[lease.GetAddress()]; !ok {
			leasesCount += 1
		}
	}

	return leasesCount, nil
}

func (p *Pool4Service) Delete(subnet *resource.Subnet4, pool *resource.Pool4) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPool4CouldBeDeleted(tx, subnet, pool); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TablePool4, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, pool.GetID(),
				pg.Error(err).Error())
		}

		if pool.Capacity != 0 {
			if err := updateResourceCapacity(tx, resource.TableSubnet4, subnet.GetID(),
				subnet.Capacity-pool.Capacity, errorno.ErrNameNetworkV4); err != nil {
				return err
			}
		}

		return sendDeletePool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	})
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

	if leasesCount, err := getPool4LeasesCount(subnet, pool, reservations); err != nil {
		return err
	} else if leasesCount != 0 {
		return errorno.ErrIPHasBeenAllocated(errorno.ErrNameDhcpPool, "")
	}

	return nil
}

func setPool4FromDB(tx restdb.Transaction, pool *resource.Pool4) error {
	pools, err := getPool4sWithCondition(tx,
		map[string]interface{}{restdb.IDField: pool.GetID()})
	if err != nil {
		return err
	} else if len(pools) == 0 {
		return errorno.ErrNotFound(errorno.ErrNameDhcpPool, pool.GetID())
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
	if len(nodes) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.DeletePool4,
		pool4ToDeletePool4Request(subnetID, pool), nil)
}

func (p *Pool4Service) Update(subnetId string, pool *resource.Pool4) error {
	if err := util.ValidateStrings(util.RegexpTypeComma, pool.Comment); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePool4, map[string]interface{}{
			resource.SqlColumnComment: pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, pool.GetID(),
				pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameDhcpPool, pool.GetID())
		}

		return nil
	})
}

func (p *Pool4Service) ActionValidTemplate(subnet *resource.Subnet4, pool *resource.Pool4, templateInfo *resource.TemplateInfo) (*resource.TemplatePool, error) {
	pool.Template = templateInfo.Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkPool4CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, err
	}

	return &resource.TemplatePool{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress}, nil
}

func GetPool4sByPrefix(prefix string) ([]*resource.Pool4, error) {
	if subnet4, err := GetSubnet4ByPrefix(prefix); err != nil {
		return nil, err
	} else {
		return listPool4s(subnet4, ListResourceModeGRPC)
	}
}
