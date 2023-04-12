package service

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"strings"

	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
)

type Reservation6Service struct {
}

func NewReservation6Service() *Reservation6Service {
	return &Reservation6Service{}
}

func (r *Reservation6Service) Create(subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	if err := reservation.Validate(); err != nil {
		return fmt.Errorf("validate reservation6 params invalid: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservation6CouldBeCreated(tx, subnet, reservation); err != nil {
			return err
		}

		if err := updateSubnet6AndPoolsCapacityWithReservation6(tx, subnet,
			reservation, true); err != nil {
			return err
		}

		reservation.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(reservation); err != nil {
			return pg.Error(err)
		}

		return sendCreateReservation6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservation)
	}); err != nil {
		return fmt.Errorf("create reservation6 %s failed: %s",
			reservation.String(), err.Error())
	}

	return nil
}

func checkReservation6CouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	} else if subnet.UseEui64 {
		return fmt.Errorf("subnet6 use EUI64, can not create reservation6")
	} else if subnet.UseAddressCode && len(reservation.Prefixes) != 0 {
		return fmt.Errorf("subnet6 use address code, can not create reservation6 prefixes")
	}

	if err := checkReservation6BelongsToIpnet(subnet.Ipnet, reservation); err != nil {
		return err
	}

	if err := checkReservation6InUsed(tx, subnet.GetID(), reservation); err != nil {
		return err
	}

	return checkReservation6ConflictWithPools(tx, subnet.GetID(), reservation)
}

func checkReservation6BelongsToIpnet(ipnet net.IPNet, reservation *resource.Reservation6) error {
	subnetMaskLen, _ := ipnet.Mask.Size()
	if subnetMaskLen < 64 && len(reservation.Ips) != 0 {
		return fmt.Errorf("subnet6 %s mask less than 64, can`t create reservation6 ips %v",
			ipnet.String(), reservation.IpAddresses)
	}

	if subnetMaskLen > 64 && len(reservation.Ipnets) != 0 {
		return fmt.Errorf("subnet6 %s mask bigger than 64, can`t create reservation6 prefixes %v",
			ipnet.String(), reservation.Prefixes)
	}

	for _, ip := range reservation.Ips {
		if !ipnet.Contains(ip) {
			return fmt.Errorf("reservation6 %s ip %s not belong to subnet6 %s",
				reservation.String(), ip.String(), ipnet.String())
		}
	}

	for _, ipnet_ := range reservation.Ipnets {
		if !ipnet.Contains(ipnet_.IP) {
			return fmt.Errorf("reservation6 %s prefix %s not belong to subnet6 %s",
				reservation.String(), ipnet_.String(), ipnet.String())
		} else if ones, _ := ipnet_.Mask.Size(); ones <= subnetMaskLen {
			return fmt.Errorf("reservation6 %s prefix %s len %d is not bigger than subnet6 %d",
				reservation.String(), ipnet_.String(), ones, subnetMaskLen)
		}
	}

	return nil
}

func checkReservation6InUsed(tx restdb.Transaction, subnetId string, reservation *resource.Reservation6) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&reservations); err != nil {
		return fmt.Errorf("get subnet6 %s reservation6 failed: %s", subnetId, pg.Error(err).Error())
	}

	for _, reservation_ := range reservations {
		if reservation_.CheckConflictWithAnother(reservation) {
			return fmt.Errorf("reservation6 %s %s conflict with exists reservation6 %s %s",
				reservation.String(), reservation.AddrString(),
				reservation_.String(), reservation_.AddrString())
		}
	}

	return nil
}

func checkReservation6ConflictWithPools(tx restdb.Transaction, subnetId string, reservation *resource.Reservation6) error {
	var reservedpools []*resource.ReservedPool6
	if len(reservation.IpAddresses) != 0 {
		if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
			&reservedpools); err != nil {
			return fmt.Errorf("get subnet6 %s reserved pool6 from db failed: %s",
				subnetId, pg.Error(err).Error())
		}
	}

	var reservedpdpools []*resource.ReservedPdPool
	if len(reservation.Prefixes) != 0 {
		if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
			&reservedpdpools); err != nil {
			return fmt.Errorf("get subnet6 %s reserved pdpool from db failed: %s",
				subnetId, pg.Error(err).Error())
		}
	}

	for _, ip := range reservation.Ips {
		for _, reservedpool := range reservedpools {
			if reservedpool.ContainsIp(ip) {
				return fmt.Errorf("reservation6 %s ip %s conflict with reserved pool6 %s",
					reservation.String(), ip.String(), reservedpool.String())
			}
		}
	}

	for _, ipnet := range reservation.Ipnets {
		for _, reservedpdpool := range reservedpdpools {
			if reservedpdpool.IntersectIpnet(ipnet) {
				return fmt.Errorf(
					"reservation6 %s prefix %s conflict with reserved pdpool %s",
					reservation.String(), ipnet.String(), reservedpdpool.String())
			}
		}
	}

	return nil
}

func updateSubnet6AndPoolsCapacityWithReservation6(tx restdb.Transaction, subnet *resource.Subnet6, reservation *resource.Reservation6, isCreate bool) error {
	affectedPools, affectedPdPools, err := recalculatePoolsCapacityWithReservation6(
		tx, subnet, reservation, isCreate)
	if err != nil {
		return err
	}

	if !subnet.UseAddressCode {
		if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
			resource.SqlColumnCapacity: subnet.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet6 %s capacity to db failed: %s",
				subnet.GetID(), pg.Error(err).Error())
		}
	}

	for affectedPoolId, capacity := range affectedPools {
		if _, err := tx.Update(resource.TablePool6, map[string]interface{}{
			resource.SqlColumnCapacity: capacity,
		}, map[string]interface{}{restdb.IDField: affectedPoolId}); err != nil {
			return fmt.Errorf("update pool6 %s capacity to db failed: %s",
				affectedPoolId, pg.Error(err).Error())
		}
	}

	for affectedPdPoolId, capacity := range affectedPdPools {
		if _, err := tx.Update(resource.TablePdPool, map[string]interface{}{
			resource.SqlColumnCapacity: capacity,
		}, map[string]interface{}{restdb.IDField: affectedPdPoolId}); err != nil {
			return fmt.Errorf("update pdpool %s capacity to db failed: %s",
				affectedPdPoolId, pg.Error(err).Error())
		}
	}

	return nil
}

func recalculatePoolsCapacityWithReservation6(tx restdb.Transaction, subnet *resource.Subnet6, reservation6 *resource.Reservation6, isCreate bool) (map[string]string, map[string]string, error) {
	if affectedPool6s, err := recalculatePool6sCapacityWithIps(tx, subnet,
		reservation6.Ips, isCreate); err != nil {
		return nil, nil, err
	} else if len(affectedPool6s) != 0 {
		return affectedPool6s, nil, nil
	}

	affectedPdPools, err := recalculatePdPoolsCapacityWithPrefixes(tx,
		subnet, reservation6.Ipnets, isCreate)
	return nil, affectedPdPools, err
}

func recalculatePool6sCapacityWithIps(tx restdb.Transaction, subnet *resource.Subnet6, ips []net.IP, isCreate bool) (map[string]string, error) {
	if len(ips) == 0 {
		return nil, nil
	}

	var pools []*resource.Pool6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet.GetID()},
		&pools); err != nil {
		return nil, fmt.Errorf("get pool6s with subnet6 %s from db failed: %s",
			subnet.GetID(), pg.Error(err).Error())
	}

	affectedPool6s := make(map[string]string)
	unreservedCount := new(big.Int)
	for _, ip := range ips {
		reserved := false
		for _, pool := range pools {
			if pool.ContainsIp(ip) {
				reserved = true
				capacity, ok := affectedPool6s[pool.GetID()]
				if !ok {
					capacity = pool.Capacity
				}

				if isCreate {
					affectedPool6s[pool.GetID()] = resource.SubCapacityWithBigInt(capacity, big.NewInt(1))
				} else {
					affectedPool6s[pool.GetID()] = resource.AddCapacityWithBigInt(capacity, big.NewInt(1))
				}

				break
			}
		}

		if !subnet.UseAddressCode && !reserved {
			unreservedCount.Add(unreservedCount, big.NewInt(1))
		}
	}

	if !subnet.UseAddressCode {
		if isCreate {
			subnet.AddCapacityWithBigInt(unreservedCount)
		} else {
			subnet.SubCapacityWithBigInt(unreservedCount)
		}
	}

	return affectedPool6s, nil
}

func recalculatePdPoolsCapacityWithPrefixes(tx restdb.Transaction, subnet *resource.Subnet6, ipnets []net.IPNet, isCreate bool) (map[string]string, error) {
	if subnet.UseAddressCode || len(ipnets) == 0 {
		return nil, nil
	}

	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet.GetID()},
		&pdpools); err != nil {
		return nil, fmt.Errorf("get pdpools with subnet6 %s from db failed: %s",
			subnet.GetID(), pg.Error(err).Error())
	}

	affectedPdPools := make(map[string]string)
	unreservedCount := new(big.Int)
	for _, ipnet := range ipnets {
		unreservedCount.Add(unreservedCount, big.NewInt(1))
		for _, pdpool := range pdpools {
			if pdpool.IntersectIpnet(ipnet) {
				capacity, ok := affectedPdPools[pdpool.GetID()]
				if !ok {
					capacity = pdpool.Capacity
				}

				reservedCount := getPdPoolReservedCountWithPrefix(pdpool, ipnet)
				unreservedCount.Sub(unreservedCount, reservedCount)
				if isCreate {
					affectedPdPools[pdpool.GetID()] = resource.SubCapacityWithBigInt(capacity, reservedCount)
				} else {
					affectedPdPools[pdpool.GetID()] = resource.AddCapacityWithBigInt(capacity, reservedCount)
				}

				break
			}
		}
	}

	if isCreate {
		subnet.AddCapacityWithBigInt(unreservedCount)
	} else {
		subnet.SubCapacityWithBigInt(unreservedCount)
	}

	return affectedPdPools, nil
}

func getPdPoolReservedCountWithPrefix(pdpool *resource.PdPool, ipnet net.IPNet) *big.Int {
	prefixLen, _ := ipnet.Mask.Size()
	return getPdPoolReservedCount(pdpool, uint32(prefixLen))
}

func getPdPoolReservedCount(pdpool *resource.PdPool, prefixLen uint32) *big.Int {
	if prefixLen <= pdpool.PrefixLen {
		return new(big.Int).Lsh(big.NewInt(1), uint(pdpool.DelegatedLen-pdpool.PrefixLen))
	} else if prefixLen >= pdpool.DelegatedLen {
		return big.NewInt(1)
	} else {
		return new(big.Int).Lsh(big.NewInt(1), uint(pdpool.DelegatedLen-prefixLen))
	}
}

func sendCreateReservation6CmdToDHCPAgent(subnetID uint64, nodes []string, reservation *resource.Reservation6) error {
	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreateReservation6,
		reservation6ToCreateReservation6Request(subnetID, reservation),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservation6,
				reservation6ToDeleteReservation6Request(subnetID, reservation)); err != nil {
				log.Errorf("create subnet6 %d reservation6 %s failed, rollback with nodes %v failed: %s",
					subnetID, reservation.String(), nodesForSucceed, err.Error())
			}
		})
}

func reservation6ToCreateReservation6Request(subnetID uint64, reservation *resource.Reservation6) *pbdhcpagent.CreateReservation6Request {
	return &pbdhcpagent.CreateReservation6Request{
		SubnetId:    subnetID,
		HwAddress:   reservation.HwAddress,
		Duid:        reservation.Duid,
		Hostname:    reservation.Hostname,
		IpAddresses: reservation.IpAddresses,
		Prefixes:    reservation.Prefixes,
	}
}

func (r *Reservation6Service) List(subnet *resource.Subnet6) ([]*resource.Reservation6, error) {
	return listReservation6s(subnet)
}

func listReservation6s(subnet *resource.Subnet6) ([]*resource.Reservation6, error) {
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet6: subnet.GetID(),
			resource.SqlOrderBy:       "ips, ipnets"}, &reservations)
	}); err != nil {
		return nil, fmt.Errorf("list reservation6s with subnet6 %s from db failed: %s",
			subnet.GetID(), pg.Error(err).Error())
	}

	if len(subnet.Nodes) != 0 {
		leasesCount := getReservation6sLeasesCount(subnetIDStrToUint64(subnet.GetID()), reservations)
		for _, reservation := range reservations {
			setReservation6LeasesUsedRatio(reservation, leasesCount[reservation.GetID()])
		}
	}

	return reservations, nil
}

func getReservation6sLeasesCount(subnetId uint64, reservations []*resource.Reservation6) map[string]uint64 {
	resp, err := getSubnet6Leases(subnetId)
	if err != nil {
		log.Warnf("get subnet6 %s leases failed: %s", subnetId, err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := make(map[string]*resource.Reservation6)
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress+"/128"] = reservation
		}

		for _, prefix := range reservation.Prefixes {
			reservationMap[prefix] = reservation
		}
	}

	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if reservation, ok := reservationMap[prefixFromAddressAndPrefixLen(lease.GetAddress(),
			lease.GetPrefixLen())]; ok &&
			(reservation.Duid == "" || reservation.Duid == lease.GetDuid()) &&
			(reservation.HwAddress == "" || reservation.HwAddress == lease.GetHwAddress()) &&
			(reservation.Hostname == "" || reservation.Hostname == lease.GetHostname()) {
			leasesCount[reservation.GetID()] += 1
		}
	}

	return leasesCount
}

func reservationMapFromReservation6s(reservations []*resource.Reservation6) map[string]struct{} {
	reservationMap := make(map[string]struct{})
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress+"/128"] = struct{}{}
		}

		for _, prefix := range reservation.Prefixes {
			reservationMap[prefix] = struct{}{}
		}
	}

	return reservationMap
}

func reservationIpMapFromReservation6s(reservations []*resource.Reservation6) map[string]struct{} {
	reservationMap := make(map[string]struct{})
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress] = struct{}{}
		}
	}

	return reservationMap
}

func reservationPrefixMapFromReservation6s(reservations []*resource.Reservation6) map[string]struct{} {
	reservationMap := make(map[string]struct{})
	for _, reservation := range reservations {
		for _, prefix := range reservation.Prefixes {
			reservationMap[prefix] = struct{}{}
		}
	}

	return reservationMap
}

func (r *Reservation6Service) Get(subnet *resource.Subnet6, reservationID string) (*resource.Reservation6, error) {
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{restdb.IDField: reservationID}, &reservations)
	}); err != nil {
		return nil, fmt.Errorf("get reservation6 %s with subnet6 %s from db failed: %s",
			reservationID, subnet.GetID(), pg.Error(err).Error())
	} else if len(reservations) == 0 {
		return nil, fmt.Errorf("no found reservation6 %s with subnet6 %s",
			reservationID, subnet.GetID())
	}

	if leasesCount, err := getReservation6LeasesCount(subnet, reservations[0]); err != nil {
		log.Warnf("get reservation6 %s with subnet6 %s leases used ratio failed: %s",
			reservations[0].String(), subnet.GetID(), err.Error())
	} else {
		setReservation6LeasesUsedRatio(reservations[0], leasesCount)
	}

	return reservations[0], nil
}

func setReservation6LeasesUsedRatio(reservation *resource.Reservation6, leasesCount uint64) {
	if leasesCount != 0 {
		reservation.UsedCount = leasesCount
		reservation.UsedRatio = fmt.Sprintf("%.4f", calculateUsedRatio(reservation.Capacity, leasesCount))
	}
}

func getReservation6LeasesCount(subnet *resource.Subnet6, reservation *resource.Reservation6) (uint64, error) {
	if len(subnet.Nodes) == 0 {
		return 0, nil
	}

	var err error
	var resp *pbdhcpagent.GetLeasesCountResponse
	if err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetReservation6LeasesCount(
			ctx, &pbdhcpagent.GetReservation6LeasesCountRequest{
				SubnetId:    subnetIDStrToUint64(reservation.Subnet6),
				HwAddress:   strings.ToLower(reservation.HwAddress),
				Duid:        reservation.Duid,
				Hostname:    reservation.Hostname,
				IpAddresses: reservation.IpAddresses,
				Prefixes:    reservation.Prefixes,
			})
		return err
	}); err != nil {
		return 0, err
	}

	return resp.GetLeasesCount(), err
}

func (r *Reservation6Service) Delete(subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservation6CouldBeDeleted(tx, subnet, reservation); err != nil {
			return err
		}

		if err := updateSubnet6AndPoolsCapacityWithReservation6(tx, subnet,
			reservation, false); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableReservation6,
			map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return pg.Error(err)
		}

		return sendDeleteReservation6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservation)
	}); err != nil {
		return fmt.Errorf("delete reservation6 %s with subnet6 %s failed: %s",
			reservation.String(), subnet.GetID(), err.Error())
	}

	return nil
}

func checkReservation6CouldBeDeleted(tx restdb.Transaction, subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	}

	if err := setReservation6FromDB(tx, reservation); err != nil {
		return err
	}

	if leasesCount, err := getReservation6LeasesCount(subnet, reservation); err != nil {
		return fmt.Errorf("get reservation6 %s leases count failed: %s",
			reservation.String(), err.Error())
	} else if leasesCount != 0 {
		return fmt.Errorf("cannot delete reservation6 with %d ips had been allocated",
			leasesCount)
	}

	return nil
}

func setReservation6FromDB(tx restdb.Transaction, reservation *resource.Reservation6) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: reservation.GetID()},
		&reservations); err != nil {
		return pg.Error(err)
	} else if len(reservations) == 0 {
		return fmt.Errorf("no found reservation6 %s", reservation.GetID())
	}

	reservation.Subnet6 = reservations[0].Subnet6
	reservation.HwAddress = reservations[0].HwAddress
	reservation.Duid = reservations[0].Duid
	reservation.Hostname = reservations[0].Hostname
	reservation.IpAddresses = reservations[0].IpAddresses
	reservation.Ips = reservations[0].Ips
	reservation.Prefixes = reservations[0].Prefixes
	reservation.Ipnets = reservations[0].Ipnets
	reservation.Capacity = reservations[0].Capacity
	return nil
}

func sendDeleteReservation6CmdToDHCPAgent(subnetID uint64, nodes []string, reservation *resource.Reservation6) error {
	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeleteReservation6,
		reservation6ToDeleteReservation6Request(subnetID, reservation), nil)
}

func reservation6ToDeleteReservation6Request(subnetID uint64, reservation *resource.Reservation6) *pbdhcpagent.DeleteReservation6Request {
	return &pbdhcpagent.DeleteReservation6Request{
		SubnetId:  subnetID,
		HwAddress: reservation.HwAddress,
		Duid:      reservation.Duid,
		Hostname:  reservation.Hostname,
	}
}

func (r *Reservation6Service) Update(subnetId string, reservation *resource.Reservation6) error {
	if err := resource.CheckCommentValid(reservation.Comment); err != nil {
		return err
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservation6, map[string]interface{}{
			resource.SqlColumnComment: reservation.Comment,
		}, map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return pg.Error(err)
		} else if rows == 0 {
			return fmt.Errorf("no found reservation6 %s", reservation.GetID())
		}

		return nil
	}); err != nil {
		return fmt.Errorf("update reservation6 %s with subnet6 %s failed: %s",
			reservation.String(), subnetId, err.Error())
	}

	return nil
}

func GetReservationPool6sByPrefix(prefix string) ([]*resource.Reservation6, error) {
	subnet6, err := GetSubnet6ByPrefix(prefix)
	if err != nil {
		return nil, err
	}

	if pools, err := listReservation6s(subnet6); err != nil {
		return nil, err
	} else {
		return pools, nil
	}
}

func BatchCreateReservation6s(prefix string, reservations []*resource.Reservation6) error {
	subnet, err := GetSubnet6ByPrefix(prefix)
	if err != nil {
		return err
	}

	for _, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return fmt.Errorf("validate reservation6 params invalid: %s", err.Error())
		}
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, reservation := range reservations {
			if err := checkReservation6CouldBeCreated(tx, subnet, reservation); err != nil {
				return err
			}

			if err := updateSubnet6AndPoolsCapacityWithReservation6(tx, subnet,
				reservation, true); err != nil {
				return err
			}

			reservation.Subnet6 = subnet.GetID()
			if _, err := tx.Insert(reservation); err != nil {
				return pg.Error(err)
			}

			if err := sendCreateReservation6CmdToDHCPAgent(
				subnet.SubnetId, subnet.Nodes, reservation); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("create reservation6s failed: %s", err.Error())
	}

	return nil
}
