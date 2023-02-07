package service

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
)

type PdPoolService struct {
}

func NewPdPoolService() *PdPoolService {
	return &PdPoolService{}
}

func (p *PdPoolService) Create(subnet *resource.Subnet6, pdpool *resource.PdPool) error {
	if err := pdpool.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPdPoolCouldBeCreated(tx, subnet, pdpool); err != nil {
			return err
		}

		if err := recalculatePdPoolCapacity(tx, subnet.GetID(), pdpool); err != nil {
			return err
		}

		if err := updateSubnet6CapacityWithPdPool(tx, subnet.GetID(),
			subnet.AddCapacityWithString(pdpool.Capacity)); err != nil {
			return err
		}

		pdpool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pdpool); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNamePdPool), pg.Error(err).Error())
		}

		return sendCreatePdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdpool)
	})
}

func checkPdPoolCouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet6, pdpool *resource.PdPool) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	} else if subnet.UseEui64 || subnet.UseAddressCode {
		return errorno.ErrSubnetWithEui64(subnet.Subnet)
	}

	if err := checkPrefixBelongsToIpnet(subnet.Ipnet, pdpool.PrefixIpnet,
		pdpool.PrefixLen); err != nil {
		return err
	}

	return checkPdPoolConflictWithSubnet6PdPools(tx, subnet.GetID(), pdpool)
}

func checkPrefixBelongsToIpnet(ipnet, prefixIpnet net.IPNet, prefixLen uint32) error {
	if ones, _ := ipnet.Mask.Size(); uint32(ones) > prefixLen {
		return errorno.ErrBiggerThan(errorno.ErrNamePrefix, ipnet.String(), prefixIpnet.String())
	}

	if !ipnet.Contains(prefixIpnet.IP) {
		return errorno.ErrNotBelongTo(errorno.ErrNamePdPool, errorno.ErrNameNetworkV6,
			prefixIpnet.String(), ipnet.String())
	}

	return nil
}

func checkPdPoolConflictWithSubnet6PdPools(tx restdb.Transaction, subnetID string, pdpool *resource.PdPool) error {
	if pdpools, err := getPdPoolsWithPrefix(tx, subnetID, pdpool.PrefixIpnet); err != nil {
		return err
	} else if len(pdpools) != 0 {
		return errorno.ErrConflict(errorno.ErrNamePdPool, errorno.ErrNamePdPool,
			pdpool.String(), pdpools[0].String())
	} else {
		return nil
	}
}

func getPdPoolsWithPrefix(tx restdb.Transaction, subnetID string, prefix net.IPNet) ([]*resource.PdPool, error) {
	var pdpools []*resource.PdPool
	if err := tx.FillEx(&pdpools,
		"select * from gr_pd_pool where subnet6 = $1 and prefix_ipnet && $2",
		subnetID, prefix); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNamePdPool), pg.Error(err).Error())
	} else {
		return pdpools, nil
	}
}

func recalculatePdPoolCapacity(tx restdb.Transaction, subnetID string, pdpool *resource.PdPool) error {
	reservations, err := getReservation6sWithPrefixesExists(tx, subnetID)
	if err != nil {
		return err
	}

	reservedPdPools, err := getReservedPdPoolsWithPrefix(tx, subnetID, pdpool.PrefixIpnet)
	if err != nil {
		return err
	}

	recalculatePdPoolCapacityWithReservations(pdpool, reservations)
	recalculatePdPoolCapacityWithReservedPdPools(pdpool, reservedPdPools)
	return nil
}

func getReservation6sWithPrefixesExists(tx restdb.Transaction, subnetID string) ([]*resource.Reservation6, error) {
	var reservations []*resource.Reservation6
	if err := tx.FillEx(&reservations,
		"select * from gr_reservation6 where subnet6 = $1 and prefixes != '{}'",
		subnetID); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	} else {
		return reservations, nil
	}
}

func getReservedPdPoolsWithPrefix(tx restdb.Transaction, subnetID string, prefix net.IPNet) ([]*resource.ReservedPdPool, error) {
	var pdpools []*resource.ReservedPdPool
	if err := tx.FillEx(&pdpools,
		"select * from gr_reserved_pd_pool where subnet6 = $1 and prefix_ipnet && $2",
		subnetID, prefix); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameReservedPdPool), pg.Error(err).Error())
	} else {
		return pdpools, nil
	}
}

func recalculatePdPoolCapacityWithReservations(pdpool *resource.PdPool, reservations []*resource.Reservation6) {
	for _, reservation := range reservations {
		for _, ipnet := range reservation.Ipnets {
			if pdpool.IntersectIpnet(ipnet) {
				pdpool.SubCapacityWithBigInt(getPdPoolReservedCountWithPrefix(pdpool, ipnet))
			}
		}
	}
}

func recalculatePdPoolCapacityWithReservedPdPools(pdpool *resource.PdPool, reservedPdPools []*resource.ReservedPdPool) {
	for _, reservedPdPool := range reservedPdPools {
		if pdpool.IntersectIpnet(reservedPdPool.PrefixIpnet) {
			pdpool.SubCapacityWithBigInt(getPdPoolReservedCount(pdpool, reservedPdPool.PrefixLen))
		}
	}
}

func updateSubnet6CapacityWithPdPool(tx restdb.Transaction, subnetID string, capacity string) error {
	if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
		resource.SqlColumnCapacity: capacity,
	}, map[string]interface{}{restdb.IDField: subnetID}); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameUpdate, subnetID, pg.Error(err).Error())
	} else {
		return nil
	}

}

func sendCreatePdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.PdPool) error {
	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreatePdPool,
		pdpoolToCreatePdPoolRequest(subnetID, pdpool), func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeletePdPool,
				pdpoolToDeletePdPoolRequest(subnetID, pdpool)); err != nil {
				log.Errorf("create subnet6 %d pdpool %s failed, rollback with nodes %v failed: %s",
					subnetID, pdpool.String(), nodesForSucceed, err.Error())
			}
		})
}

func pdpoolToCreatePdPoolRequest(subnetID uint64, pdpool *resource.PdPool) *pbdhcpagent.CreatePdPoolRequest {
	return &pbdhcpagent.CreatePdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}

func (p *PdPoolService) List(subnet *resource.Subnet6) ([]*resource.PdPool, error) {
	return listPdPools(subnet)
}

func listPdPools(subnet *resource.Subnet6) ([]*resource.PdPool, error) {
	var pdpools []*resource.PdPool
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		err := tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet6: subnet.GetID(),
			resource.SqlOrderBy:       resource.SqlColumnPrefixIpNet}, &pdpools)
		if err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNamePdPool), pg.Error(err).Error())
		}

		if len(subnet.Nodes) != 0 {
			reservations, err = getReservation6sWithPrefixesExists(tx, subnet.GetID())
		}

		return err
	}); err != nil {
		return nil, err
	}

	if len(subnet.Nodes) != 0 {
		pdpoolsLeases := loadPdPoolsLeases(subnet.GetID(), pdpools, reservations)
		for _, pdpool := range pdpools {
			setPdPoolLeasesUsedRatio(pdpool, pdpoolsLeases[pdpool.GetID()])
		}
	}

	return pdpools, nil
}

func loadPdPoolsLeases(subnetID string, pdpools []*resource.PdPool, reservations []*resource.Reservation6) map[string]uint64 {
	resp, err := getSubnet6Leases(subnetIDStrToUint64(subnetID))
	if err != nil {
		log.Warnf("get subnet6 %s leases failed: %s", subnetID, err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := reservationPrefixMapFromReservation6s(reservations)
	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		leasePrefix := prefixFromAddressAndPrefixLen(lease.GetAddress(), lease.GetPrefixLen())
		if _, ok := reservationMap[leasePrefix]; ok {
			continue
		}

		for _, pdpool := range pdpools {
			if !resource.IsCapacityZero(pdpool.Capacity) && pdpool.Contains(leasePrefix) {
				leasesCount[pdpool.GetID()] += 1
				break
			}
		}
	}

	return leasesCount
}

func prefixFromAddressAndPrefixLen(address string, prefixLen uint32) string {
	return address + "/" + strconv.Itoa(int(prefixLen))
}

func setPdPoolLeasesUsedRatio(pdpool *resource.PdPool, leasesCount uint64) {
	if !resource.IsCapacityZero(pdpool.Capacity) && leasesCount != 0 {
		pdpool.UsedCount = leasesCount
		pdpool.UsedRatio = fmt.Sprintf("%.4f", calculateUsedRatio(pdpool.Capacity, leasesCount))
	}
}

func (p *PdPoolService) Get(subnet *resource.Subnet6, pdpoolId string) (*resource.PdPool, error) {
	var pdpools []*resource.PdPool
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		err := tx.Fill(map[string]interface{}{restdb.IDField: pdpoolId}, &pdpools)
		if err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, pdpoolId, pg.Error(err).Error())
		} else if len(pdpools) != 1 {
			return errorno.ErrNotFound(errorno.ErrNamePdPool, pdpoolId)
		}

		if len(subnet.Nodes) != 0 {
			reservations, err = getReservation6sWithPrefixesExists(tx, subnet.GetID())
		}

		return err
	}); err != nil {
		return nil, err
	}

	leasesCount, err := getPdPoolLeasesCount(subnet, pdpools[0], reservations)
	if err != nil {
		log.Warnf("get pdpool %s with subnet6 %s from db failed: %s",
			pdpoolId, subnet.GetID(), err.Error())
	}

	setPdPoolLeasesUsedRatio(pdpools[0], leasesCount)
	return pdpools[0], nil
}

func getPdPoolLeasesCount(subnet *resource.Subnet6, pdpool *resource.PdPool, reservations []*resource.Reservation6) (uint64, error) {
	if resource.IsCapacityZero(pdpool.Capacity) || len(subnet.Nodes) == 0 {
		return 0, nil
	}

	beginAddr, endAddr := pdpool.GetRange()
	var resp *pbdhcpagent.GetLeases6Response
	var err error
	if err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetPool6Leases(ctx,
			&pbdhcpagent.GetPool6LeasesRequest{
				SubnetId:     subnetIDStrToUint64(pdpool.Subnet6),
				BeginAddress: beginAddr,
				EndAddress:   endAddr,
			})
		if err != nil {
			err = errorno.ErrNetworkError(errorno.ErrNameLease, err.Error())
		}
		return err
	}); err != nil {
		return 0, err
	}

	if len(resp.GetLeases()) == 0 {
		return 0, nil
	}

	if len(reservations) == 0 {
		return uint64(len(resp.GetLeases())), nil
	}

	reservationMap := reservationPrefixMapFromReservation6s(reservations)
	var leasesCount uint64
	for _, lease := range resp.GetLeases() {
		if _, ok := reservationMap[prefixFromAddressAndPrefixLen(lease.GetAddress(),
			lease.GetPrefixLen())]; !ok {
			leasesCount += 1
		}
	}

	return leasesCount, nil
}

func (p *PdPoolService) Delete(subnet *resource.Subnet6, pdpool *resource.PdPool) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPdPoolCouldBeDeleted(tx, subnet, pdpool); err != nil {
			return err
		}

		if err := updateSubnet6CapacityWithPdPool(tx, subnet.GetID(),
			subnet.SubCapacityWithString(pdpool.Capacity)); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TablePdPool,
			map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, pdpool.GetID(), pg.Error(err).Error())
		}

		return sendDeletePdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdpool)
	})
}

func checkPdPoolCouldBeDeleted(tx restdb.Transaction, subnet *resource.Subnet6, pdpool *resource.PdPool) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	}

	if err := setPdPoolFromDB(tx, pdpool); err != nil {
		return err
	}

	reservations, err := getReservation6sWithPrefixesExists(tx, subnet.GetID())
	if err != nil {
		return err
	}

	pdpool.Subnet6 = subnet.GetID()
	if leasesCount, err := getPdPoolLeasesCount(subnet, pdpool, reservations); err != nil {
		return err
	} else if leasesCount != 0 {
		return errorno.ErrIPHasBeenAllocated(errorno.ErrNamePdPool, pdpool.GetID())
	}

	return nil
}

func setPdPoolFromDB(tx restdb.Transaction, pdpool *resource.PdPool) error {
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pdpool.GetID()},
		&pdpools); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, pdpool.GetID(), pg.Error(err).Error())
	} else if len(pdpools) == 0 {
		return errorno.ErrNotFound(errorno.ErrNamePdPool, pdpool.GetID())
	}

	pdpool.Subnet6 = pdpools[0].Subnet6
	pdpool.Prefix = pdpools[0].Prefix
	pdpool.PrefixLen = pdpools[0].PrefixLen
	pdpool.PrefixIpnet = pdpools[0].PrefixIpnet
	pdpool.DelegatedLen = pdpools[0].DelegatedLen
	pdpool.Capacity = pdpools[0].Capacity
	return nil
}

func sendDeletePdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.PdPool) error {
	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeletePdPool,
		pdpoolToDeletePdPoolRequest(subnetID, pdpool), nil)
}

func pdpoolToDeletePdPoolRequest(subnetID uint64, pdpool *resource.PdPool) *pbdhcpagent.DeletePdPoolRequest {
	return &pbdhcpagent.DeletePdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}

func (p *PdPoolService) Update(subnetId string, pdpool *resource.PdPool) error {
	if err := resource.CheckCommentValid(pdpool.Comment); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePdPool,
			map[string]interface{}{resource.SqlColumnComment: pdpool.Comment},
			map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, pdpool.GetID(), pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNamePdPool, pdpool.GetID())
		}
		return nil
	})
}

func GetPdPool6sByPrefix(prefix string) ([]*resource.PdPool, error) {
	subnet6, err := GetSubnet6ByPrefix(prefix)
	if err != nil {
		return nil, err
	}

	if pools, err := listPdPools(subnet6); err != nil {
		return nil, err
	} else {
		return pools, nil
	}
}
