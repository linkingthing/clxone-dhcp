package service

import (
	"context"
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	"github.com/linkingthing/clxone-dhcp/pkg/util"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type ReservedPdPoolService struct {
}

func NewReservedPdPoolService() *ReservedPdPoolService {
	return &ReservedPdPoolService{}
}

func (p *ReservedPdPoolService) Create(subnet *resource.Subnet6, pdpool *resource.ReservedPdPool) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		} else if subnet.UseEui64 {
			return fmt.Errorf("subnet use EUI64, can not create reserved pdpool")
		}

		if err := checkPrefixBelongsToIpnet(subnet.Ipnet, pdpool.PrefixIpnet,
			pdpool.PrefixLen); err != nil {
			return err
		}

		if err := checkReservedPdPoolConflictWithSubnet6Pools(tx, subnet.GetID(),
			pdpool); err != nil {
			return err
		}

		pdpool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pdpool); err != nil {
			return err
		}

		return sendCreateReservedPdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdpool)
	}); err != nil {
		return nil, err
	}

	return pdpool, nil
}

func checkReservedPdPoolConflictWithSubnet6Pools(tx restdb.Transaction, subnetID string, pdpool *resource.ReservedPdPool) error {
	if err := checkReservedPdPoolConflictWithSubnet6ReservedPdPools(tx,
		subnetID, pdpool); err != nil {
		return err
	}

	return checkReservedPdPoolConflictWithSubnet6Reservation6s(tx, subnetID, pdpool)
}

func checkReservedPdPoolConflictWithSubnet6ReservedPdPools(tx restdb.Transaction, subnetID string, pdpool *resource.ReservedPdPool) error {
	var pdpools []*resource.ReservedPdPool
	if err := tx.FillEx(&pdpools,
		"select * from gr_reserved_pd_pool where subnet6 = $1 and prefix_ipnet && $2",
		subnetID, pdpool.PrefixIpnet); err != nil {
		return fmt.Errorf("get reserved pdpools with subnet %s from db failed: %s",
			subnetID, err.Error())
	}

	if len(pdpools) != 0 {
		return fmt.Errorf("reserved pdpool %s conflict with reserved pdpool %s",
			pdpool.String(), pdpools[0].String())
	}

	return nil
}

func checkReservedPdPoolConflictWithSubnet6Reservation6s(tx restdb.Transaction, subnetID string, pdpool *resource.ReservedPdPool) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetID},
		&reservations); err != nil {
		return fmt.Errorf("get reservation6s with subnet %s from db failed: %s",
			subnetID, err.Error())
	}

	for _, reservation := range reservations {
		for _, prefix := range reservation.Prefixes {
			if pdpool.Contains(prefix) {
				return fmt.Errorf("reserved pdpool %s conflict with reservation6 %s prefix %s",
					pdpool.String(), reservation.String(), prefix)
			}
		}
	}

	return nil
}

func sendCreateReservedPdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.ReservedPdPool) error {
	nodesForSucceed, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreateReservedPdPool,
		reservedPdPoolToCreateReservedPdPoolRequest(subnetID, pdpool))
	if err != nil {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, kafka.DeleteReservedPdPool,
			reservedPdPoolToDeleteReservedPdPoolRequest(subnetID, pdpool)); err != nil {
			log.Errorf("create subnet %d reserved pdpool %s failed, and rollback it failed: %s",
				subnetID, pdpool.String(), err.Error())
		}
	}

	return err
}

func reservedPdPoolToCreateReservedPdPoolRequest(subnetID uint64, pdpool *resource.ReservedPdPool) *pbdhcpagent.CreateReservedPdPoolRequest {
	return &pbdhcpagent.CreateReservedPdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}

func (p *ReservedPdPoolService) List(subnetID string) (interface{}, error) {
	var pdpools []*resource.ReservedPdPool
	if err := db.GetResources(map[string]interface{}{
		resource.SqlColumnSubnet6: subnetID,
		util.SqlOrderBy:           resource.SqlColumnPrefixIpNet}, &pdpools); err != nil {
		return nil, err
	}

	return pdpools, nil
}

func (p *ReservedPdPoolService) Get(subnetID, pdpoolID string) (restresource.Resource, error) {
	var pdpools []*resource.ReservedPdPool
	pdpool, err := restdb.GetResourceWithID(db.GetDB(), pdpoolID, &pdpools)
	if err != nil {
		return nil,
			fmt.Errorf("get pdpool %s with subnet %s from db failed: %s", pdpoolID, subnetID, err.Error())
	}

	return pdpool.(*resource.ReservedPdPool), nil
}

func (p *ReservedPdPoolService) Delete(subnet *resource.Subnet6, pdpool *resource.ReservedPdPool) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservedPdPoolFromDB(tx, pdpool); err != nil {
			return err
		}

		pdpool.Subnet6 = subnet.GetID()
		if leasesCount, err := getReservedPdPoolLeasesCount(pdpool); err != nil {
			return fmt.Errorf("get pdpool %s leases count failed: %s",
				pdpool.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete pdpool with %d ips had been allocated",
				leasesCount)
		}

		if _, err := tx.Delete(resource.TableReservedPdPool,
			map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservedPdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdpool)
	}); err != nil {
		return err
	}

	return nil
}

func setReservedPdPoolFromDB(tx restdb.Transaction, pdpool *resource.ReservedPdPool) error {
	var pdpools []*resource.ReservedPdPool
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pdpool.GetID()},
		&pdpools); err != nil {
		return fmt.Errorf("get pdpool from db failed: %s", err.Error())
	}

	if len(pdpools) == 0 {
		return fmt.Errorf("no found pool %s", pdpool.GetID())
	}

	pdpool.Subnet6 = pdpools[0].Subnet6
	pdpool.Prefix = pdpools[0].Prefix
	pdpool.PrefixLen = pdpools[0].PrefixLen
	pdpool.PrefixIpnet = pdpools[0].PrefixIpnet
	pdpool.DelegatedLen = pdpools[0].DelegatedLen
	pdpool.Capacity = pdpools[0].Capacity
	return nil
}

func getReservedPdPoolLeasesCount(pdpool *resource.ReservedPdPool) (uint64, error) {
	if pdpool.Capacity == 0 {
		return 0, nil
	}

	beginAddr, endAddr := pdpool.GetRange()
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetPool6LeasesCount(context.TODO(),
		&pbdhcpagent.GetPool6LeasesCountRequest{
			SubnetId:     subnetIDStrToUint64(pdpool.Subnet6),
			BeginAddress: beginAddr,
			EndAddress:   endAddr,
		})
	return resp.GetLeasesCount(), err
}

func sendDeleteReservedPdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.ReservedPdPool) error {
	_, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeleteReservedPdPool,
		reservedPdPoolToDeleteReservedPdPoolRequest(subnetID, pdpool))
	return err
}

func reservedPdPoolToDeleteReservedPdPoolRequest(subnetID uint64, pdpool *resource.ReservedPdPool) *pbdhcpagent.DeleteReservedPdPoolRequest {
	return &pbdhcpagent.DeleteReservedPdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}

func (p *ReservedPdPoolService) Update(pool *resource.ReservedPdPool) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservedPdPool, map[string]interface{}{
			util.SqlColumnsComment: pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found reserved pdpool %s", pool.GetID())
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return pool, nil
}
