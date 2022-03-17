package api

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type PdPoolHandler struct {
}

func NewPdPoolHandler() *PdPoolHandler {
	return &PdPoolHandler{}
}

func (p *PdPoolHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := pdpool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pdpool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPdPoolCouldBeCreated(tx, subnet, pdpool); err != nil {
			return err
		}

		if err := recalculatePdPoolCapacity(tx, subnet.GetID(), pdpool); err != nil {
			return fmt.Errorf("recalculate pdpool capacity failed: %s", err.Error())
		}

		if err := updateSubnet6CapacityWithPdPool(tx, subnet.GetID(),
			subnet.Capacity+pdpool.Capacity); err != nil {
			return err
		}

		pdpool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pdpool); err != nil {
			return err
		}

		return sendCreatePdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdpool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pdpool %s with subnet %s failed: %s",
				pdpool.String(), subnet.GetID(), err.Error()))
	}

	return pdpool, nil
}

func checkPdPoolCouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet6, pdpool *resource.PdPool) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	} else if subnet.UseEui64 {
		return fmt.Errorf("subnet use EUI64, can not create pdpool")
	}

	if err := checkPrefixBelongsToIpnet(subnet.Ipnet, pdpool.PrefixIpnet,
		pdpool.PrefixLen); err != nil {
		return err
	}

	return checkPdPoolConflictWithSubnet6PdPools(tx, subnet.GetID(), pdpool)
}

func checkPrefixBelongsToIpnet(ipnet, prefixIpnet net.IPNet, prefixLen uint32) error {
	if ones, _ := ipnet.Mask.Size(); uint32(ones) > prefixLen {
		return fmt.Errorf("pdpool %s prefix len %d should bigger than subnet mask len %d",
			prefixIpnet.String(), prefixLen, ones)
	}

	if ipnet.Contains(prefixIpnet.IP) == false {
		return fmt.Errorf("pdpool %s not belongs to subnet %s",
			prefixIpnet.String(), ipnet.String())
	}

	return nil
}

func checkPdPoolConflictWithSubnet6PdPools(tx restdb.Transaction, subnetID string, pdpool *resource.PdPool) error {
	if pdpools, err := getPdPoolsWithPrefix(tx, subnetID, pdpool.PrefixIpnet); err != nil {
		return err
	} else if len(pdpools) != 0 {
		return fmt.Errorf("pdpool %s conflict with pdpool %s",
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
		return nil, fmt.Errorf("get pdpools with subnet %s from db failed: %s",
			subnetID, err.Error())
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
		return nil, fmt.Errorf("get reservation6s with subnet %s from db failed: %s",
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
		return nil, fmt.Errorf("get reserved pdpools with subnet %s from db failed: %s",
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
		return fmt.Errorf("update subnet %s capacity to db failed: %s",
			subnetID, err.Error())
	} else {
		return nil
	}

}

func sendCreatePdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.PdPool) error {
	nodesForSucceed, err := sendDHCPCmdWithNodes(false, nodes, dhcpservice.CreatePdPool,
		pdpoolToCreatePdPoolRequest(subnetID, pdpool))
	if err != nil {
		if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, dhcpservice.DeletePdPool,
			pdpoolToDeletePdPoolRequest(subnetID, pdpool)); err != nil {
			log.Errorf("create subnet %d pdpool %s failed, and rollback it failed: %s",
				subnetID, pdpool.String(), err.Error())
		}
	}

	return err
}

func pdpoolToCreatePdPoolRequest(subnetID uint64, pdpool *resource.PdPool) *pbdhcpagent.CreatePdPoolRequest {
	return &pbdhcpagent.CreatePdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}

func (p *PdPoolHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var pdpools []*resource.PdPool
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{
			"subnet6": subnetID, "orderby": "prefix_ipnet"}, &pdpools)
		if err != nil {
			return err
		}

		reservations, err = getReservation6sWithPrefixesExists(tx, subnetID)
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pdpools with subnet %s from db failed: %s",
				subnetID, err.Error()))
	}

	pdpoolsLeases := loadPdPoolsLeases(subnetID, pdpools, reservations)
	for _, pdpool := range pdpools {
		setPdPoolLeasesUsedRatio(pdpool, pdpoolsLeases[pdpool.GetID()])
	}

	return pdpools, nil
}

func loadPdPoolsLeases(subnetID string, pdpools []*resource.PdPool, reservations []*resource.Reservation6) map[string]uint64 {
	resp, err := getSubnet6Leases(subnetIDStrToUint64(subnetID))
	if err != nil {
		log.Warnf("get subnet %s leases failed: %s", subnetID, err.Error())
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

func setPdPoolLeasesUsedRatio(pdpool *resource.PdPool, leasesCount uint64) {
	if leasesCount != 0 && pdpool.Capacity != 0 {
		pdpool.UsedCount = leasesCount
		pdpool.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(pdpool.Capacity))
	}
}

func (p *PdPoolHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pdpoolID := ctx.Resource.GetID()
	var pdpools []*resource.PdPool
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{restdb.IDField: pdpoolID}, &pdpools)
		if err != nil {
			return err
		} else if len(pdpools) != 1 {
			return fmt.Errorf("no found pdpool %s with subnet %s", pdpoolID, subnetID)
		}

		reservations, err = getReservation6sWithPrefixesExists(tx, subnetID)
		return err
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pdpool %s with subnet %s from db failed: %s",
				pdpoolID, subnetID, err.Error()))
	}

	leasesCount, err := getPdPoolLeasesCount(pdpools[0], reservations)
	if err != nil {
		log.Warnf("get pdpool %s with subnet %s from db failed: %s",
			pdpoolID, subnetID, err.Error())
	}

	setPdPoolLeasesUsedRatio(pdpools[0], leasesCount)
	return pdpools[0], nil
}

func getPdPoolLeasesCount(pdpool *resource.PdPool, reservations []*resource.Reservation6) (uint64, error) {
	if pdpool.Capacity == 0 {
		return 0, nil
	}

	beginAddr, endAddr := pdpool.GetRange()
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetPool6Leases(context.TODO(),
		&pbdhcpagent.GetPool6LeasesRequest{
			SubnetId:     subnetIDStrToUint64(pdpool.Subnet6),
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

func (p *PdPoolHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPdPoolCouldBeDeleted(tx, subnet, pdpool); err != nil {
			return err
		}

		if err := updateSubnet6CapacityWithPdPool(tx, subnet.GetID(),
			subnet.Capacity-pdpool.Capacity); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TablePdPool,
			map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return err
		}

		return sendDeletePdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdpool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pdpool %s with subnet %s failed: %s",
				pdpool.String(), subnet.GetID(), err.Error()))
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

func setPdPoolFromDB(tx restdb.Transaction, pdpool *resource.PdPool) error {
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pdpool.GetID()},
		&pdpools); err != nil {
		return fmt.Errorf("get pdpool from db failed: %s", err.Error())
	} else if len(pdpools) == 0 {
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

func sendDeletePdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.PdPool) error {
	_, err := sendDHCPCmdWithNodes(false, nodes, dhcpservice.DeletePdPool,
		pdpoolToDeletePdPoolRequest(subnetID, pdpool))
	return err
}

func pdpoolToDeletePdPoolRequest(subnetID uint64, pdpool *resource.PdPool) *pbdhcpagent.DeletePdPoolRequest {
	return &pbdhcpagent.DeletePdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}

func (p *PdPoolHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := pdpool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pdpool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePdPool, map[string]interface{}{
			"comment": pdpool.Comment,
		}, map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found pdpool %s", pdpool.GetID())
		}

		return nil
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update pdpool %s with subnet %s failed: %s",
				pdpool.String(), ctx.Resource.GetParent().GetID(), err.Error()))
	}

	return pdpool, nil
}
