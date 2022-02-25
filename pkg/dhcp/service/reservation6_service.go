package service

import (
	"context"
	"fmt"
	"net"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Reservation6Service struct {
}

func NewReservation6Service() *Reservation6Service {
	return &Reservation6Service{}
}

func (r *Reservation6Service) Create(subnet *resource.Subnet6, reservation *resource.Reservation6) (restresource.Resource, error) {
	if rets, err := BatchCreateReservation6s(subnet, []*resource.Reservation6{reservation}); err != nil || len(rets) == 0 {
		return nil, err
	} else {
		return rets[0], nil
	}
}

func BatchCreateReservation6s(subnet *resource.Subnet6, reservations []*resource.Reservation6) ([]restresource.Resource, error) {
	rets := make([]restresource.Resource, 0)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, v := range reservations {
			if err := execCreateReservation6s(tx, subnet, v); err != nil {
				return err
			}
			rets = append(rets, v)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return rets, nil
}

func execCreateReservation6s(tx restdb.Transaction, subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	} else if subnet.UseEui64 {
		return fmt.Errorf("subnet use EUI64, can not create reservation6")
	}

	if err := checkReservation6InUsed(tx, subnet.GetID(), reservation); err != nil {
		return err
	}

	if err := checkReservation6BelongsToIpnet(subnet.Ipnet, reservation); err != nil {
		return err
	}

	if err := checkReservation6ConflictWithReservedPools(tx, subnet.GetID(),
		reservation); err != nil {
		return err
	}

	conflictPdPool, conflictPool, err := checkReservation6ConflictWithSubnet6Pools(
		tx, subnet.GetID(), reservation)
	if err != nil {
		return err
	}

	if conflictPool != nil {
		if _, err := tx.Update(resource.TablePool6, map[string]interface{}{
			resource.SqlColumnCapacity: conflictPool.Capacity - reservation.Capacity,
		}, map[string]interface{}{restdb.IDField: conflictPool.GetID()}); err != nil {
			return fmt.Errorf("update pool %s capacity to db failed: %s",
				conflictPool.String(), err.Error())
		}
	} else if conflictPdPool != nil {
		if _, err := tx.Update(resource.TablePdPool, map[string]interface{}{
			resource.SqlColumnCapacity: conflictPdPool.Capacity - reservation.Capacity,
		}, map[string]interface{}{restdb.IDField: conflictPdPool.GetID()}); err != nil {
			return fmt.Errorf("update pdpool %s capacity to db failed: %s",
				conflictPdPool.String(), err.Error())
		}
	} else {
		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			resource.SqlColumnCapacity: subnet.Capacity + reservation.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s",
				subnet.GetID(), err.Error())
		}
	}

	reservation.Subnet6 = subnet.GetID()
	if _, err := tx.Insert(reservation); err != nil {
		return err
	}

	return sendCreateReservation6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
		reservation)
}

func checkReservation6InUsed(tx restdb.Transaction, subnetId string, reservation *resource.Reservation6) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&reservations); err != nil {
		return fmt.Errorf("get subnet6 %s reservation6 failed: %s", subnetId, err.Error())
	}

	for _, reservation_ := range reservations {
		if reservation_.CheckConflictWithAnother(reservation) {
			return fmt.Errorf("reservation6 %s conflict with exists reservation6 %s",
				reservation.String(), reservation_.String())
		}
	}

	return nil
}

func checkReservation6BelongsToIpnet(ipnet net.IPNet, reservation *resource.Reservation6) error {
	for _, ip := range reservation.Ips {
		if checkIPsBelongsToIpnet(ipnet, ip) == false {
			return fmt.Errorf("reservation %s ip %s not belong to subnet %s",
				reservation.String(), ip.String(), ipnet.String())
		}
	}

	subnetMaskLen, _ := ipnet.Mask.Size()
	for _, prefix := range reservation.Prefixes {
		if ip, ipnet_, _ := net.ParseCIDR(prefix); ipnet.Contains(ip) == false {
			return fmt.Errorf("reservation %s prefix %s not belong to subnet %s",
				reservation.String(), prefix, ipnet.String())
		} else if ones, _ := ipnet_.Mask.Size(); ones <= subnetMaskLen {
			return fmt.Errorf("reservation %s prefix %s len %d less than subnet %d",
				reservation.String(), prefix, ones, subnetMaskLen)
		}

	}

	return nil
}

func checkReservation6ConflictWithReservedPools(tx restdb.Transaction, subnetId string, reservation *resource.Reservation6) error {
	var reservedpools []*resource.ReservedPool6
	if len(reservation.IpAddresses) != 0 {
		if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
			&reservedpools); err != nil {
			return fmt.Errorf("get subnet %s reserved pool6 from db failed: %s",
				subnetId, err.Error())
		}
	}

	var reservedpdpools []*resource.ReservedPdPool
	if len(reservation.Prefixes) != 0 {
		if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
			&reservedpdpools); err != nil {
			return fmt.Errorf("get subnet %s reserved pdpool from db failed: %s",
				subnetId, err.Error())
		}
	}

	for _, ipAddress := range reservation.IpAddresses {
		for _, reservedpool := range reservedpools {
			if reservedpool.Contains(ipAddress) {
				return fmt.Errorf("reservation %s ip %s conflict with reserved pool6 %s",
					reservation.String(), ipAddress, reservedpool.String())
			}
		}
	}

	for _, prefix := range reservation.Prefixes {
		for _, reservedpdpool := range reservedpdpools {
			if reservedpdpool.Contains(prefix) {
				return fmt.Errorf("reservation %s prefix %s conflict with reserved pdpool %s",
					reservation.String(), prefix, reservedpdpool.String())
			}
		}
	}

	return nil
}

func checkReservation6ConflictWithSubnet6Pools(tx restdb.Transaction, subnetID string, reservation6 *resource.Reservation6) (*resource.PdPool, *resource.Pool6, error) {
	if conflictPool, err := checkIpsConflictWithSubnet6Pool6s(tx,
		subnetID, reservation6.IpAddresses); err != nil {
		return nil, nil, err
	} else if conflictPool != nil {
		return nil, conflictPool, nil
	}

	conflictPdPool, err := checkPrefixesConflictWithSubnetPdPool(tx, subnetID,
		reservation6.Prefixes)
	return conflictPdPool, nil, err
}

func checkIpsConflictWithSubnet6Pool6s(tx restdb.Transaction, subnetID string, ips []string) (*resource.Pool6, error) {
	if len(ips) == 0 {
		return nil, nil
	}

	var pools []*resource.Pool6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetID}, &pools); err != nil {
		return nil, fmt.Errorf("get pools with subnet %s from db failed: %s",
			subnetID, err.Error())
	}

	for _, ip := range ips {
		for _, pool := range pools {
			if pool.Contains(ip) {
				return pool, nil
			}
		}
	}

	return nil, nil
}

func checkPrefixesConflictWithSubnetPdPool(tx restdb.Transaction, subnetID string, prefixes []string) (*resource.PdPool, error) {
	if len(prefixes) == 0 {
		return nil, nil
	}

	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetID}, &pdpools); err != nil {
		return nil, fmt.Errorf("get pdpools with subnet %s from db failed: %s",
			subnetID, err.Error())
	}

	for _, prefix := range prefixes {
		for _, pdpool := range pdpools {
			if pdpool.Contains(prefix) {
				return pdpool, nil
			}
		}
	}

	return nil, nil
}

func sendCreateReservation6CmdToDHCPAgent(subnetID uint64, nodes []string, reservation *resource.Reservation6) error {
	nodesForSucceed, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreateReservation6,
		reservation6ToCreateReservation6Request(subnetID, reservation))
	if err != nil {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, kafka.DeleteReservation6,
			reservation6ToDeleteReservation6Request(subnetID, reservation)); err != nil {
			log.Errorf("create subnet %d reservation6 %s failed, and rollback it failed: %s",
				subnetID, reservation.String(), err.Error())
		}
	}

	return err
}

func reservation6ToCreateReservation6Request(subnetID uint64, reservation *resource.Reservation6) *pbdhcpagent.CreateReservation6Request {
	return &pbdhcpagent.CreateReservation6Request{
		SubnetId:    subnetID,
		HwAddress:   reservation.HwAddress,
		Duid:        reservation.Duid,
		IpAddresses: reservation.IpAddresses,
		Prefixes:    reservation.Prefixes,
	}
}

func (r *Reservation6Service) List(subnetID string) (interface{}, error) {
	return GetReservation6List(subnetID)
}

func GetReservation6List(subnetID string) ([]*resource.Reservation6, error) {
	var reservations []*resource.Reservation6
	if err := db.GetResources(map[string]interface{}{
		resource.SqlColumnSubnet6: subnetID,
		util.SqlOrderBy:           "duid, hw_address"}, &reservations); err != nil {
		return nil, fmt.Errorf("list reservations with subnet %s from db failed :%s", subnetID, err.Error())
	}

	leasesCount := getReservation6sLeasesCount(subnetIDStrToUint64(subnetID), reservations)
	for _, reservation := range reservations {
		setReservation6LeasesUsedRatio(reservation, leasesCount[reservation.GetID()])
	}

	return reservations, nil
}

func getReservation6sLeasesCount(subnetId uint64, reservations []*resource.Reservation6) map[string]uint64 {
	resp, err := getSubnet6Leases(subnetId)
	if err != nil {
		log.Warnf("get subnet %s leases failed: %s", subnetId, err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := make(map[string]*resource.Reservation6)
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress] = reservation
		}
	}

	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if reservation, ok := reservationMap[lease.GetAddress()]; ok {
			count := leasesCount[reservation.GetID()]
			if (len(reservation.Duid) != 0 && reservation.Duid == lease.GetDuid()) ||
				(len(reservation.HwAddress) != 0 &&
					reservation.HwAddress == lease.GetHwAddress()) {
				count += 1
			}
			leasesCount[reservation.GetID()] = count
		}
	}

	return leasesCount
}

func reservationMapFromReservation6s(reservations []*resource.Reservation6) map[string]struct{} {
	reservationMap := make(map[string]struct{})
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress] = struct{}{}
		}
	}

	return reservationMap
}

func (r *Reservation6Service) Get(subnetID, reservationID string) (restresource.Resource, error) {
	var reservations []*resource.Reservation6
	reservationInterface, err := restdb.GetResourceWithID(db.GetDB(), reservationID,
		&reservations)
	if err != nil {
		return nil, fmt.Errorf("get reservation %s with subnetID %s from db failed: %s",
			reservationID, subnetID, err.Error())
	}

	reservation := reservationInterface.(*resource.Reservation6)
	if leasesCount, err := getReservation6LeasesCount(reservation); err != nil {
		log.Warnf("get reservation %s with subnet %s leases used ratio failed: %s",
			reservation.String(), subnetID, err.Error())
	} else {
		setReservation6LeasesUsedRatio(reservation, leasesCount)
	}

	return reservation, nil
}

func setReservation6LeasesUsedRatio(reservation *resource.Reservation6, leasesCount uint64) {
	if leasesCount != 0 {
		reservation.UsedCount = leasesCount
		reservation.UsedRatio = fmt.Sprintf("%.4f",
			float64(leasesCount)/float64(reservation.Capacity))
	}
}

func getReservation6LeasesCount(reservation *resource.Reservation6) (uint64, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetReservation6LeasesCount(
		context.TODO(), &pbdhcpagent.GetReservation6LeasesCountRequest{
			SubnetId:    subnetIDStrToUint64(reservation.Subnet6),
			HwAddress:   reservation.HwAddress,
			Duid:        reservation.Duid,
			IpAddresses: reservation.IpAddresses,
			Prefixes:    reservation.Prefixes,
		})

	return resp.GetLeasesCount(), err
}

func (r *Reservation6Service) Delete(subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservation6FromDB(tx, reservation); err != nil {
			return err
		}

		if leasesCount, err := getReservation6LeasesCount(reservation); err != nil {
			return fmt.Errorf("get reservation %s leases count failed: %s",
				reservation.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("cannot delete reservation with %d ips had been allocated",
				leasesCount)
		}

		conflictPdPool, conflictPool, err := checkReservation6ConflictWithSubnet6Pools(
			tx, subnet.GetID(), reservation)
		if err != nil {
			return err
		} else if conflictPool != nil {
			if _, err := tx.Update(resource.TablePool6, map[string]interface{}{
				resource.SqlColumnCapacity: conflictPool.Capacity + reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: conflictPool.GetID()}); err != nil {
				return fmt.Errorf("update pool %s capacity to db failed: %s",
					conflictPool.GetID(), err.Error())
			}
		} else if conflictPdPool != nil {
			if _, err := tx.Update(resource.TablePdPool, map[string]interface{}{
				resource.SqlColumnCapacity: conflictPdPool.Capacity + reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: conflictPdPool.GetID()}); err != nil {
				return fmt.Errorf("update pdpool %s capacity to db failed: %s",
					conflictPool.GetID(), err.Error())
			}
		} else {
			if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
				resource.SqlColumnCapacity: subnet.Capacity - reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
				return fmt.Errorf("update subnet %s capacity to db failed: %s",
					subnet.GetID(), err.Error())
			}
		}

		if _, err := tx.Delete(resource.TableReservation6,
			map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservation6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservation)
	}); err != nil {
		return err
	}

	return nil
}

func setReservation6FromDB(tx restdb.Transaction, reservation *resource.Reservation6) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: reservation.GetID()},
		&reservations); err != nil {
		return err
	}

	if len(reservations) == 0 {
		return fmt.Errorf("no found reservation %s", reservation.GetID())
	}

	reservation.Subnet6 = reservations[0].Subnet6
	reservation.HwAddress = reservations[0].HwAddress
	reservation.Duid = reservations[0].Duid
	reservation.IpAddresses = reservations[0].IpAddresses
	reservation.Ips = reservations[0].Ips
	reservation.Prefixes = reservations[0].Prefixes
	reservation.Capacity = reservations[0].Capacity
	return nil
}

func sendDeleteReservation6CmdToDHCPAgent(subnetID uint64, nodes []string, reservation *resource.Reservation6) error {
	_, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeleteReservation6,
		reservation6ToDeleteReservation6Request(subnetID, reservation))
	return err
}

func reservation6ToDeleteReservation6Request(subnetID uint64, reservation *resource.Reservation6) *pbdhcpagent.DeleteReservation6Request {
	return &pbdhcpagent.DeleteReservation6Request{
		SubnetId:  subnetID,
		HwAddress: reservation.HwAddress,
		Duid:      reservation.Duid,
	}
}

func (r *Reservation6Service) Update(reservation *resource.Reservation6) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservation6, map[string]interface{}{
			util.SqlColumnsComment: reservation.Comment,
		}, map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found reservation6 %s", reservation.GetID())
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return reservation, nil
}
