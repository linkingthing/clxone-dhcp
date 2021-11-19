package api

import (
	"context"
	"fmt"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

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

		if subnet.Ipnet.Contains(reservation.Ip) == false {
			return fmt.Errorf("reservation ipaddress %s not belongs to subnet %s",
				reservation.IpAddress, subnet.Subnet)
		}

		if err := checkReservation4ConflictWithReservedPool4(tx, subnet.GetID(),
			reservation); err != nil {
			return err
		}

		conflictPool, err := getConflictPool4InSubnet4(tx, subnet.GetID(),
			&resource.Pool4{BeginIp: reservation.Ip, EndIp: reservation.Ip})
		if err != nil {
			return err
		}

		if conflictPool == nil {
			if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
				"capacity": subnet.Capacity + reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
				return fmt.Errorf("update subnet %s capacity to db failed: %s",
					subnet.GetID(), err.Error())
			}
		} else {
			if _, err := tx.Update(resource.TablePool4, map[string]interface{}{
				"capacity": conflictPool.Capacity - reservation.Capacity,
			}, map[string]interface{}{restdb.IDField: conflictPool.GetID()}); err != nil {
				return fmt.Errorf("update pool %s capacity to db failed: %s",
					conflictPool.String(), err.Error())
			}
		}

		reservation.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(reservation); err != nil {
			return err
		}

		return sendCreateReservation4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservation)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create reservation %s failed: %s",
				reservation.String(), err.Error()))
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
	if err := tx.FillEx(&reservedpools,
		"select * from gr_reserved_pool4 where subnet4 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetId, reservation.Ip, reservation.Ip); err != nil {
		return fmt.Errorf("get pools with subnet %s from db failed: %s",
			subnetId, err.Error())
	}

	if len(reservedpools) != 0 {
		return fmt.Errorf("reservation %s conflict with reserved pool %s",
			reservation.String(), reservedpools[0].String())
	} else {
		return nil
	}
}

func sendCreateReservation4CmdToDHCPAgent(subnetID uint64, nodes []string, reservation *resource.Reservation4) error {
	nodesForSucceed, err := sendDHCPCmdWithNodes(nodes, dhcpservice.CreateReservation4,
		reservation4ToCreateReservation4Request(subnetID, reservation))
	if err != nil {
		if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, dhcpservice.DeleteReservation4,
			reservation4ToDeleteReservation4Request(subnetID, reservation)); err != nil {
			log.Errorf("create subnet %d reservation4 %s failed, and rollback it failed: %s",
				subnetID, reservation.String(), err.Error())
		}
	}

	return err
}

func reservation4ToCreateReservation4Request(subnetID uint64, reservation *resource.Reservation4) *dhcpagent.CreateReservation4Request {
	return &dhcpagent.CreateReservation4Request{
		SubnetId:  subnetID,
		HwAddress: reservation.HwAddress,
		IpAddress: reservation.IpAddress,
	}
}

func (r *Reservation4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var reservations []*resource.Reservation4
	if err := db.GetResources(map[string]interface{}{
		"subnet4": subnetID, "orderby": "ip"}, &reservations); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list reservations with subnet %s from db failed: %s",
				subnetID, err.Error()))
	}

	leasesCount := getReservation4sLeasesCount(subnetIDStrToUint64(subnetID), reservations)
	for _, reservation := range reservations {
		setReservation4LeasesUsedRatio(reservation, leasesCount[reservation.IpAddress])
	}

	return reservations, nil
}

func getReservation4sLeasesCount(subnetId uint64, reservations []*resource.Reservation4) map[string]uint64 {
	resp, err := getSubnet4Leases(subnetId)
	if err != nil {
		log.Warnf("get subnet %s leases failed: %s", subnetId, err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := reservationMapFromReservation4s(reservations)
	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if mac, ok := reservationMap[lease.GetAddress()]; ok &&
			mac == lease.GetHwAddress() {
			leasesCount[lease.GetAddress()] = 1
		}
	}

	return leasesCount
}

func reservationMapFromReservation4s(reservations []*resource.Reservation4) map[string]string {
	reservationMap := make(map[string]string)
	for _, reservation := range reservations {
		reservationMap[reservation.IpAddress] = reservation.HwAddress
	}

	return reservationMap
}

func (r *Reservation4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	reservationID := ctx.Resource.GetID()
	var reservations []*resource.Reservation4
	reservationInterface, err := restdb.GetResourceWithID(db.GetDB(),
		reservationID, &reservations)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get reservation %s with subnetID %s from db failed: %s",
				reservationID, subnetID, err.Error()))
	}

	reservation := reservationInterface.(*resource.Reservation4)
	if leasesCount, err := getReservation4LeaseCount(reservation); err != nil {
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
		reservation.UsedRatio = fmt.Sprintf("%.4f",
			float64(leasesCount)/float64(reservation.Capacity))
	}
}

func getReservation4LeaseCount(reservation *resource.Reservation4) (uint64, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetReservation4LeaseCount(
		context.TODO(), &dhcpagent.GetReservation4LeaseCountRequest{
			SubnetId:  subnetIDStrToUint64(reservation.Subnet4),
			HwAddress: reservation.HwAddress,
			IpAddress: reservation.IpAddress,
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

		if leasesCount, err := getReservation4LeaseCount(reservation); err != nil {
			return fmt.Errorf("get reservation %s leases count failed: %s",
				reservation.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete reservation with %d ips had been allocated",
				leasesCount)
		}

		conflictPool, err := getConflictPool4InSubnet4(tx, subnet.GetID(),
			&resource.Pool4{BeginIp: reservation.Ip, EndIp: reservation.Ip})
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

		return sendDeleteReservation4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservation)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete reservation %s with subnet %s failed: %s",
				reservation.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func setReservation4FromDB(tx restdb.Transaction, reservation *resource.Reservation4) error {
	var reservations []*resource.Reservation4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: reservation.GetID()},
		&reservations); err != nil {
		return err
	}

	if len(reservations) == 0 {
		return fmt.Errorf("no found reservation %s", reservation.GetID())
	}

	reservation.Subnet4 = reservations[0].Subnet4
	reservation.HwAddress = reservations[0].HwAddress
	reservation.IpAddress = reservations[0].IpAddress
	reservation.Ip = reservations[0].Ip
	reservation.Capacity = reservations[0].Capacity
	return nil
}

func sendDeleteReservation4CmdToDHCPAgent(subnetID uint64, nodes []string, reservation *resource.Reservation4) error {
	_, err := sendDHCPCmdWithNodes(nodes, dhcpservice.DeleteReservation4,
		reservation4ToDeleteReservation4Request(subnetID, reservation))
	return err
}

func reservation4ToDeleteReservation4Request(subnetID uint64, reservation *resource.Reservation4) *dhcpagent.DeleteReservation4Request {
	return &dhcpagent.DeleteReservation4Request{
		SubnetId:  subnetID,
		HwAddress: reservation.HwAddress,
		IpAddress: reservation.IpAddress,
	}
}
