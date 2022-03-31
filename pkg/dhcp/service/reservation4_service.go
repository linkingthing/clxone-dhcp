package service

import (
	"context"
	"fmt"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type Reservation4Service struct {
}

func NewReservation4Service() *Reservation4Service {
	return &Reservation4Service{}
}

func (r *Reservation4Service) Create(subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	if err := reservation.Validate(); err != nil {
		return fmt.Errorf("validate reservation4 params invalid: %s", err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservation4CouldBeCreated(tx, subnet, reservation); err != nil {
			return err
		}

		if err := updateSubnet4OrPool4CapacityWithReservation4(tx, subnet,
			reservation, true); err != nil {
			return err
		}

		reservation.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(reservation); err != nil {
			return err
		}

		return sendCreateReservation4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservation)
	}); err != nil {
		return fmt.Errorf("create reservation4 %s failed: %s", reservation.String(), err.Error())
	}

	return nil
}

func checkReservation4CouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	if err := setSubnet4FromDB(tx, subnet); err != nil {
		return err
	}

	if subnet.Ipnet.Contains(reservation.Ip) == false {
		return fmt.Errorf("reservation4 ipaddress %s not belongs to subnet4 %s",
			reservation.IpAddress, subnet.Subnet)
	}

	if err := checkReservation4InUsed(tx, subnet.GetID(), reservation); err != nil {
		return err
	}

	return checkReservation4ConflictWithReservedPool4(tx, subnet.GetID(), reservation)
}

func checkReservation4InUsed(tx restdb.Transaction, subnetId string, reservation *resource.Reservation4) error {
	if count, err := tx.CountEx(resource.TableReservation4,
		"select count(*) from gr_reservation4 where subnet4 = $1 and (hw_address = $2 or ip_address = $3)",
		subnetId, reservation.HwAddress, reservation.IpAddress); err != nil {
		return fmt.Errorf("check reservation4 %s with subnet4 %s exists in db failed: %s",
			reservation.String(), subnetId, err.Error())
	} else if count != 0 {
		return fmt.Errorf("reservation4 exists with subnet4 %s and mac %s or ip %s",
			subnetId, reservation.HwAddress, reservation.IpAddress)
	} else {
		return nil
	}
}

func checkReservation4ConflictWithReservedPool4(tx restdb.Transaction, subnetId string, reservation *resource.Reservation4) error {
	if reservedpools, err := getReservedPool4sWithBeginAndEndIp(tx, subnetId,
		reservation.Ip, reservation.Ip); err != nil {
		return err
	} else if len(reservedpools) != 0 {
		return fmt.Errorf("reservation4 %s conflict with reserved pool4 %s",
			reservation.String(), reservedpools[0].String())
	} else {
		return nil
	}
}

func updateSubnet4OrPool4CapacityWithReservation4(tx restdb.Transaction, subnet *resource.Subnet4, reservation *resource.Reservation4, isCreate bool) error {
	conflictPools, err := getPool4sWithBeginAndEndIp(tx, subnet.GetID(),
		reservation.Ip, reservation.Ip)
	if err != nil {
		return err
	}

	if len(conflictPools) == 0 {
		if isCreate {
			subnet.Capacity += reservation.Capacity
		} else {
			subnet.Capacity -= reservation.Capacity
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			"capacity": subnet.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet4 %s capacity to db failed: %s",
				subnet.GetID(), err.Error())
		}
	} else {
		if isCreate {
			conflictPools[0].Capacity -= reservation.Capacity
		} else {
			conflictPools[0].Capacity += reservation.Capacity
		}

		if _, err := tx.Update(resource.TablePool4, map[string]interface{}{
			"capacity": conflictPools[0].Capacity,
		}, map[string]interface{}{restdb.IDField: conflictPools[0].GetID()}); err != nil {
			return fmt.Errorf("update pool4 %s capacity to db failed: %s",
				conflictPools[0].String(), err.Error())
		}
	}

	return nil
}

func sendCreateReservation4CmdToDHCPAgent(subnetID uint64, nodes []string, reservation *resource.Reservation4) error {
	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.CreateReservation4,
		reservation4ToCreateReservation4Request(subnetID, reservation),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservation4,
				reservation4ToDeleteReservation4Request(subnetID, reservation)); err != nil {
				log.Errorf("create subnet4 %d reservation4 %s failed, rollback with nodes %v failed: %s",
					subnetID, reservation.String(), nodesForSucceed, err.Error())
			}
		})
}

func reservation4ToCreateReservation4Request(subnetID uint64, reservation *resource.Reservation4) *pbdhcpagent.CreateReservation4Request {
	return &pbdhcpagent.CreateReservation4Request{
		SubnetId:  subnetID,
		HwAddress: reservation.HwAddress,
		IpAddress: reservation.IpAddress,
	}
}

func (r *Reservation4Service) List(subnetID string) ([]*resource.Reservation4, error) {
	return listReservation4s(subnetID)
}

func listReservation4s(subnetID string) ([]*resource.Reservation4, error) {
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet4: subnetID,
			resource.SqlOrderBy:       resource.SqlColumnsIp}, &reservations)
	}); err != nil {
		return nil, fmt.Errorf("list reservation4s with subnet4 %s from db failed: %s",
			subnetID, err.Error())
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
		log.Warnf("get subnet4 %s leases failed: %s", subnetId, err.Error())
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

func (r *Reservation4Service) Get(subnet *resource.Subnet4, reservationID string) (*resource.Reservation4, error) {
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: reservationID}, &reservations)
	}); err != nil {
		return nil, fmt.Errorf("get reservation4 %s with subnetID %s failed: %s",
			reservationID, subnet.GetID(), err.Error())
	} else if len(reservations) == 0 {
		return nil, fmt.Errorf("no found reservation4 %s with subnetID %s", reservationID, subnet.GetID())
	}

	if leasesCount, err := getReservation4LeaseCount(reservations[0]); err != nil {
		log.Warnf("get reservation4 %s with subnet4 %s leases used ratio failed: %s",
			reservations[0].String(), subnet.GetID(), err.Error())
	} else {
		setReservation4LeasesUsedRatio(reservations[0], leasesCount)
	}

	return reservations[0], nil
}

func setReservation4LeasesUsedRatio(reservation *resource.Reservation4, leasesCount uint64) {
	if leasesCount != 0 {
		reservation.UsedCount = leasesCount
		reservation.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(reservation.Capacity))
	}
}

func getReservation4LeaseCount(reservation *resource.Reservation4) (uint64, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetReservation4LeaseCount(
		context.TODO(), &pbdhcpagent.GetReservation4LeaseCountRequest{
			SubnetId:  subnetIDStrToUint64(reservation.Subnet4),
			HwAddress: reservation.HwAddress,
			IpAddress: reservation.IpAddress,
		})

	return resp.GetLeasesCount(), err
}

func (r *Reservation4Service) Delete(subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservation4CouldBeDeleted(tx, subnet, reservation); err != nil {
			return err
		}

		if err := updateSubnet4OrPool4CapacityWithReservation4(tx, subnet,
			reservation, false); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableReservation4,
			map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservation4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservation)
	}); err != nil {
		return fmt.Errorf("delete reservation4 %s with subnet4 %s failed: %s",
			reservation.String(), subnet.GetID(), err.Error())
	}

	return nil
}

func checkReservation4CouldBeDeleted(tx restdb.Transaction, subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	if err := setSubnet4FromDB(tx, subnet); err != nil {
		return err
	}

	if err := setReservation4FromDB(tx, reservation); err != nil {
		return err
	}

	if leasesCount, err := getReservation4LeaseCount(reservation); err != nil {
		return fmt.Errorf("get reservation4 %s leases count failed: %s",
			reservation.String(), err.Error())
	} else if leasesCount != 0 {
		return fmt.Errorf("can not delete reservation4 with %d ips had been allocated",
			leasesCount)
	}

	return nil
}

func setReservation4FromDB(tx restdb.Transaction, reservation *resource.Reservation4) error {
	var reservations []*resource.Reservation4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: reservation.GetID()},
		&reservations); err != nil {
		return err
	} else if len(reservations) == 0 {
		return fmt.Errorf("no found reservation4 %s", reservation.GetID())
	}

	reservation.Subnet4 = reservations[0].Subnet4
	reservation.HwAddress = reservations[0].HwAddress
	reservation.IpAddress = reservations[0].IpAddress
	reservation.Ip = reservations[0].Ip
	reservation.Capacity = reservations[0].Capacity
	return nil
}

func sendDeleteReservation4CmdToDHCPAgent(subnetID uint64, nodes []string, reservation *resource.Reservation4) error {
	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.DeleteReservation4,
		reservation4ToDeleteReservation4Request(subnetID, reservation), nil)
}

func reservation4ToDeleteReservation4Request(subnetID uint64, reservation *resource.Reservation4) *pbdhcpagent.DeleteReservation4Request {
	return &pbdhcpagent.DeleteReservation4Request{
		SubnetId:  subnetID,
		HwAddress: reservation.HwAddress,
		IpAddress: reservation.IpAddress,
	}
}

func (r *Reservation4Service) Update(subnetId string, reservation *resource.Reservation4) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservation4, map[string]interface{}{
			resource.SqlColumnComment: reservation.Comment,
		}, map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found reservation4 %s", reservation.GetID())
		}

		return nil
	}); err != nil {
		return fmt.Errorf("update reservation4 %s with subnet4 %s failed: %s",
			reservation.String(), subnetId, err.Error())
	}

	return nil
}

func GetReservationPool4sByPrefix(prefix string) ([]*resource.Reservation4, error) {
	subnet4, err := GetSubnet4ByPrefix(prefix)
	if err != nil {
		return nil, err
	}

	if pools, err := listReservation4s(subnet4.GetID()); err != nil {
		return nil, err
	} else {
		return pools, nil
	}
}

func BatchCreateReservation4s(prefix string, reservations []*resource.Reservation4) error {
	subnet, err := GetSubnet4ByPrefix(prefix)
	if err != nil {
		return err
	}

	for _, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return fmt.Errorf("validate reservation4 params invalid: %s", err.Error())
		}
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, reservation := range reservations {
			if err := checkReservation4CouldBeCreated(tx, subnet, reservation); err != nil {
				return err
			}

			if err := updateSubnet4OrPool4CapacityWithReservation4(tx, subnet,
				reservation, true); err != nil {
				return err
			}

			reservation.Subnet4 = subnet.GetID()
			if _, err := tx.Insert(reservation); err != nil {
				return err
			}

			if err := sendCreateReservation4CmdToDHCPAgent(
				subnet.SubnetId, subnet.Nodes, reservation); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("create reservation4s failed: %s", err.Error())
	}

	return nil
}
