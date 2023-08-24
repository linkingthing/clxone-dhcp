package service

import (
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
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
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
		return errorno.ErrNotBelongTo(errorno.ErrNameDhcpReservedPool, errorno.ErrNameNetworkV4,
			pool.String(), subnet.Subnet)
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
		return errorno.ErrConflict(errorno.ErrNameDhcpReservedPool, errorno.ErrNameDhcpReservedPool,
			pool.String(), reservedpools[0].String())
	} else {
		return nil
	}
}

func getReservedPool4sWithBeginAndEndIp(tx restdb.Transaction, subnetID string, begin, end net.IP) ([]*resource.ReservedPool4, error) {
	var reservedpools []*resource.ReservedPool4
	if err := tx.FillEx(&reservedpools,
		"select * from gr_reserved_pool4 where subnet4 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetID, end, begin); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
	} else {
		return reservedpools, nil
	}
}

func checkReservedPool4ConflictWithSubnet4Reservation4s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool4) error {
	if reservations, err := getReservation4sWithBeginAndEndIp(tx, subnetID,
		pool.BeginIp, pool.EndIp); err != nil {
		return err
	} else if len(reservations) != 0 {
		return errorno.ErrConflict(errorno.ErrNameDhcpReservedPool, errorno.ErrNameDhcpReservation,
			pool.String(), reservations[0].String())
	} else {
		return nil
	}
}

func updateSubnet4AndPoolsCapacityWithReservedPool4(tx restdb.Transaction, subnet *resource.Subnet4, reservedPool *resource.ReservedPool4, isCreate bool) error {
	affectPools, err := recalculatePool4sCapacityWithReservedPool4(tx, subnet,
		reservedPool, isCreate)
	if err != nil {
		return err
	}

	if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
		resource.SqlColumnCapacity: subnet.Capacity,
	}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameUpdate, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
	}

	for affectPoolID, capacity := range affectPools {
		if _, err := tx.Update(resource.TablePool4, map[string]interface{}{
			resource.SqlColumnCapacity: capacity,
		}, map[string]interface{}{restdb.IDField: affectPoolID}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
		}
	}

	return nil
}

func recalculatePool4sCapacityWithReservedPool4(tx restdb.Transaction, subnet *resource.Subnet4, reservedPool *resource.ReservedPool4, isCreate bool) (map[string]uint64, error) {
	pools, err := getPool4sWithBeginAndEndIp(tx, subnet.GetID(),
		reservedPool.BeginIp, reservedPool.EndIp)
	if err != nil {
		return nil, err
	}

	affectedPool4s := make(map[string]uint64)
	for _, pool := range pools {
		reservedCount := getPool4ReservedCountWithReservedPool4(pool, reservedPool)
		if isCreate {
			affectedPool4s[pool.GetID()] = pool.Capacity - reservedCount
			subnet.Capacity -= reservedCount
		} else {
			affectedPool4s[pool.GetID()] = pool.Capacity + reservedCount
			subnet.Capacity += reservedCount
		}
	}

	return affectedPool4s, nil
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

func sendCreateReservedPool4CmdToDHCPAgent(subnetID uint64, nodes []string, pools ...*resource.ReservedPool4) error {
	if len(pools) == 0 {
		return nil
	}
	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.CreateReservedPool4s,
		reservedPool4sToCreateReservedPools4Request(subnetID, pools),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservedPool4s,
				reservedPool4sToDeleteReservedPools4Request(subnetID, pools)); err != nil {
				log.Errorf("create subnet4 %d reserved pool4 %s failed, rollback with nodes %v failed: %s",
					subnetID, pools[0].String(), nodesForSucceed, err.Error())
			}
		})
}

func reservedPool4sToCreateReservedPools4Request(subnetID uint64, pools []*resource.ReservedPool4) *pbdhcpagent.CreateReservedPools4Request {
	pbPools := make([]*pbdhcpagent.CreateReservedPool4Request, len(pools))
	for i, pool := range pools {
		pbPools[i] = reservedPool4ToCreateReservedPool4Request(subnetID, pool)
	}
	return &pbdhcpagent.CreateReservedPools4Request{
		SubnetId:      subnetID,
		ReservedPools: pbPools,
	}
}

func reservedPool4ToCreateReservedPool4Request(subnetID uint64, pool *resource.ReservedPool4) *pbdhcpagent.CreateReservedPool4Request {
	return &pbdhcpagent.CreateReservedPool4Request{
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
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet4: subnetID,
			resource.SqlOrderBy:       resource.SqlColumnBeginIp},
			&pools)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
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
			return errorno.ErrDBError(errorno.ErrDBNameDelete, pool.GetID(), pg.Error(err).Error())
		}

		return sendDeleteReservedPool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	})
}

func setReservedPool4FromDB(tx restdb.Transaction, pool *resource.ReservedPool4) error {
	var pools []*resource.ReservedPool4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pool.GetID()},
		&pools); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, pool.GetID(), pg.Error(err).Error())
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

func sendDeleteReservedPool4CmdToDHCPAgent(subnetID uint64, nodes []string, pools ...*resource.ReservedPool4) error {
	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.DeleteReservedPool4s,
		reservedPool4sToDeleteReservedPools4Request(subnetID, pools), nil)
}

func reservedPool4sToDeleteReservedPools4Request(subnetID uint64, pools []*resource.ReservedPool4) *pbdhcpagent.DeleteReservedPools4Request {
	pbPools := make([]*pbdhcpagent.DeleteReservedPool4Request, len(pools))
	for i, pool := range pools {
		pbPools[i] = reservedPool4ToDeleteReservedPool4Request(subnetID, pool)
	}
	return &pbdhcpagent.DeleteReservedPools4Request{
		SubnetId:      subnetID,
		ReservedPools: pbPools,
	}
}

func reservedPool4ToDeleteReservedPool4Request(subnetID uint64, pool *resource.ReservedPool4) *pbdhcpagent.DeleteReservedPool4Request {
	return &pbdhcpagent.DeleteReservedPool4Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *ReservedPool4Service) Update(subnetId string, pool *resource.ReservedPool4) error {
	if err := util.ValidateStrings(util.RegexpTypeComma, pool.Comment); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservedPool4, map[string]interface{}{
			resource.SqlColumnComment: pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, pool.GetID(), pg.Error(err).Error())
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
	subnet4, err := GetSubnet4ByPrefix(prefix)
	if err != nil {
		return nil, err
	}

	if pools, err := listReservedPool4s(subnet4.GetID()); err != nil {
		return nil, err
	} else {
		return pools, nil
	}
}

func BatchCreateReservedPool4s(prefix string, pools []*resource.ReservedPool4) error {
	subnet, err := GetSubnet4ByPrefix(prefix)
	if err != nil {
		return err
	}

	for _, pool := range pools {
		if err := pool.Validate(); err != nil {
			return err
		}
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, pool := range pools {
			if err := checkReservedPool4CouldBeCreated(tx, subnet, pool); err != nil {
				return err
			}

			if err := updateSubnet4AndPoolsCapacityWithReservedPool4(tx, subnet,
				pool, true); err != nil {
				return err
			}

			pool.Subnet4 = subnet.GetID()
			if _, err := tx.Insert(pool); err != nil {
				return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
			}

			if err := sendCreateReservedPool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool); err != nil {
				return err
			}
		}

		return nil
	})
}
