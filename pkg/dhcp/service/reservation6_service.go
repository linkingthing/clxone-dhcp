package service

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/linkingthing/cement/log"
	"github.com/linkingthing/clxone-utils/excel"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Reservation6Service struct {
}

func NewReservation6Service() *Reservation6Service {
	return &Reservation6Service{}
}

func (r *Reservation6Service) Create(subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := batchCreateReservation6s(tx, subnet, []*resource.Reservation6{reservation}); err != nil {
			return err
		}
		return sendCreateReservation6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, reservation)
	})
}

func checkReservation6CouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	} else if subnet.UseEui64 || subnet.AddressCode != "" {
		return errorno.ErrSubnetWithEui64OrCode(subnet.Subnet)
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
		return errorno.ErrLessThan(errorno.ErrNamePrefix, ipnet.String(), 64)
	}

	if subnetMaskLen > 64 && len(reservation.Ipnets) != 0 {
		return errorno.ErrBiggerThan(errorno.ErrNamePrefix, ipnet.String(), 64)
	}

	for _, ip := range reservation.Ips {
		if !ipnet.Contains(ip) {
			return errorno.ErrNotBelongTo(errorno.ErrNameIp, errorno.ErrNamePrefix,
				ip.String(), ipnet.String())
		}
	}

	for _, ipnet_ := range reservation.Ipnets {
		if !ipnet.Contains(ipnet_.IP) {
			return errorno.ErrNotBelongTo(errorno.ErrNameIp, errorno.ErrNamePrefix,
				ipnet_.String(), ipnet.String())
		} else if ones, _ := ipnet_.Mask.Size(); ones <= subnetMaskLen {
			return errorno.ErrLessThan(errorno.ErrNamePrefix, ipnet_.String(), subnetMaskLen)
		}
	}

	return nil
}

func checkReservation6InUsed(tx restdb.Transaction, subnetId string, reservation *resource.Reservation6) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&reservations); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	for _, reservation_ := range reservations {
		if reservation_.CheckConflictWithAnother(reservation) {
			return errorno.ErrConflict(errorno.ErrNameDhcpReservation, errorno.ErrNameDhcpReservation,
				reservation.String(), reservation_.String())
		}
	}

	return nil
}

func checkReservation6ConflictWithPools(tx restdb.Transaction, subnetId string, reservation *resource.Reservation6) error {
	var reservedpools []*resource.ReservedPool6
	if len(reservation.IpAddresses) != 0 {
		if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
			&reservedpools); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
		}
	}

	var reservedpdpools []*resource.ReservedPdPool
	if len(reservation.Prefixes) != 0 {
		if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
			&reservedpdpools); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameReservedPdPool), pg.Error(err).Error())
		}
	}

	for _, ip := range reservation.Ips {
		for _, reservedpool := range reservedpools {
			if reservedpool.ContainsIp(ip) {
				return errorno.ErrConflict(errorno.ErrNameDhcpReservation, errorno.ErrNameDhcpReservedPool,
					reservation.String(), reservedpool.String())
			}
		}
	}

	for _, ipnet := range reservation.Ipnets {
		for _, reservedpdpool := range reservedpdpools {
			if reservedpdpool.IntersectIpnet(ipnet) {
				return errorno.ErrConflict(errorno.ErrNameDhcpReservation, errorno.ErrNameReservedPdPool,
					reservation.String(), reservedpdpool.String())
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

	if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
		resource.SqlColumnCapacity: subnet.Capacity,
	}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameUpdate, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	}

	for affectedPoolId, capacity := range affectedPools {
		if _, err := tx.Update(resource.TablePool6, map[string]interface{}{
			resource.SqlColumnCapacity: capacity,
		}, map[string]interface{}{restdb.IDField: affectedPoolId}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
		}
	}

	for affectedPdPoolId, capacity := range affectedPdPools {
		if _, err := tx.Update(resource.TablePdPool, map[string]interface{}{
			resource.SqlColumnCapacity: capacity,
		}, map[string]interface{}{restdb.IDField: affectedPdPoolId}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNamePdPool), pg.Error(err).Error())
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
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

		if !reserved {
			unreservedCount.Add(unreservedCount, big.NewInt(1))
		}
	}

	if isCreate {
		subnet.AddCapacityWithBigInt(unreservedCount)
	} else {
		subnet.SubCapacityWithBigInt(unreservedCount)
	}

	return affectedPool6s, nil
}

func recalculatePdPoolsCapacityWithPrefixes(tx restdb.Transaction, subnet *resource.Subnet6, ipnets []net.IPNet, isCreate bool) (map[string]string, error) {
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet.GetID()},
		&pdpools); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
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

func sendCreateReservation6CmdToDHCPAgent(subnetID uint64, nodes []string, reservations ...*resource.Reservation6) error {
	if len(reservations) == 0 {
		return nil
	}
	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreateReservation6s,
		reservation6sToCreateReservations6Request(subnetID, reservations),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservation6s,
				reservation6sToDeleteReservations6Request(subnetID, reservations)); err != nil {
				log.Errorf("create subnet6 %d reservation6 %s failed, rollback with nodes %v failed: %s",
					subnetID, reservations[0].String(), nodesForSucceed, err.Error())
			}
		})
}

func reservation6sToCreateReservations6Request(subnetID uint64, reservations []*resource.Reservation6) *pbdhcpagent.CreateReservations6Request {
	pbPools := make([]*pbdhcpagent.CreateReservation6Request, len(reservations))
	for i, pool := range reservations {
		pbPools[i] = reservation6ToCreateReservation6Request(subnetID, pool)
	}
	return &pbdhcpagent.CreateReservations6Request{
		SubnetId:     subnetID,
		Reservations: pbPools,
	}
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

		if err := tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet6: subnet.GetID(),
			resource.SqlOrderBy:       "ips, ipnets"}, &reservations); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}
		return nil
	}); err != nil {
		return nil, err
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

func reservationMapFromReservation6s(reservations []*resource.Reservation6) map[string]*resource.Reservation6 {
	reservationMap := make(map[string]*resource.Reservation6)
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress+"/128"] = reservation
		}

		for _, prefix := range reservation.Prefixes {
			reservationMap[prefix] = reservation
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

		if err := tx.Fill(map[string]interface{}{restdb.IDField: reservationID}, &reservations); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, reservationID, pg.Error(err).Error())
		}
		return nil
	}); err != nil {
		return nil, err
	} else if len(reservations) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameDhcpReservation, reservationID)
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
		if err != nil {
			err = errorno.ErrNetworkError(errorno.ErrNameLease, err.Error())
		}
		return err
	}); err != nil {
		return 0, err
	}

	return resp.GetLeasesCount(), err
}

func (r *Reservation6Service) Delete(subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservation6CouldBeDeleted(tx, subnet, reservation); err != nil {
			return err
		}

		if err := updateSubnet6AndPoolsCapacityWithReservation6(tx, subnet,
			reservation, false); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableReservation6,
			map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, reservation.GetID(), pg.Error(err).Error())
		}

		return sendDeleteReservation6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, reservation)
	})
}

func checkReservation6CouldBeDeleted(tx restdb.Transaction, subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	if err := setSubnet6FromDB(tx, subnet); err != nil {
		return err
	}

	if err := setReservation6FromDB(tx, reservation); err != nil {
		return err
	}

	return checkReservation6WithLease(subnet, reservation)
}

func checkReservation6WithLease(subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	if leasesCount, err := getReservation6LeasesCount(subnet, reservation); err != nil {
		return err
	} else if leasesCount != 0 {
		return errorno.ErrIPHasBeenAllocated(errorno.ErrNameDhcpReservation, reservation.String())
	}

	return nil
}

func setReservation6FromDB(tx restdb.Transaction, reservation *resource.Reservation6) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: reservation.GetID()},
		&reservations); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, reservation.GetID(), pg.Error(err).Error())
	} else if len(reservations) == 0 {
		return errorno.ErrNotFound(errorno.ErrNameDhcpReservation, reservation.GetID())
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

func sendDeleteReservation6CmdToDHCPAgent(subnetID uint64, nodes []string, reservations ...*resource.Reservation6) error {
	return kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeleteReservation6s,
		reservation6sToDeleteReservations6Request(subnetID, reservations), nil)
}

func reservation6sToDeleteReservations6Request(subnetID uint64, reservations []*resource.Reservation6) *pbdhcpagent.DeleteReservations6Request {
	pbPools := make([]*pbdhcpagent.DeleteReservation6Request, len(reservations))
	for i, pool := range reservations {
		pbPools[i] = reservation6ToDeleteReservation6Request(subnetID, pool)
	}
	return &pbdhcpagent.DeleteReservations6Request{
		SubnetId:     subnetID,
		Reservations: pbPools,
	}
}

func reservation6ToDeleteReservation6Request(subnetID uint64, reservation *resource.Reservation6) *pbdhcpagent.DeleteReservation6Request {
	return &pbdhcpagent.DeleteReservation6Request{
		SubnetId:    subnetID,
		HwAddress:   reservation.HwAddress,
		Duid:        reservation.Duid,
		Hostname:    reservation.Hostname,
		IpAddresses: reservation.IpAddresses,
		Prefixes:    reservation.Prefixes,
	}
}

func (r *Reservation6Service) Update(subnetId string, reservation *resource.Reservation6) error {
	if err := util.ValidateStrings(util.RegexpTypeComma, reservation.Comment); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservation6, map[string]interface{}{
			resource.SqlColumnComment: reservation.Comment,
		}, map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, reservation.GetID(), pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameDhcpReservation, reservation.GetID())
		}

		return nil
	})
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

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err = batchCreateReservation6s(tx, subnet, reservations); err != nil {
			return err
		}
		return sendCreateReservation6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, reservations...)
	})
}

func batchCreateReservation6s(tx restdb.Transaction, subnet *resource.Subnet6, reservations []*resource.Reservation6) error {
	for _, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return err
		}
		if err := checkReservation6CouldBeCreated(tx, subnet, reservation); err != nil {
			return err
		}

		if err := updateSubnet6AndPoolsCapacityWithReservation6(tx, subnet,
			reservation, true); err != nil {
			return err
		}

		reservation.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(reservation); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}
	}

	return nil
}

func (s *Reservation6Service) BatchDeleteReservation6s(subnetId string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	var reservations []*resource.Reservation6
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet, err := getSubnet6FromDB(tx, subnetId)
		if err != nil {
			return err
		}

		if err = tx.Fill(map[string]interface{}{restdb.IDField: restdb.FillValue{
			Operator: restdb.OperatorAny, Value: ids}},
			&reservations); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}

		for _, reservation := range reservations {
			if err = setReservation6FromDB(tx, reservation); err != nil {
				return err
			}

			if err = checkReservation6WithLease(subnet, reservation); err != nil {
				return err
			}

			if err = updateSubnet6AndPoolsCapacityWithReservation6(tx, subnet,
				reservation, false); err != nil {
				return err
			}

			if _, err = tx.Delete(resource.TableReservation6,
				map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
				return errorno.ErrDBError(errorno.ErrDBNameDelete, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
			}
		}

		return sendDeleteReservation6CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, reservations...)
	})
}

func (s *Reservation6Service) ImportExcel(file *excel.ImportFile, subnetId string) (interface{}, error) {
	var subnet6s []*resource.Subnet6
	if err := db.GetResources(map[string]interface{}{restdb.IDField: subnetId},
		&subnet6s); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	} else if len(subnet6s) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetwork, subnetId)
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(Reservation6ImportFileNamePrefix, TableHeaderReservation6Fail, response)
	reservations, err := s.parseReservation6sFromFile(file.Name, subnet6s[0], response)
	if err != nil {
		return response, err
	}

	if len(reservations) == 0 {
		return response, nil
	}

	validReservations := make([]*resource.Reservation6, 0, len(reservations))
	if err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, reservation := range reservations {
			if err = checkReservation6CouldBeCreated(tx, subnet6s[0], reservation); err != nil {
				addFailDataToResponse(response, TableHeaderReservation6FailLen,
					localizationReservation6ToStrSlice(reservation), errorno.TryGetErrorCNMsg(err))
				continue
			}

			if err = batchInsertReservation6s(tx, subnet6s[0], reservation); err != nil {
				return err
			}

			validReservations = append(validReservations, reservation)
		}

		return sendCreateReservation6CmdToDHCPAgent(subnet6s[0].SubnetId, subnet6s[0].Nodes, validReservations...)
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func batchInsertReservation6s(tx restdb.Transaction, subnet *resource.Subnet6, reservation *resource.Reservation6) error {
	if err := updateSubnet6AndPoolsCapacityWithReservation6(tx, subnet,
		reservation, true); err != nil {
		return err
	}

	reservation.Subnet6 = subnet.GetID()
	if _, err := tx.Insert(reservation); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	return nil
}

func (s *Reservation6Service) parseReservation6sFromFile(fileName string, subnet6 *resource.Subnet6,
	response *excel.ImportResult) ([]*resource.Reservation6, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0],
		TableHeaderReservation6, Reservation6MandatoryFields)
	if err != nil {
		return nil, errorno.ErrInvalidTableHeader()
	}

	response.InitData(len(contents) - 1)
	fieldcontents := contents[1:]
	reservations := make([]*resource.Reservation6, 0, len(fieldcontents))
	reservationMap := make(map[string]struct{}, len(fieldcontents))
	var contains bool
	var invalidIp string
	for j, fields := range fieldcontents {
		contains = true
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, Reservation6MandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderReservation6FailLen,
				localizationReservation6ToStrSlice(&resource.Reservation6{}),
				errorno.ErrMissingMandatory(j+2, Reservation6MandatoryFields).ErrorCN())
			continue
		}

		reservation6, err := s.parseReservation6FromFields(fields, tableHeaderFields)
		if err != nil {
			addFailDataToResponse(response, TableHeaderReservation6FailLen,
				localizationReservation6ToStrSlice(reservation6), errorno.TryGetErrorCNMsg(err))
			continue
		}

		if err = reservation6.Validate(); err != nil {
			addFailDataToResponse(response, TableHeaderReservation6FailLen,
				localizationReservation6ToStrSlice(reservation6), errorno.TryGetErrorCNMsg(err))
			continue
		}

		for _, ip := range reservation6.Ips {
			if !subnet6.Ipnet.Contains(ip) {
				invalidIp = ip.String()
				contains = false
				break
			}
		}

		if !contains {
			addFailDataToResponse(response, TableHeaderReservation6FailLen,
				localizationReservation6ToStrSlice(reservation6),
				errorno.ErrNotBelongTo(errorno.ErrNameIp, errorno.ErrNameNetwork,
					invalidIp, subnet6.Ipnet.String()).ErrorCN())
			continue
		}

		hasBreak := false
		for _, IpAddress := range reservation6.IpAddresses {
			if _, ok := reservationMap[IpAddress]; ok {
				addFailDataToResponse(response, TableHeaderReservation6FailLen,
					localizationReservation6ToStrSlice(reservation6), errorno.ErrDuplicate(errorno.ErrNameIp, IpAddress).ErrorCN())
				hasBreak = true
				break
			}
			reservationMap[IpAddress] = struct{}{}
		}

		if hasBreak {
			continue
		}
		reservations = append(reservations, reservation6)
	}

	return reservations, nil
}

func (s *Reservation6Service) parseReservation6FromFields(fields, tableHeaderFields []string) (*resource.Reservation6, error) {
	reservation6 := &resource.Reservation6{}

	var deviceFlag string
	var err error
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}
		field = strings.TrimSpace(field)
		switch tableHeaderFields[i] {
		case FieldNameIpV6Address:
			addresses := strings.Split(strings.TrimSpace(field), ",")
			reservation6.IpAddresses = addresses
		case FieldNameReservation6DeviceFlag:
			deviceFlag = field
		case FieldNameReservation6DeviceFlagValue:
			if deviceFlag == ReservationFlagMac {
				reservation6.HwAddress = field
				continue
			} else if deviceFlag == ReservationFlagHostName {
				reservation6.Hostname = field
				continue
			} else if deviceFlag == ReservationFlagDUID {
				reservation6.Duid = field
				continue
			}
			err = errorno.ErrInvalidParams(errorno.ErrNameDeviceFlag, field)
		case FieldNameComment:
			reservation6.Comment = field
		}
	}
	return reservation6, err
}

func (s *Reservation6Service) ExportExcel(subnetId string) (*excel.ExportFile, error) {
	var reservation6s []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId}, &reservation6s)
		return err
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	strMatrix := make([][]string, 0, len(reservation6s))
	for _, reservation6 := range reservation6s {
		strMatrix = append(strMatrix, localizationReservation6ToStrSlice(reservation6))
	}

	if filepath, err := excel.WriteExcelFile(Reservation6FileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderReservation6, strMatrix,
		getOpt(Reservation6DropList, len(strMatrix)+1)); err != nil {
		return nil, errorno.ErrExport(errorno.ErrNameDhcpReservation, err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (s *Reservation6Service) ExportExcelTemplate() (*excel.ExportFile, error) {
	if filepath, err := excel.WriteExcelFile(Reservation6TemplateFileName,
		TableHeaderReservation6, TemplateReservation6, getOpt(Reservation6DropList, len(TemplateReservation6)+1)); err != nil {
		return nil, errorno.ErrExportTmp(errorno.ErrNameDhcpReservation, err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}
