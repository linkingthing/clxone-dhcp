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

type Reservation4Handler struct {
}

func NewReservation4Handler() *Reservation4Handler {
	return &Reservation4Handler{}
}

func (r *Reservation4Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	reservation := ctx.Resource.(*resource.Reservation4)
	if err := reservation.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create reservation params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservation4InUsed(tx, subnet.GetID(), reservation); err != nil {
			return err
		}

		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if subnet.Ipnet.Contains(net.ParseIP(reservation.IpAddress)) == false {
			return fmt.Errorf("reservation ipaddress %s not belongs to subnet %s",
				reservation.IpAddress, subnet.Subnet)
		}

		if err := checkReservation4ConflictWithReservedPool4(tx, subnet.GetID(), reservation); err != nil {
			return err
		}

		conflictPool, err := checkPool4ConflictWithSubnet4Pool4s(tx, subnet.GetID(),
			&resource.Pool4{BeginAddress: reservation.IpAddress, EndAddress: reservation.IpAddress})
		if err != nil {
			return err
		}

		if conflictPool == nil {
			if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
				"capacity": subnet.Capacity + 1,
			}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
				return fmt.Errorf("update subnet %s capacity to db failed: %s",
					subnet.GetID(), err.Error())
			}
		} else {
			if _, err := tx.Update(resource.TablePool4, map[string]interface{}{
				"capacity": conflictPool.Capacity - 1,
			}, map[string]interface{}{restdb.IDField: conflictPool.GetID()}); err != nil {
				return fmt.Errorf("update pool %s capacity to db failed: %s",
					conflictPool.String(), err.Error())
			}
		}

		reservation.Capacity = 1
		reservation.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(reservation); err != nil {
			return err
		}

		return sendCreateReservation4CmdToDHCPAgent(subnet.SubnetId, reservation)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create reservation %s failed: %s", reservation.String(), err.Error()))
	}

	return reservation, nil
}

func checkReservation4InUsed(tx restdb.Transaction, subnetId string, reservation *resource.Reservation4) error {
	count, err := tx.CountEx(resource.TableReservation4,
		"select count(*) from gr_reservation4 where subnet4 = $1 and (hw_address = $2 or ip_address = $3)",
		subnetId, reservation.HwAddress, reservation.IpAddress)

	if err != nil {
		return fmt.Errorf("check reservation %s with subnet %s exists in db failed: %s",
			reservation.String(), subnetId, err.Error())
	} else if count != 0 {
		return fmt.Errorf("reservation exists with subnet %s and mac %s or ip %s",
			subnetId, reservation.HwAddress, reservation.IpAddress)
	}

	return nil
}

func checkReservation4ConflictWithReservedPool4(tx restdb.Transaction, subnetId string, reservation *resource.Reservation4) error {
	var reservedpools []*resource.ReservedPool4
	if err := tx.Fill(map[string]interface{}{"subnet4": subnetId}, &reservedpools); err != nil {
		return fmt.Errorf("get subnet %s reserved pool4 from db failed: %s",
			subnetId, err.Error())
	}

	for _, reservedpool := range reservedpools {
		if reservedpool.Contains(reservation.IpAddress) {
			return fmt.Errorf("reservation %s conflict with reserved pool4 %s",
				reservation.String(), reservedpool.String())
		}
	}

	return nil
}

func sendCreateReservation4CmdToDHCPAgent(subnetID uint64, reservation *resource.Reservation4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateReservation4,
		&dhcpagent.CreateReservation4Request{
			SubnetId:  subnetID,
			HwAddress: reservation.HwAddress,
			IpAddress: reservation.IpAddress,
		})
}

func (r *Reservation4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var reservations resource.Reservation4s
	if err := db.GetResources(map[string]interface{}{"subnet4": subnetID}, &reservations); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list reservations with subnet %s from db failed: %s", subnetID, err.Error()))
	}

	leasesCount := getReservation4sLeasesCount(subnetIDStrToUint64(subnetID), reservations)
	for _, reservation := range reservations {
		setReservation4LeasesUsedRatio(reservation, leasesCount[reservation.IpAddress])
	}

	sort.Sort(reservations)
	return reservations, nil
}

func getReservation4sLeasesCount(subnetId uint64, reservations resource.Reservation4s) map[string]uint64 {
	resp, err := getSubnet4Leases(subnetId)
	if err != nil {
		log.Warnf("get subnet %s leases failed: %s", subnetId, err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := make(map[string]string)
	for _, reservation := range reservations {
		reservationMap[reservation.IpAddress] = reservation.HwAddress
	}

	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if mac, ok := reservationMap[lease.GetAddress()]; ok && mac == lease.GetHwAddress() {
			leasesCount[lease.GetAddress()] = 1
		}
	}

	return leasesCount
}

func (r *Reservation4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	reservationID := ctx.Resource.GetID()
	var reservations []*resource.Reservation4
	reservationInterface, err := restdb.GetResourceWithID(db.GetDB(), reservationID, &reservations)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get reservation %s with subnetID %s from db failed: %s",
				reservationID, subnetID, err.Error()))
	}

	reservation := reservationInterface.(*resource.Reservation4)
	if leasesCount, err := getReservation4LeasesCount(reservation); err != nil {
		log.Warnf("get reservation %s with subnet %s leases used ratio failed: %s",
			reservation.String(), subnetID, err.Error())
	} else {
		setReservation4LeasesUsedRatio(reservation, leasesCount)
	}

	return reservation, nil
}

func setReservation4LeasesUsedRatio(reservation *resource.Reservation4, leasesCount uint64) {
	if leasesCount != 0 {
		reservation.UsedCount = leasesCount
		reservation.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(reservation.Capacity))
	}
}

func getReservation4LeasesCount(reservation *resource.Reservation4) (uint64, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetReservation4LeasesCount(context.TODO(),
		&dhcpagent.GetReservation4LeasesCountRequest{
			SubnetId:  subnetIDStrToUint64(reservation.Subnet4),
			HwAddress: reservation.HwAddress,
		})

	return resp.GetLeasesCount(), err
}

func (r *Reservation4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	reservation := ctx.Resource.(*resource.Reservation4)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservation4FromDB(tx, reservation); err != nil {
			return err
		}

		if leasesCount, err := getReservation4LeasesCount(reservation); err != nil {
			return fmt.Errorf("get reservation %s leases count failed: %s",
				reservation.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete reservation with %d ips had been allocated", leasesCount)
		}

		conflictPool, err := checkPool4ConflictWithSubnet4Pool4s(tx, subnet.GetID(),
			&resource.Pool4{BeginAddress: reservation.IpAddress, EndAddress: reservation.IpAddress})
		if err != nil {
			return err
		} else if conflictPool != nil {
			if _, err := tx.Update(resource.TablePool4, map[string]interface{}{
				"capacity": conflictPool.Capacity + reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: conflictPool.GetID()}); err != nil {
				return fmt.Errorf("update pool %s capacity to db failed: %s",
					conflictPool.GetID(), err.Error())
			}
		} else {
			if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
				"capacity": subnet.Capacity - reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
				return fmt.Errorf("update subnet %s capacity to db failed: %s",
					subnet.GetID(), err.Error())
			}
		}

		if _, err := tx.Delete(resource.TableReservation4,
			map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservation4CmdToDHCPAgent(subnet.SubnetId, reservation)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete reservation %s with subnet %s failed: %s",
				reservation.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func setReservation4FromDB(tx restdb.Transaction, reservation *resource.Reservation4) error {
	var reservations []*resource.Reservation4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: reservation.GetID()}, &reservations); err != nil {
		return err
	}

	if len(reservations) == 0 {
		return fmt.Errorf("no found reservation %s", reservation.GetID())
	}

	reservation.Subnet4 = reservations[0].Subnet4
	reservation.HwAddress = reservations[0].HwAddress
	reservation.IpAddress = reservations[0].IpAddress
	reservation.Capacity = reservations[0].Capacity
	return nil
}

func sendDeleteReservation4CmdToDHCPAgent(subnetID uint64, reservation *resource.Reservation4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteReservation4,
		&dhcpagent.DeleteReservation4Request{
			SubnetId:  subnetID,
			HwAddress: reservation.HwAddress,
		})
}
