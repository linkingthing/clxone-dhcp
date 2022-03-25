package service

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type PdPoolService struct {
}

func NewPdPoolService() *PdPoolService {
	return &PdPoolService{}
}

func (p *PdPoolService) Create(subnet *resource.Subnet6, pdPool *resource.PdPool) error {
	if err := pdPool.Validate(); err != nil {
		return fmt.Errorf("validate pdpool params invalid: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPdPoolCouldBeCreated(tx, subnet, pdPool); err != nil {
			return err
		}

		if err := recalculatePdPoolCapacity(tx, subnet.GetID(), pdPool); err != nil {
			return fmt.Errorf("recalculate pdpool capacity failed: %s", err.Error())
		}

		if err := updateSubnet6CapacityWithPdPool(tx, subnet.GetID(),
			subnet.Capacity+pdPool.Capacity); err != nil {
			return err
		}

		pdPool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pdPool); err != nil {
			return err
		}

		return sendCreatePdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdPool)
	}); err != nil {
		return fmt.Errorf("create pdpool %s failed:%s", pdPool.Prefix, err.Error())
	}

	return nil
}

func checkPdPoolCouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet6, pdPool *resource.PdPool) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	} else if subnet.UseEui64 {
		return fmt.Errorf("subnet6 use EUI64, can not create pdpool")
	}

	if err := checkPrefixBelongsToIpnet(subnet.Ipnet, pdPool.PrefixIpnet,
		pdPool.PrefixLen); err != nil {
		return err
	}

	return checkPdPoolConflictWithSubnet6PdPools(tx, subnet.GetID(), pdPool)
}

func checkPrefixBelongsToIpnet(ipnet, prefixIpnet net.IPNet, prefixLen uint32) error {
	if ones, _ := ipnet.Mask.Size(); uint32(ones) > prefixLen {
		return fmt.Errorf("pdpool %s prefix len %d should bigger than subnet mask len %d",
			prefixIpnet.String(), prefixLen, ones)
	}

	if ipnet.Contains(prefixIpnet.IP) == false {
		return fmt.Errorf("pdpool %s not belongs to subnet6 %s",
			prefixIpnet.String(), ipnet.String())
	}

	return nil
}

func checkPdPoolConflictWithSubnet6PdPools(tx restdb.Transaction, subnetID string, pdPool *resource.PdPool) error {
	if pdPools, err := getPdPoolsWithPrefix(tx, subnetID, pdPool.PrefixIpnet); err != nil {
		return err
	} else if len(pdPools) != 0 {
		return fmt.Errorf("pdpool %s conflict with pdpool %s",
			pdPool.String(), pdPools[0].String())
	} else {
		return nil
	}
}

func getPdPoolsWithPrefix(tx restdb.Transaction, subnetID string, prefix net.IPNet) ([]*resource.PdPool, error) {
	var pdPools []*resource.PdPool
	if err := tx.FillEx(&pdPools,
		"select * from gr_pd_pool where subnet6 = $1 and prefix_ipnet && $2",
		subnetID, prefix); err != nil {
		return nil, fmt.Errorf("get pdpools with subnet6 %s from db failed: %s",
			subnetID, err.Error())
	} else {
		return pdPools, nil
	}
}

func recalculatePdPoolCapacity(tx restdb.Transaction, subnetID string, pdPool *resource.PdPool) error {
	reservations, err := getReservation6sWithPrefixesExists(tx, subnetID)
	if err != nil {
		return err
	}

	reservedPdPools, err := getReservedPdPoolsWithPrefix(tx, subnetID, pdPool.PrefixIpnet)
	if err != nil {
		return err
	}

	recalculatePdPoolCapacityWithReservations(pdPool, reservations)
	recalculatePdPoolCapacityWithReservedPdPools(pdPool, reservedPdPools)
	return nil
}

func getReservation6sWithPrefixesExists(tx restdb.Transaction, subnetID string) ([]*resource.Reservation6, error) {
	var reservations []*resource.Reservation6
	if err := tx.FillEx(&reservations,
		"select * from gr_reservation6 where subnet6 = $1 and prefixes != '{}'",
		subnetID); err != nil {
		return nil, fmt.Errorf("get reservation6s with subnet6 %s from db failed: %s",
			subnetID, err.Error())
	} else {
		return reservations, nil
	}
}

func getReservedPdPoolsWithPrefix(tx restdb.Transaction, subnetID string, prefix net.IPNet) ([]*resource.ReservedPdPool, error) {
	var pdpools []*resource.ReservedPdPool
	if err := tx.FillEx(&pdpools,
		"select * from gr_reserved_pd_pool where subnet6 = $1 and prefix_ipnet && $2",
		subnetID, prefix); err != nil {
		return nil, fmt.Errorf("get reserved pdpools with subnet6 %s from db failed: %s",
			subnetID, err.Error())
	} else {
		return pdpools, nil
	}
}

func recalculatePdPoolCapacityWithReservations(pdpool *resource.PdPool, reservations []*resource.Reservation6) {
	for _, reservation := range reservations {
		for _, prefix := range reservation.Prefixes {
			if pdpool.IntersectPrefix(prefix) {
				pdpool.Capacity -= getPdPoolReservedCountWithPrefix(pdpool, prefix)
			}
		}
	}
}

func recalculatePdPoolCapacityWithReservedPdPools(pdpool *resource.PdPool, reservedPdPools []*resource.ReservedPdPool) {
	for _, reservedPdPool := range reservedPdPools {
		if pdpool.IntersectIpnet(reservedPdPool.PrefixIpnet) {
			pdpool.Capacity -= getPdPoolReservedCount(pdpool, reservedPdPool.PrefixLen)
		}
	}
}

func updateSubnet6CapacityWithPdPool(tx restdb.Transaction, subnetID string, capacity uint64) error {
	if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
		"capacity": capacity,
	}, map[string]interface{}{restdb.IDField: subnetID}); err != nil {
		return fmt.Errorf("update subnet6 %s capacity to db failed: %s",
			subnetID, err.Error())
	} else {
		return nil
	}

}

func sendCreatePdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.PdPool) error {
	nodesForSucceed, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreatePdPool,
		pdPoolToCreatePdPoolRequest(subnetID, pdpool))
	if err != nil {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, kafka.DeletePdPool,
			pdPoolToDeletePdPoolRequest(subnetID, pdpool)); err != nil {
			log.Errorf("create subnet6 %d pdpool %s failed, and rollback it failed: %s",
				subnetID, pdpool.String(), err.Error())
		}
	}

	return err
}

func pdPoolToCreatePdPoolRequest(subnetID uint64, pdPool *resource.PdPool) *pbdhcpagent.CreatePdPoolRequest {
	return &pbdhcpagent.CreatePdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdPool.Prefix,
		PrefixLen:    pdPool.PrefixLen,
		DelegatedLen: pdPool.DelegatedLen,
	}
}

func (p *PdPoolService) List(subnetId string) ([]*resource.PdPool, error) {
	return listPdPools(subnetId)
}

func listPdPools(subnetID string) ([]*resource.PdPool, error) {
	var pdPools []*resource.PdPool
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet6: subnetID,
			resource.SqlOrderBy:       resource.SqlColumnPrefixIpNet}, &pdPools)
		if err != nil {
			return err
		}

		reservations, err = getReservation6sWithPrefixesExists(tx, subnetID)
		return err
	}); err != nil {
		return nil, fmt.Errorf("list pdpools with subnet6 %s failed: %s",
			subnetID, err.Error())
	}

	pdPoolsLeases := loadPdPoolsLeases(subnetID, pdPools, reservations)
	for _, pdPool := range pdPools {
		setPdPoolLeasesUsedRatio(pdPool, pdPoolsLeases[pdPool.GetID()])
	}

	return pdPools, nil
}

func loadPdPoolsLeases(subnetID string, pdPools []*resource.PdPool, reservations []*resource.Reservation6) map[string]uint64 {
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

		for _, pdpool := range pdPools {
			if pdpool.Capacity != 0 && pdpool.Contains(leasePrefix) {
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

func setPdPoolLeasesUsedRatio(pdPool *resource.PdPool, leasesCount uint64) {
	if leasesCount != 0 && pdPool.Capacity != 0 {
		pdPool.UsedCount = leasesCount
		pdPool.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(pdPool.Capacity))
	}
}

func (p *PdPoolService) Get(subnet *resource.Subnet6, pdPoolId string) (*resource.PdPool, error) {
	var pdPools []*resource.PdPool
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{restdb.IDField: pdPoolId}, &pdPools)
		if err != nil {
			return err
		} else if len(pdPools) != 1 {
			return fmt.Errorf("no found pdpool %s with subnet6 %s", pdPoolId, subnet.GetID())
		}

		reservations, err = getReservation6sWithPrefixesExists(tx, subnet.GetID())
		return err
	}); err != nil {
		return nil, fmt.Errorf("get pdpool %s with subnet6 %s from db failed: %s",
			pdPoolId, subnet.GetID(), err.Error())
	}

	leasesCount, err := getPdPoolLeasesCount(pdPools[0], reservations)
	if err != nil {
		log.Warnf("get pdpool %s with subnet6 %s from db failed: %s",
			pdPoolId, subnet.GetID(), err.Error())
	}

	setPdPoolLeasesUsedRatio(pdPools[0], leasesCount)
	return pdPools[0], nil
}

func getPdPoolLeasesCount(pdPool *resource.PdPool, reservations []*resource.Reservation6) (uint64, error) {
	if pdPool.Capacity == 0 {
		return 0, nil
	}

	beginAddr, endAddr := pdPool.GetRange()
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetPool6Leases(context.TODO(),
		&pbdhcpagent.GetPool6LeasesRequest{
			SubnetId:     subnetIDStrToUint64(pdPool.Subnet6),
			BeginAddress: beginAddr,
			EndAddress:   endAddr,
		})

	if err != nil {
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
			lease.GetPrefixLen())]; ok == false {
			leasesCount += 1
		}
	}

	return leasesCount, nil
}

func (p *PdPoolService) Delete(subnet *resource.Subnet6, pdPool *resource.PdPool) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPdPoolCouldBeDeleted(tx, subnet, pdPool); err != nil {
			return err
		}

		if err := updateSubnet6CapacityWithPdPool(tx, subnet.GetID(),
			subnet.Capacity-pdPool.Capacity); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TablePdPool,
			map[string]interface{}{restdb.IDField: pdPool.GetID()}); err != nil {
			return err
		}

		return sendDeletePdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdPool)
	}); err != nil {
		return fmt.Errorf("delete pdpool %s with subnet6 %s failed:%s",
			pdPool.GetID(), subnet.GetID(), err.Error())
	}

	return nil
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
	if leasesCount, err := getPdPoolLeasesCount(pdpool, reservations); err != nil {
		return fmt.Errorf("get pdpool %s leases count failed: %s",
			pdpool.String(), err.Error())
	} else if leasesCount != 0 {
		return fmt.Errorf("can not delete pdpool with %d ips had been allocated",
			leasesCount)
	}

	return nil
}

func setPdPoolFromDB(tx restdb.Transaction, pdPool *resource.PdPool) error {
	var pdPools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pdPool.GetID()},
		&pdPools); err != nil {
		return fmt.Errorf("get pdpool from db failed: %s", err.Error())
	} else if len(pdPools) == 0 {
		return fmt.Errorf("no found pdpool %s", pdPool.GetID())
	}

	pdPool.Subnet6 = pdPools[0].Subnet6
	pdPool.Prefix = pdPools[0].Prefix
	pdPool.PrefixLen = pdPools[0].PrefixLen
	pdPool.PrefixIpnet = pdPools[0].PrefixIpnet
	pdPool.DelegatedLen = pdPools[0].DelegatedLen
	pdPool.Capacity = pdPools[0].Capacity
	return nil
}

func sendDeletePdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdPool *resource.PdPool) error {
	_, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeletePdPool,
		pdPoolToDeletePdPoolRequest(subnetID, pdPool))
	return err
}

func pdPoolToDeletePdPoolRequest(subnetID uint64, pdPool *resource.PdPool) *pbdhcpagent.DeletePdPoolRequest {
	return &pbdhcpagent.DeletePdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdPool.Prefix,
		PrefixLen:    pdPool.PrefixLen,
		DelegatedLen: pdPool.DelegatedLen,
	}
}

func (p *PdPoolService) Update(subnetId string, pdPool *resource.PdPool) error {
	if err := pdPool.Validate(); err != nil {
		return fmt.Errorf("validate pdpool params invalid: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePdPool,
			map[string]interface{}{resource.SqlColumnComment: pdPool.Comment},
			map[string]interface{}{restdb.IDField: pdPool.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found pdpool %s", pdPool.GetID())
		}
		return nil
	}); err != nil {
		return fmt.Errorf("update pdpool %s with subnet6 %s failed: %s",
			pdPool.String(), subnetId, err.Error())
	}

	return nil
}

func GetPdPool6sByPrefix(prefix string) ([]*resource.PdPool, error) {
	subnet6, err := GetSubnet6ByPrefix(prefix)
	if err != nil {
		return nil, err
	}

	if pools, err := listPdPools(subnet6.GetID()); err != nil {
		return nil, err
	} else {
		return pools, nil
	}
}
