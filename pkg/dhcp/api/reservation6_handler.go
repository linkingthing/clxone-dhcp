package api

import (
	"context"
	"fmt"
	"net"
	"sort"

	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	dhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"

	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
)

type Reservation6Handler struct {
}

func NewReservation6Handler() *Reservation6Handler {
	return &Reservation6Handler{}
}

func (r *Reservation6Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	reservation := ctx.Resource.(*resource.Reservation6)
	if err := reservation.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create reservation params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservation6InUsed(tx, subnet.GetID(), reservation); err != nil {
			return err
		}

		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkReservation6BelongsToIpnet(subnet.Ipnet, reservation); err != nil {
			return err
		}

		if err := checkReservation6ConflictWithReservedPools(tx, subnet.GetID(), reservation); err != nil {
			return err
		}

		conflictPdPool, conflictPool, err := checkReservation6ConflictWithSubnet6Pools(
			tx, subnet.GetID(), reservation)
		if err != nil {
			return err
		}

		reservation.Capacity = uint64(len(reservation.IpAddresses) + len(reservation.Prefixes))
		if conflictPool != nil {
			if _, err := tx.Update(resource.TablePool6, map[string]interface{}{
				"capacity": conflictPool.Capacity - reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: conflictPool.GetID()}); err != nil {
				return fmt.Errorf("update pool %s capacity to db failed: %s",
					conflictPool.String(), err.Error())
			}
		} else if conflictPdPool != nil {
			if _, err := tx.Update(resource.TablePdPool, map[string]interface{}{
				"capacity": conflictPdPool.Capacity - reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: conflictPdPool.GetID()}); err != nil {
				return fmt.Errorf("update pdpool %s capacity to db failed: %s",
					conflictPdPool.String(), err.Error())
			}
		} else {
			if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
				"capacity": subnet.Capacity + reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
				return fmt.Errorf("update subnet %s capacity to db failed: %s",
					subnet.GetID(), err.Error())
			}
		}

		reservation.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(reservation); err != nil {
			return err
		}

		return sendCreateReservation6CmdToDHCPAgent(subnet.SubnetId, reservation)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create reservation %s failed: %s", reservation.String(), err.Error()))
	}

	return reservation, nil
}

func checkReservation6InUsed(tx restdb.Transaction, subnetId string, reservation *resource.Reservation6) error {
	count, err := tx.CountEx(resource.TableReservation6,
		"select count(*) from gr_reservation6 where subnet6 = $1 and hw_address = $2 and duid = $3",
		subnetId, reservation.HwAddress, reservation.Duid)
	if err != nil {
		return fmt.Errorf("check reservation %s with subnet %s exists in db failed: %s",
			reservation.String(), subnetId, err.Error())
	} else if count != 0 {
		return fmt.Errorf("reservation exists with subnet %s and mac %s and duid %s",
			subnetId, reservation.HwAddress, reservation.Duid)
	}

	for _, ipAddress := range reservation.IpAddresses {
		count, err := tx.CountEx(resource.TableReservation6,
			"select count(*) from gr_reservation6 where subnet6 = $1 and $2=any(ip_addresses)",
			subnetId, ipAddress)
		if err != nil {
			return fmt.Errorf("check reservation %s with subnet %s exists in db failed: %s",
				reservation.String(), subnetId, err.Error())
		} else if count != 0 {
			return fmt.Errorf("reservation exists with subnet %s and ip %s",
				subnetId, ipAddress)
		}
	}

	for _, prefix := range reservation.Prefixes {
		count, err := tx.CountEx(resource.TableReservation6,
			"select count(*) from gr_reservation6 where subnet6 = $1 and $2=any(prefixes)",
			subnetId, prefix)
		if err != nil {
			return fmt.Errorf("check reservation %s with subnet %s exists in db failed: %s",
				reservation.String(), subnetId, err.Error())
		} else if count != 0 {
			return fmt.Errorf("reservation exists with subnet %s and prefix %s",
				subnetId, prefix)
		}
	}

	return nil
}

func checkReservation6BelongsToIpnet(ipnet net.IPNet, reservation *resource.Reservation6) error {
	for _, ipAddress := range reservation.IpAddresses {
		if checkIPsBelongsToIpnet(ipnet, ipAddress) == false {
			return fmt.Errorf("reservation %s ip %s not belong to subnet %s",
				reservation.String(), ipAddress, ipnet.String())
		}
	}

	subnetMaskLen, _ := ipnet.Mask.Size()
	for _, prefix := range reservation.Prefixes {
		if ip, ipnet_, _ := net.ParseCIDR(prefix); ipnet.Contains(ip) == false {
			return fmt.Errorf("reservation %s prefix %s not belong to subnet %s",
				reservation.String(), prefix, ipnet.String())
		} else {
			if ones, _ := ipnet_.Mask.Size(); ones <= subnetMaskLen {
				return fmt.Errorf("reservation %s prefix %s len %d less than subnet %d",
					reservation.String(), prefix, ones, subnetMaskLen)
			}
		}
	}

	return nil
}

func checkReservation6ConflictWithReservedPools(tx restdb.Transaction, subnetId string, reservation *resource.Reservation6) error {
	var reservedpools []*resource.ReservedPool6
	if len(reservation.IpAddresses) != 0 {
		if err := tx.Fill(map[string]interface{}{"subnet6": subnetId}, &reservedpools); err != nil {
			return fmt.Errorf("get subnet %s reserved pool6 from db failed: %s",
				subnetId, err.Error())
		}
	}

	var reservedpdpools []*resource.ReservedPdPool
	if len(reservation.Prefixes) != 0 {
		if err := tx.Fill(map[string]interface{}{"subnet6": subnetId}, &reservedpdpools); err != nil {
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

	conflictPdPool, err := checkPrefixesConflictWithSubnetPdPool(tx, subnetID, reservation6.Prefixes)
	return conflictPdPool, nil, err
}

func checkIpsConflictWithSubnet6Pool6s(tx restdb.Transaction, subnetID string, ips []string) (*resource.Pool6, error) {
	if len(ips) == 0 {
		return nil, nil
	}

	var pools []*resource.Pool6
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetID}, &pools); err != nil {
		return nil, fmt.Errorf("get pools with subnet %s from db failed: %s", subnetID, err.Error())
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
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetID}, &pdpools); err != nil {
		return nil, fmt.Errorf("get pdpools with subnet %s from db failed: %s", subnetID, err.Error())
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

func sendCreateReservation6CmdToDHCPAgent(subnetID uint64, reservation *resource.Reservation6) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateReservation6,
		&dhcpagent.CreateReservation6Request{
			SubnetId:    subnetID,
			HwAddress:   reservation.HwAddress,
			Duid:        reservation.Duid,
			IpAddresses: reservation.IpAddresses,
			Prefixes:    reservation.Prefixes,
		})
}

func (r *Reservation6Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var reservations resource.Reservation6s
	if err := db.GetResources(map[string]interface{}{"subnet6": subnetID}, &reservations); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list reservations with subnet %s from db failed: %s", subnetID, err.Error()))
	}

	leasesCount := getReservation6sLeasesCount(subnetIDStrToUint64(subnetID), reservations)
	for _, reservation := range reservations {
		setReservation6LeasesUsedRatio(reservation, leasesCount[reservation.GetID()])
	}

	sort.Sort(reservations)
	return reservations, nil
}

func getReservation6sLeasesCount(subnetId uint64, reservations resource.Reservation6s) map[string]uint64 {
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
				(len(reservation.HwAddress) != 0 && reservation.HwAddress == lease.GetHwAddress()) {
				count += 1
			}
			leasesCount[reservation.GetID()] = count
		}
	}

	return leasesCount
}

func (r *Reservation6Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	reservationID := ctx.Resource.GetID()
	var reservations []*resource.Reservation6
	reservationInterface, err := restdb.GetResourceWithID(db.GetDB(), reservationID, &reservations)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get reservation %s with subnetID %s from db failed: %s",
				reservationID, subnetID, err.Error()))
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
		reservation.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(reservation.Capacity))
	}
}

func getReservation6LeasesCount(reservation *resource.Reservation6) (uint64, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetReservation6LeasesCount(context.TODO(),
		&dhcpagent.GetReservation6LeasesCountRequest{
			SubnetId:  subnetIDStrToUint64(reservation.Subnet6),
			HwAddress: reservation.HwAddress,
			Duid:      reservation.Duid,
		})

	return resp.GetLeasesCount(), err
}

func (r *Reservation6Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	reservation := ctx.Resource.(*resource.Reservation6)
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
			return fmt.Errorf("can not delete reservation with %d ips had been allocated", leasesCount)
		}

		conflictPdPool, conflictPool, err := checkReservation6ConflictWithSubnet6Pools(
			tx, subnet.GetID(), reservation)
		if err != nil {
			return err
		} else if conflictPool != nil {
			if _, err := tx.Update(resource.TablePool6, map[string]interface{}{
				"capacity": conflictPool.Capacity + reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: conflictPool.GetID()}); err != nil {
				return fmt.Errorf("update pool %s capacity to db failed: %s",
					conflictPool.GetID(), err.Error())
			}
		} else if conflictPdPool != nil {
			if _, err := tx.Update(resource.TablePdPool, map[string]interface{}{
				"capacity": conflictPdPool.Capacity + reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: conflictPdPool.GetID()}); err != nil {
				return fmt.Errorf("update pdpool %s capacity to db failed: %s",
					conflictPool.GetID(), err.Error())
			}
		} else {
			if _, err := tx.Update(resource.TableSubnet6, map[string]interface{}{
				"capacity": subnet.Capacity - reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
				return fmt.Errorf("update subnet %s capacity to db failed: %s",
					subnet.GetID(), err.Error())
			}
		}

		if _, err := tx.Delete(resource.TableReservation6,
			map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservation6CmdToDHCPAgent(subnet.SubnetId, reservation)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete reservation %s with subnet %s failed: %s",
				reservation.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func setReservation6FromDB(tx restdb.Transaction, reservation *resource.Reservation6) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: reservation.GetID()}, &reservations); err != nil {
		return err
	}

	if len(reservations) == 0 {
		return fmt.Errorf("no found reservation %s", reservation.GetID())
	}

	reservation.Subnet6 = reservations[0].Subnet6
	reservation.HwAddress = reservations[0].HwAddress
	reservation.Duid = reservations[0].Duid
	reservation.IpAddresses = reservations[0].IpAddresses
	reservation.Prefixes = reservations[0].Prefixes
	reservation.Capacity = reservations[0].Capacity
	return nil
}

func sendDeleteReservation6CmdToDHCPAgent(subnetID uint64, reservation *resource.Reservation6) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteReservation6,
		&dhcpagent.DeleteReservation6Request{
			SubnetId:  subnetID,
			HwAddress: reservation.HwAddress,
			Duid:      reservation.Duid,
		})
}
