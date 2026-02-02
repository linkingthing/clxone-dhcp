package service

import (
	"net"
	"strconv"

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

type ReservedPool4Service struct {
}

func NewReservedPool4Service() *ReservedPool4Service {
	return &ReservedPool4Service{}
}

func (p *ReservedPool4Service) Create(subnet *resource.Subnet4, pool *resource.ReservedPool4) error {
	if err := pool.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservedPool4CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := updateSubnet4AndPoolsCapacityWithReservedPool4(tx, subnet,
			pool, true); err != nil {
			return err
		}

		pool.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery,
				string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
		}

		return sendCreateReservedPool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	})
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

	if !checkIPsBelongsToIpnet(subnet.Ipnet, pool.BeginIp, pool.EndIp) {
		return errorno.ErrNotBelongTo(errorno.ErrNameDhcpReservedPool,
			errorno.ErrNameNetworkV4, pool.String(), subnet.Subnet)
	}

	return checkReservedPool4ConflictWithSubnet4Pools(tx, subnet.GetID(), pool)
}

func checkReservedPool4ConflictWithSubnet4Pools(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool4) error {
	if err := checkReservedPool4ConflictWithSubnet4ReservedPool4s(tx,
		subnetID, pool); err != nil {
		return err
	}

	return checkReservedPool4ConflictWithSubnet4Reservation4s(tx, subnetID, pool)
}

func checkReservedPool4ConflictWithSubnet4ReservedPool4s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool4) error {
	if reservedpools, err := getReservedPool4sWithBeginAndEndIp(tx, subnetID,
		pool.BeginIp, pool.EndIp); err != nil {
		return err
	} else if len(reservedpools) != 0 {
		return errorno.ErrConflict(errorno.ErrNameDhcpReservedPool,
			errorno.ErrNameDhcpReservedPool, pool.String(), reservedpools[0].String())
	} else {
		return nil
	}
}

func getReservedPool4sWithBeginAndEndIp(tx restdb.Transaction, subnetID string, begin, end net.IP) ([]*resource.ReservedPool4, error) {
	var reservedpools []*resource.ReservedPool4
	if err := tx.FillEx(&reservedpools,
		`select * from gr_reserved_pool4 where
			subnet4 = $1 and begin_ip <= $2 and end_ip >= $3`,
		subnetID, end, begin); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
	} else {
		return reservedpools, nil
	}
}

func checkReservedPool4ConflictWithSubnet4Reservation4s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool4) error {
	if reservations, err := getReservation4sWithBeginAndEndIp(tx, subnetID,
		pool.BeginIp, pool.EndIp); err != nil {
		return err
	} else if len(reservations) != 0 {
		return errorno.ErrConflict(errorno.ErrNameDhcpReservedPool,
			errorno.ErrNameDhcpReservation, pool.String(), reservations[0].String())
	} else {
		return nil
	}
}

func updateSubnet4AndPoolsCapacityWithReservedPool4(tx restdb.Transaction, subnet *resource.Subnet4, reservedPool *resource.ReservedPool4, isCreate bool) error {
	pools, err := getPool4sWithBeginAndEndIp(tx, subnet.GetID(),
		reservedPool.BeginIp, reservedPool.EndIp)
	if err != nil {
		return err
	}

	poolsCapacity := make(map[string]uint64, len(pools))
	recalculateSubnet4AndPool4sCapacityWithReservedPool4(subnet, pools, reservedPool,
		poolsCapacity, isCreate)
	return updateSubnet4AndPool4sCapacity(tx, subnet, poolsCapacity)
}

func recalculateSubnet4AndPool4sCapacityWithReservedPool4(subnet *resource.Subnet4, pools []*resource.Pool4, reservedPool4 *resource.ReservedPool4, poolsCapacity map[string]uint64, isCreate bool) {
	for _, pool := range pools {
		reservedCount := getPool4ReservedCountWithReservedPool4(pool, reservedPool4)
		if reservedCount == 0 {
			continue
		}

		if isCreate {
			poolsCapacity[pool.GetID()] = pool.Capacity - reservedCount
			subnet.Capacity -= reservedCount
		} else {
			poolsCapacity[pool.GetID()] = pool.Capacity + reservedCount
			subnet.Capacity += reservedCount
		}
	}
}

func getPool4ReservedCountWithReservedPool4(pool *resource.Pool4, reservedPool *resource.ReservedPool4) uint64 {
	begin := gohelperip.IPv4ToUint32(pool.BeginIp)
	if reservedPoolBegin := gohelperip.IPv4ToUint32(
		reservedPool.BeginIp); reservedPoolBegin > begin {
		begin = reservedPoolBegin
	}

	end := gohelperip.IPv4ToUint32(pool.EndIp)
	if reservedPoolEnd := gohelperip.IPv4ToUint32(
		reservedPool.EndIp); reservedPoolEnd < end {
		end = reservedPoolEnd
	}

	return uint64(end) - uint64(begin) + 1
}

func batchUpdatePool4sCapacity(tx restdb.Transaction, poolsCapacity map[string]uint64) error {
	if len(poolsCapacity) == 0 {
		return nil
	}

	updateValues := make(map[string]string, len(poolsCapacity))
	for poolId, capacity := range poolsCapacity {
		updateValues[poolId] = "('" + poolId + "'," + strconv.FormatUint(capacity, 10) + ")"
	}

	if err := util.BatchUpdateById(tx, string(resource.TablePool4),
		[]string{resource.SqlColumnCapacity}, updateValues); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameUpdate,
			string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
	}

	return nil
}

func sendCreateReservedPool4CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.ReservedPool4) error {
	if len(nodes) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.CreateReservedPool4,
		reservedPool4ToCreateReservedPool4Request(subnetID, pool),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservedPool4,
				reservedPool4ToDeleteReservedPool4Request(subnetID, pool)); err != nil {
				log.Errorf("create subnet4 %d reservedpool4 %s failed, rollback %v failed: %s",
					subnetID, pool.String(), nodesForSucceed, err.Error())
			}
		})
}

func reservedPool4ToCreateReservedPool4Request(subnetID uint64, pool *resource.ReservedPool4) *pbdhcpagent.CreateReservedPool4Request {
	return &pbdhcpagent.CreateReservedPool4Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func reservedPool4ToDeleteReservedPool4Request(subnetID uint64, pool *resource.ReservedPool4) *pbdhcpagent.DeleteReservedPool4Request {
	return &pbdhcpagent.DeleteReservedPool4Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *ReservedPool4Service) List(subnetID string) ([]*resource.ReservedPool4, error) {
	return listReservedPool4s(subnetID)
}

func listReservedPool4s(subnetID string) ([]*resource.ReservedPool4, error) {
	var pools []*resource.ReservedPool4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		pools, err = getReservedPool4sWithCondition(tx, map[string]interface{}{
			resource.SqlColumnSubnet4: subnetID,
			resource.SqlOrderBy:       resource.SqlColumnBeginIp})
		return
	}); err != nil {
		return nil, err
	}

	return pools, nil
}

func getReservedPool4sWithCondition(tx restdb.Transaction, condition map[string]interface{}) ([]*resource.ReservedPool4, error) {
	var pools []*resource.ReservedPool4
	if err := tx.Fill(condition, &pools); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
	}

	return pools, nil
}

func (p *ReservedPool4Service) Get(subnet *resource.Subnet4, poolID string) (*resource.ReservedPool4, error) {
	var pools []*resource.ReservedPool4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, poolID, pg.Error(err).Error())
	} else if len(pools) != 1 {
		return nil, errorno.ErrNotFound(errorno.ErrNameDhcpReservedPool, poolID)
	}

	return pools[0], nil
}

func (p *ReservedPool4Service) Delete(subnet *resource.Subnet4, pool *resource.ReservedPool4) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservedPool4FromDB(tx, pool); err != nil {
			return err
		}

		if err := updateSubnet4AndPoolsCapacityWithReservedPool4(tx, subnet,
			pool, false); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableReservedPool4, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, pool.GetID(),
				pg.Error(err).Error())
		}

		return sendDeleteReservedPool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	})
}

func setReservedPool4FromDB(tx restdb.Transaction, pool *resource.ReservedPool4) error {
	pools, err := getReservedPool4sWithCondition(tx,
		map[string]interface{}{restdb.IDField: pool.GetID()})
	if err != nil {
		return err
	} else if len(pools) == 0 {
		return errorno.ErrNotFound(errorno.ErrNameDhcpReservedPool, pool.GetID())
	}

	pool.Subnet4 = pools[0].Subnet4
	pool.BeginAddress = pools[0].BeginAddress
	pool.BeginIp = pools[0].BeginIp
	pool.EndAddress = pools[0].EndAddress
	pool.EndIp = pools[0].EndIp
	pool.Capacity = pools[0].Capacity
	return nil
}

func sendDeleteReservedPool4CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.ReservedPool4) error {
	if len(nodes) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.DeleteReservedPool4,
		reservedPool4ToDeleteReservedPool4Request(subnetID, pool), nil)
}

func (p *ReservedPool4Service) Update(subnetId string, pool *resource.ReservedPool4) error {
	if err := util.ValidateStrings(util.RegexpTypeComma, pool.Comment); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservedPool4, map[string]interface{}{
			resource.SqlColumnComment: pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, pool.GetID(),
				pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameDhcpReservedPool, pool.GetID())
		}

		return nil
	})
}

func (p *ReservedPool4Service) ActionValidTemplate(subnet *resource.Subnet4, pool *resource.ReservedPool4,
	templateInfo *resource.TemplateInfo) (*resource.TemplatePool, error) {
	pool.Template = templateInfo.Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkReservedPool4CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, err
	}

	return &resource.TemplatePool{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress}, nil
}

func GetReservedPool4sByPrefix(prefix string) ([]*resource.ReservedPool4, error) {
	if subnet4, err := GetSubnet4ByPrefix(prefix); err != nil {
		return nil, err
	} else {
		return listReservedPool4s(subnet4.GetID())
	}
}

func BatchCreateReservedPool4s(prefix string, reservedpools []*resource.ReservedPool4) error {
	for _, reservedpool := range reservedpools {
		if err := reservedpool.Validate(); err != nil {
			return err
		}
	}

	for i, reservedpool := range reservedpools {
		if err := checkReservedPool4ConflictWithReservedPool4s(reservedpool,
			reservedpools[i+1:]); err != nil {
			return err
		}
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet, err := getSubnet4WithPrefix(tx, prefix)
		if err != nil {
			return err
		}

		oldReservedpools, err := getReservedPool4sWithSubnetId(tx, subnet.GetID())
		if err != nil {
			return err
		}

		reservations, err := getReservation4sWithSubnetId(tx, subnet.GetID())
		if err != nil {
			return err
		}

		pools, err := getPool4sWithSubnetId(tx, subnet.GetID())
		if err != nil {
			return err
		}

		poolsCapacity := make(map[string]uint64, len(pools))
		reservedPoolValues := make([][]interface{}, 0, len(reservedpools))
		for _, reservedpool := range reservedpools {
			if !checkIPsBelongsToIpnet(subnet.Ipnet,
				reservedpool.BeginIp, reservedpool.EndIp) {
				return errorno.ErrNotBelongTo(errorno.ErrNameDhcpReservedPool,
					errorno.ErrNameNetworkV4, reservedpool.String(), subnet.Subnet)
			}

			if err := checkReservedPool4ConflictWithReservedPool4s(reservedpool,
				oldReservedpools); err != nil {
				return err
			}

			if err := checkReservedPool4ConflictWithReservation4s(reservedpool,
				reservations); err != nil {
				return err
			}

			recalculateSubnet4AndPool4sCapacityWithReservedPool4(subnet, pools,
				reservedpool, poolsCapacity, true)
			reservedpool.Subnet4 = subnet.GetID()
			reservedPoolValues = append(reservedPoolValues, reservedpool.GenCopyValues())
		}

		if err := updateSubnet4AndPool4sCapacity(tx, subnet, poolsCapacity); err != nil {
			return err
		}

		if _, err := tx.CopyFromEx(resource.TableReservedPool4,
			resource.ReservedPool4Columns, reservedPoolValues); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert,
				string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
		}

		return sendCreateReservedPool4sCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservedpools)
	})
}

func checkReservedPool4ConflictWithReservedPool4s(reservedPool4 *resource.ReservedPool4, reservedPools []*resource.ReservedPool4) error {
	for _, reservedPool := range reservedPools {
		if reservedPool.CheckConflictWithAnother(reservedPool4) {
			return errorno.ErrConflict(errorno.ErrNameDhcpReservedPool,
				errorno.ErrNameDhcpReservedPool, reservedPool.String(), reservedPool4.String())
		}
	}

	return nil
}

func checkReservedPool4ConflictWithReservation4s(reservedPool4 *resource.ReservedPool4, reservations []*resource.Reservation4) error {
	for _, reservation := range reservations {
		if reservedPool4.ContainsIp(reservation.Ip) {
			return errorno.ErrConflict(errorno.ErrNameDhcpReservedPool,
				errorno.ErrNameDhcpReservation, reservedPool4.String(), reservation.String())
		}
	}

	return nil
}

func sendCreateReservedPool4sCmdToDHCPAgent(subnetID uint64, nodes []string, pools []*resource.ReservedPool4) error {
	if len(nodes) == 0 || len(pools) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.CreateReservedPool4s,
		reservedPool4sToCreateReservedPool4sRequest(subnetID, pools),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservedPool4s,
				reservedPool4sToDeleteReservedPool4sRequest(subnetID, pools)); err != nil {
				log.Errorf("create subnet4 %d reservedpool4 %s failed, rollback %v failed: %s",
					subnetID, pools[0].String(), nodesForSucceed, err.Error())
			}
		})
}

func reservedPool4sToCreateReservedPool4sRequest(subnetID uint64, pools []*resource.ReservedPool4) *pbdhcpagent.CreateReservedPools4Request {
	pbReservedPools := make([]*pbdhcpagent.CreateReservedPool4Request, len(pools))
	for i, pool := range pools {
		pbReservedPools[i] = reservedPool4ToCreateReservedPool4Request(subnetID, pool)
	}

	return &pbdhcpagent.CreateReservedPools4Request{
		SubnetId:      subnetID,
		ReservedPools: pbReservedPools,
	}
}

func reservedPool4sToDeleteReservedPool4sRequest(subnetID uint64, pools []*resource.ReservedPool4) *pbdhcpagent.DeleteReservedPools4Request {
	pbReservedPools := make([]*pbdhcpagent.DeleteReservedPool4Request, len(pools))
	for i, pool := range pools {
		pbReservedPools[i] = reservedPool4ToDeleteReservedPool4Request(subnetID, pool)
	}

	return &pbdhcpagent.DeleteReservedPools4Request{
		SubnetId:      subnetID,
		ReservedPools: pbReservedPools,
	}
}
