package service

import (
	"math/big"

	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type ReservedPdPoolService struct {
}

func NewReservedPdPoolService() *ReservedPdPoolService {
	return &ReservedPdPoolService{}
}

func (p *ReservedPdPoolService) Create(subnet *resource.Subnet6, pdpool *resource.ReservedPdPool) error {
	if err := pdpool.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservedPdPoolCouldBeCreated(tx, subnet, pdpool); err != nil {
			return err
		}

		if err := updateSubnet6AndPdPoolsCapacityWithReservedPdPool(tx, subnet,
			pdpool, true); err != nil {
			return err
		}

		pdpool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pdpool); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNamePdPool), pg.Error(err).Error())
		}

		return sendCreateReservedPdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdpool)
	})
}

func checkReservedPdPoolCouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet6, pdpool *resource.ReservedPdPool) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	} else if subnet.UseEui64 || subnet.UseAddressCode {
		return errorno.ErrSubnetWithEui64(subnet.Subnet)
	}

	if err := checkPrefixBelongsToIpnet(subnet.Ipnet, pdpool.PrefixIpnet,
		pdpool.PrefixLen); err != nil {
		return err
	}

	return checkReservedPdPoolConflictWithSubnet6Pools(tx, subnet.GetID(), pdpool)
}

func checkReservedPdPoolConflictWithSubnet6Pools(tx restdb.Transaction, subnetID string, pdpool *resource.ReservedPdPool) error {
	if err := checkReservedPdPoolConflictWithSubnet6ReservedPdPools(tx,
		subnetID, pdpool); err != nil {
		return err
	}

	return checkReservedPdPoolConflictWithSubnet6Reservation6s(tx, subnetID, pdpool)
}

func checkReservedPdPoolConflictWithSubnet6ReservedPdPools(tx restdb.Transaction, subnetID string, pdpool *resource.ReservedPdPool) error {
	if pdpools, err := getReservedPdPoolsWithPrefix(tx, subnetID, pdpool.PrefixIpnet); err != nil {
		return err
	} else if len(pdpools) != 0 {
		return errorno.ErrConflict(errorno.ErrNameReservedPdPool, errorno.ErrNameReservedPdPool,
			pdpool.String(), pdpools[0].String())
	} else {
		return nil
	}
}

func checkReservedPdPoolConflictWithSubnet6Reservation6s(tx restdb.Transaction, subnetID string, pdpool *resource.ReservedPdPool) error {
	reservations, err := getReservation6sWithPrefixesExists(tx, subnetID)
	if err != nil {
		return err
	}

	for _, reservation := range reservations {
		for _, ipnet := range reservation.Ipnets {
			if pdpool.IntersectIpnet(ipnet) {
				return errorno.ErrConflict(errorno.ErrNameReservedPdPool, errorno.ErrNameDhcpReservation,
					pdpool.String(), reservation.String())
			}
		}
	}

	return nil
}

func updateSubnet6AndPdPoolsCapacityWithReservedPdPool(tx restdb.Transaction, subnet *resource.Subnet6, reservedPdPool *resource.ReservedPdPool, isCreate bool) error {
	affectPdPools, err := recalculatePdPoolsCapacityWithReservedPdPool(tx, subnet,
		reservedPdPool, isCreate)
	if err != nil {
		return err
	}

	if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
		resource.SqlColumnCapacity: subnet.Capacity,
	}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameUpdate, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	}

	for affectPdPoolID, capacity := range affectPdPools {
		if _, err := tx.Update(resource.TablePdPool, map[string]interface{}{
			resource.SqlColumnCapacity: capacity,
		}, map[string]interface{}{restdb.IDField: affectPdPoolID}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, string(errorno.ErrNamePdPool), pg.Error(err).Error())
		}
	}

	return nil
}

func recalculatePdPoolsCapacityWithReservedPdPool(tx restdb.Transaction, subnet *resource.Subnet6, reservedPdPool *resource.ReservedPdPool, isCreate bool) (map[string]string, error) {
	pdpools, err := getPdPoolsWithPrefix(tx, subnet.GetID(), reservedPdPool.PrefixIpnet)
	if err != nil {
		return nil, err
	}

	allReservedCount := new(big.Int)
	affectedPdPools := make(map[string]string)
	for _, pdpool := range pdpools {
		if pdpool.IntersectIpnet(reservedPdPool.PrefixIpnet) {
			reservedCount := getPdPoolReservedCount(pdpool, reservedPdPool.PrefixLen)
			allReservedCount.Add(allReservedCount, reservedCount)
			if isCreate {
				affectedPdPools[pdpool.GetID()] = pdpool.SubCapacityWithBigInt(reservedCount)
			} else {
				affectedPdPools[pdpool.GetID()] = pdpool.AddCapacityWithBigInt(reservedCount)
			}

			break
		}
	}

	if isCreate {
		subnet.SubCapacityWithBigInt(allReservedCount)
	} else {
		subnet.AddCapacityWithBigInt(allReservedCount)
	}

	return affectedPdPools, nil
}

func sendCreateReservedPdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.ReservedPdPool) error {
	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreateReservedPdPool,
		reservedPdPoolToCreateReservedPdPoolRequest(subnetID, pdpool),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservedPdPool,
				reservedPdPoolToDeleteReservedPdPoolRequest(subnetID, pdpool)); err != nil {
				log.Errorf("create subnet %d reserved pdpool %s failed, rollback with nodes %v failed: %s",
					subnetID, pdpool.String(), nodesForSucceed, err.Error())
			}
		})
}

func reservedPdPoolToCreateReservedPdPoolRequest(subnetID uint64, pdpool *resource.ReservedPdPool) *pbdhcpagent.CreateReservedPdPoolRequest {
	return &pbdhcpagent.CreateReservedPdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}

func (p *ReservedPdPoolService) List(subnetID string) ([]*resource.ReservedPdPool, error) {
	var pdpools []*resource.ReservedPdPool
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet6: subnetID,
			resource.SqlOrderBy:       resource.SqlColumnPrefixIpNet}, &pdpools)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameReservedPdPool), pg.Error(err).Error())
	}

	return pdpools, nil
}

func (p *ReservedPdPoolService) Get(subnet *resource.Subnet6, pdpoolID string) (*resource.ReservedPdPool, error) {
	var pdpools []*resource.ReservedPdPool
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: pdpoolID}, &pdpools)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, pdpoolID, pg.Error(err).Error())
	} else if len(pdpools) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameReservedPdPool, pdpoolID)
	}

	return pdpools[0], nil
}

func (p *ReservedPdPoolService) Delete(subnet *resource.Subnet6, pdpool *resource.ReservedPdPool) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservedPdPoolFromDB(tx, pdpool); err != nil {
			return err
		}

		if err := updateSubnet6AndPdPoolsCapacityWithReservedPdPool(tx, subnet,
			pdpool, false); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableReservedPdPool,
			map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, pdpool.GetID(), pg.Error(err).Error())
		}

		return sendDeleteReservedPdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdpool)
	})
}

func setReservedPdPoolFromDB(tx restdb.Transaction, pdpool *resource.ReservedPdPool) error {
	var pdpools []*resource.ReservedPdPool
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pdpool.GetID()},
		&pdpools); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, pdpool.GetID(), pg.Error(err).Error())
	} else if len(pdpools) == 0 {
		return errorno.ErrNotFound(errorno.ErrNameReservedPdPool, pdpool.GetID())
	}

	pdpool.Subnet6 = pdpools[0].Subnet6
	pdpool.Prefix = pdpools[0].Prefix
	pdpool.PrefixLen = pdpools[0].PrefixLen
	pdpool.PrefixIpnet = pdpools[0].PrefixIpnet
	pdpool.DelegatedLen = pdpools[0].DelegatedLen
	pdpool.Capacity = pdpools[0].Capacity
	return nil
}

func sendDeleteReservedPdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.ReservedPdPool) error {
	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeleteReservedPdPool,
		reservedPdPoolToDeleteReservedPdPoolRequest(subnetID, pdpool), nil)
}

func reservedPdPoolToDeleteReservedPdPoolRequest(subnetID uint64, pdpool *resource.ReservedPdPool) *pbdhcpagent.DeleteReservedPdPoolRequest {
	return &pbdhcpagent.DeleteReservedPdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}

func (p *ReservedPdPoolService) Update(subnetId string, pool *resource.ReservedPdPool) error {
	if err := resource.CheckCommentValid(pool.Comment); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservedPdPool, map[string]interface{}{
			resource.SqlColumnComment: pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, pool.GetID(), pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameReservedPdPool, pool.GetID())
		}

		return nil
	})
}
