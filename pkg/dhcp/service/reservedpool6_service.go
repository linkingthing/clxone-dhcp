package service

import (
	"math/big"
	"net"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type ReservedPool6Service struct {
}

func NewReservedPool6Service() *ReservedPool6Service {
	return &ReservedPool6Service{}
}

func (p *ReservedPool6Service) Create(subnet *resource.Subnet6, pool *resource.ReservedPool6) error {
	if err := pool.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservedPool6CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := updateSubnet6AndPool6sCapacityWithReservedPool6(tx, subnet,
			pool, true); err != nil {
			return err
		}

		pool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert,
				string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
		}

		return sendCreateReservedPool6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	})
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

	if !checkIPsBelongsToIpnet(subnet.Ipnet, pool.BeginIp, pool.EndIp) {
		return errorno.ErrNotBelongTo(errorno.ErrNameDhcpReservedPool,
			errorno.ErrNameNetworkV6, pool.String(), subnet.Subnet)
	}

	return checkReservedPool6ConflictWithSubnet6Pools(tx, subnet.GetID(), pool)
}

func checkReservedPool6ConflictWithSubnet6Pools(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool6) error {
	if err := checkReservedPool6ConflictWithSubnet6ReservedPool6s(tx, subnetID,
		pool); err != nil {
		return err
	}

	return checkReservedPool6ConflictWithSubnet6Reservation6s(tx, subnetID, pool)
}

func checkReservedPool6ConflictWithSubnet6ReservedPool6s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool6) error {
	if pools, err := getReservedPool6sWithBeginAndEndIp(tx, subnetID,
		pool.BeginIp, pool.EndIp); err != nil {
		return err
	} else if len(pools) != 0 {
		return errorno.ErrConflict(errorno.ErrNameDhcpReservedPool,
			errorno.ErrNameDhcpReservedPool, pool.String(), pools[0].String())
	} else {
		return nil
	}
}

func getReservedPool6sWithBeginAndEndIp(tx restdb.Transaction, subnetID string, begin, end net.IP) ([]*resource.ReservedPool6, error) {
	var reservedpools []*resource.ReservedPool6
	if err := tx.FillEx(&reservedpools,
		`select * from gr_reserved_pool6 where
			subnet6 = $1 and begin_ip <= $2 and end_ip >= $3`,
		subnetID, end, begin); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
	} else {
		return reservedpools, nil
	}
}

func checkReservedPool6ConflictWithSubnet6Reservation6s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool6) error {
	if reservations, err := getReservation6sWithIpsExists(tx, subnetID); err != nil {
		return err
	} else {
		return checkReservedPool6ConflictWithReservation6s(pool, reservations)
	}
}

func checkReservedPool6ConflictWithReservation6s(pool *resource.ReservedPool6, reservations []*resource.Reservation6) error {
	for _, reservation := range reservations {
		for _, ip := range reservation.Ips {
			if pool.ContainsIp(ip) {
				return errorno.ErrConflict(errorno.ErrNameDhcpReservedPool,
					errorno.ErrNameDhcpReservation, pool.String(), reservation.String())
			}
		}
	}

	return nil
}

func updateSubnet6AndPool6sCapacityWithReservedPool6(tx restdb.Transaction, subnet *resource.Subnet6, reservedPool *resource.ReservedPool6, isCreate bool) error {
	pools, err := getPool6sWithBeginAndEndIp(tx, subnet.GetID(),
		reservedPool.BeginIp, reservedPool.EndIp)
	if err != nil {
		return err
	}

	poolsCapacity := make(map[string]string, len(pools))
	recalculateSubnet6AndPool6sCapacityWithReservedPool6(subnet, pools,
		reservedPool, poolsCapacity, isCreate)
	return updateSubnet6AndPoolsCapacity(tx, subnet, poolsCapacity, nil)
}

func recalculateSubnet6AndPool6sCapacityWithReservedPool6(subnet *resource.Subnet6, pools []*resource.Pool6, reservedPool *resource.ReservedPool6, poolsCapacity map[string]string, isCreate bool) {
	allReservedCount := new(big.Int)
	for _, pool := range pools {
		reservedCount := getPool6ReservedCountWithReservedPool6(pool, reservedPool)
		if reservedCount == nil || reservedCount.Sign() == 0 {
			continue
		}

		allReservedCount.Add(allReservedCount, reservedCount)
		if isCreate {
			poolsCapacity[pool.GetID()] = pool.SubCapacityWithBigInt(reservedCount)
		} else {
			poolsCapacity[pool.GetID()] = pool.AddCapacityWithBigInt(reservedCount)
		}
	}

	if isCreate {
		subnet.SubCapacityWithBigInt(allReservedCount)
	} else {
		subnet.AddCapacityWithBigInt(allReservedCount)
	}
}

func getPool6ReservedCountWithReservedPool6(pool *resource.Pool6, reservedPool *resource.ReservedPool6) *big.Int {
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

func batchUpdateResource6sCapacity(tx restdb.Transaction, resourceType restdb.ResourceType, resourcesCapacity map[string]string) error {
	if len(resourcesCapacity) == 0 {
		return nil
	}

	for resourceId, capacity := range resourcesCapacity {
		resourcesCapacity[resourceId] = "('" + resourceId + "'," + capacity + ")"
	}

	if err := util.BatchUpdateById(tx, string(resourceType),
		[]string{resource.SqlColumnCapacity}, resourcesCapacity); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameUpdate,
			string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
	}

	return nil
}

func sendCreateReservedPool6CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.ReservedPool6) error {
	if len(nodes) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreateReservedPool6,
		reservedPool6ToCreateReservedPool6Request(subnetID, pool),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservedPool6,
				reservedPool6ToDeleteReservedPool6Request(subnetID, pool)); err != nil {
				log.Errorf("create subnet6 %d reservedpool6 %s failed, rollback %v failed: %s",
					subnetID, pool.String(), nodesForSucceed, err.Error())
			}
		})
}

func reservedPool6ToCreateReservedPool6Request(subnetID uint64, pool *resource.ReservedPool6) *pbdhcpagent.CreateReservedPool6Request {
	return &pbdhcpagent.CreateReservedPool6Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func reservedPool6ToDeleteReservedPool6Request(subnetID uint64, pool *resource.ReservedPool6) *pbdhcpagent.DeleteReservedPool6Request {
	return &pbdhcpagent.DeleteReservedPool6Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *ReservedPool6Service) List(subnetId string) ([]*resource.ReservedPool6, error) {
	return listReservedPool6s(subnetId)
}

func listReservedPool6s(subnetId string) ([]*resource.ReservedPool6, error) {
	var pools []*resource.ReservedPool6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		pools, err = getReservedPool6sWithCondition(tx, map[string]interface{}{
			resource.SqlColumnSubnet6: subnetId,
			resource.SqlOrderBy:       resource.SqlColumnBeginIp})
		return
	}); err != nil {
		return nil, err
	}

	return pools, nil
}

func getReservedPool6sWithCondition(tx restdb.Transaction, condition map[string]interface{}) ([]*resource.ReservedPool6, error) {
	var pools []*resource.ReservedPool6
	if err := tx.Fill(condition, &pools); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
	}

	return pools, nil
}

func (p *ReservedPool6Service) Get(subnet *resource.Subnet6, poolID string) (*resource.ReservedPool6, error) {
	var pools []*resource.ReservedPool6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, poolID, pg.Error(err).Error())
	} else if len(pools) != 1 {
		return nil, errorno.ErrNotFound(errorno.ErrNameDhcpReservedPool, poolID)
	}

	return pools[0], nil
}

func (p *ReservedPool6Service) Delete(subnet *resource.Subnet6, pool *resource.ReservedPool6) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservedPool6FromDB(tx, pool); err != nil {
			return err
		}

		if err := updateSubnet6AndPool6sCapacityWithReservedPool6(tx, subnet,
			pool, false); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableReservedPool6, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, pool.GetID(),
				pg.Error(err).Error())
		}

		return sendDeleteReservedPool6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	})
}

func setReservedPool6FromDB(tx restdb.Transaction, pool *resource.ReservedPool6) error {
	pools, err := getReservedPool6sWithCondition(tx,
		map[string]interface{}{restdb.IDField: pool.GetID()})
	if err != nil {
		return err
	} else if len(pools) == 0 {
		return errorno.ErrNotFound(errorno.ErrNameDhcpReservedPool, pool.GetID())
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
	if len(nodes) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeleteReservedPool6,
		reservedPool6ToDeleteReservedPool6Request(subnetID, pool), nil)
}

func (p *ReservedPool6Service) ActionValidTemplate(subnet *resource.Subnet6, pool *resource.ReservedPool6, templateInfo *resource.TemplateInfo) (*resource.TemplatePool, error) {
	pool.Template = templateInfo.Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkReservedPool6CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, err
	}

	return &resource.TemplatePool{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress}, nil
}

func (p *ReservedPool6Service) Update(subnetId string, pool *resource.ReservedPool6) error {
	if err := util.ValidateStrings(util.RegexpTypeComma, pool.Comment); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservedPool6,
			map[string]interface{}{resource.SqlColumnComment: pool.Comment},
			map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, pool.GetID(),
				pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameDhcpReservedPool, pool.GetID())
		}

		return nil
	})
}

func GetReservedPool6sByPrefix(prefix string) ([]*resource.ReservedPool6, error) {
	if subnet6, err := GetSubnet6ByPrefix(prefix); err != nil {
		return nil, err
	} else {
		return listReservedPool6s(subnet6.GetID())
	}
}

func BatchCreateReservedPool6s(prefix string, reservedpools []*resource.ReservedPool6) error {
	for _, reservedpool := range reservedpools {
		if err := reservedpool.Validate(); err != nil {
			return err
		}
	}

	for i, reservedpool := range reservedpools {
		if err := checkReservedPool6ConflictWithReservedPool6s(reservedpool,
			reservedpools[i+1:]); err != nil {
			return err
		}
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet, err := getSubnet6WithPrefix(tx, prefix)
		if err != nil {
			return err
		}

		if err := checkSubnet6IfCanCreateDynamicPool(subnet); err != nil {
			return err
		}

		pools, err := getPool6sWithSubnetId(tx, subnet.GetID())
		if err != nil {
			return err
		}

		reservations, err := getReservation6sWithSubnetId(tx, subnet.GetID())
		if err != nil {
			return err
		}

		oldReservedpools, err := getReservedPool6sWithSubnetId(tx, subnet.GetID())
		if err != nil {
			return err
		}

		poolsCapacity := make(map[string]string, len(pools))
		reservedPoolValues := make([][]interface{}, 0, len(reservedpools))
		for _, reservedpool := range reservedpools {
			if !checkIPsBelongsToIpnet(subnet.Ipnet,
				reservedpool.BeginIp, reservedpool.EndIp) {
				return errorno.ErrNotBelongTo(errorno.ErrNameDhcpReservedPool,
					errorno.ErrNameNetworkV6, reservedpool.String(), subnet.Subnet)
			}

			if err := checkReservedPool6ConflictWithReservedPool6s(reservedpool,
				oldReservedpools); err != nil {
				return err
			}

			if err := checkReservedPool6ConflictWithReservation6s(reservedpool,
				reservations); err != nil {
				return err
			}

			recalculateSubnet6AndPool6sCapacityWithReservedPool6(subnet, pools,
				reservedpool, poolsCapacity, true)
			reservedpool.Subnet6 = subnet.GetID()
			reservedPoolValues = append(reservedPoolValues, reservedpool.GenCopyValues())
		}

		if err := updateSubnet6AndPoolsCapacity(tx, subnet, poolsCapacity, nil); err != nil {
			return err
		}

		if _, err := tx.CopyFromEx(resource.TableReservedPool6,
			resource.ReservedPool6Columns, reservedPoolValues); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert,
				string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
		}

		return sendCreateReservedPool6sCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservedpools)
	})
}

func checkReservedPool6ConflictWithReservedPool6s(reservedPool6 *resource.ReservedPool6, reservedpools []*resource.ReservedPool6) error {
	for _, reservedpool := range reservedpools {
		if reservedpool.CheckConflictWithAnother(reservedPool6) {
			return errorno.ErrConflict(errorno.ErrNameDhcpReservedPool,
				errorno.ErrNameDhcpReservedPool, reservedpool.String(),
				reservedPool6.String())
		}
	}

	return nil
}

func sendCreateReservedPool6sCmdToDHCPAgent(subnetID uint64, nodes []string, pools []*resource.ReservedPool6) error {
	if len(nodes) == 0 || len(pools) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.CreateReservedPool6s,
		reservedPool6sToCreateReservedPool6sRequest(subnetID, pools),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservedPool6s,
				reservedPool6sToDeleteReservedPool6sRequest(subnetID, pools)); err != nil {
				log.Errorf("create subnet6 %d reservedpool6 %s failed, rollback %v failed: %s",
					subnetID, pools[0].String(), nodesForSucceed, err.Error())
			}
		})
}

func reservedPool6sToCreateReservedPool6sRequest(subnetID uint64, pools []*resource.ReservedPool6) *pbdhcpagent.CreateReservedPools6Request {
	pbReservedPools := make([]*pbdhcpagent.CreateReservedPool6Request, len(pools))
	for i, pool := range pools {
		pbReservedPools[i] = reservedPool6ToCreateReservedPool6Request(subnetID, pool)
	}

	return &pbdhcpagent.CreateReservedPools6Request{
		SubnetId:      subnetID,
		ReservedPools: pbReservedPools,
	}
}

func reservedPool6sToDeleteReservedPool6sRequest(subnetID uint64, pools []*resource.ReservedPool6) *pbdhcpagent.DeleteReservedPools6Request {
	pbReservedPools := make([]*pbdhcpagent.DeleteReservedPool6Request, len(pools))
	for i, pool := range pools {
		pbReservedPools[i] = reservedPool6ToDeleteReservedPool6Request(subnetID, pool)
	}

	return &pbdhcpagent.DeleteReservedPools6Request{
		SubnetId:      subnetID,
		ReservedPools: pbReservedPools,
	}
}
