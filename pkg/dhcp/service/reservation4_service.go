package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/linkingthing/cement/log"
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

type Reservation4Service struct {
}

func NewReservation4Service() *Reservation4Service {
	return &Reservation4Service{}
}

func (r *Reservation4Service) Create(subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	if err := reservation.Validate(); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservation4CouldBeCreated(tx, subnet, reservation); err != nil {
			return err
		}

		if err := updateSubnet4OrPool4CapacityWithReservation4(tx, subnet,
			reservation, true); err != nil {
			return err
		}

		reservation.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(reservation); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}

		return sendCreateReservation4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservation)
	})
}

func checkReservation4CouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	if err := setSubnet4FromDB(tx, subnet); err != nil {
		return err
	}

	if !subnet.Ipnet.Contains(reservation.Ip) {
		return errorno.ErrNotBelongTo(errorno.ErrNameDhcpReservation, errorno.ErrNameNetworkV4,
			reservation.IpAddress, subnet.Subnet)
	}

	if err := checkReservation4InUsed(tx, subnet.GetID(), reservation); err != nil {
		return err
	}

	return checkReservation4ConflictWithReservedPool4(tx, subnet.GetID(), reservation)
}

func checkReservation4InUsed(tx restdb.Transaction, subnetId string, reservation *resource.Reservation4) error {
	if count, err := tx.CountEx(resource.TableReservation4,
		"select count(*) from gr_reservation4 where subnet4 = $1 and (hw_address = $2 and hostname = $3 or ip_address = $4)",
		subnetId, reservation.HwAddress, reservation.Hostname, reservation.IpAddress); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameCount, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	} else if count != 0 {
		return errorno.ErrUsedReservation()
	} else {
		return nil
	}
}

func checkReservation4ConflictWithReservedPool4(tx restdb.Transaction, subnetId string, reservation *resource.Reservation4) error {
	if reservedpools, err := getReservedPool4sWithBeginAndEndIp(tx, subnetId,
		reservation.Ip, reservation.Ip); err != nil {
		return err
	} else if len(reservedpools) != 0 {
		return errorno.ErrConflict(errorno.ErrNameDhcpReservation, errorno.ErrNameDhcpReservedPool,
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
			resource.SqlColumnCapacity: subnet.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
		}
	} else {
		if isCreate {
			conflictPools[0].Capacity -= reservation.Capacity
		} else {
			conflictPools[0].Capacity += reservation.Capacity
		}

		if _, err := tx.Update(resource.TablePool4, map[string]interface{}{
			resource.SqlColumnCapacity: conflictPools[0].Capacity,
		}, map[string]interface{}{restdb.IDField: conflictPools[0].GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, conflictPools[0].String(), pg.Error(err).Error())
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
		Hostname:  reservation.Hostname,
		IpAddress: reservation.IpAddress,
	}
}

func (r *Reservation4Service) List(subnet *resource.Subnet4) ([]*resource.Reservation4, error) {
	return listReservation4s(subnet)
}

func listReservation4s(subnet *resource.Subnet4) ([]*resource.Reservation4, error) {
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet4: subnet.GetID(),
			resource.SqlOrderBy:       resource.SqlColumnsIp}, &reservations); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if len(subnet.Nodes) != 0 {
		leasesCount := getReservation4sLeasesCount(subnetIDStrToUint64(subnet.GetID()), reservations)
		for _, reservation := range reservations {
			setReservation4LeasesUsedRatio(reservation, leasesCount[reservation.IpAddress])
		}
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
		if reservation, ok := reservationMap[lease.GetAddress()]; ok &&
			(reservation.HwAddress == "" || reservation.HwAddress == lease.GetHwAddress()) &&
			(reservation.Hostname == "" || reservation.Hostname == lease.GetHostname()) {
			leasesCount[lease.GetAddress()] = 1
		}
	}

	return leasesCount
}

func reservationMapFromReservation4s(reservations []*resource.Reservation4) map[string]*resource.Reservation4 {
	reservationMap := make(map[string]*resource.Reservation4)
	for _, reservation := range reservations {
		reservationMap[reservation.IpAddress] = reservation
	}

	return reservationMap
}

func (r *Reservation4Service) Get(subnet *resource.Subnet4, reservationID string) (*resource.Reservation4, error) {
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
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

	if leasesCount, err := getReservation4LeaseCount(subnet, reservations[0]); err != nil {
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

func getReservation4LeaseCount(subnet *resource.Subnet4, reservation *resource.Reservation4) (uint64, error) {
	if len(subnet.Nodes) == 0 {
		return 0, nil
	}

	var resp *pbdhcpagent.GetLeasesCountResponse
	var err error
	if err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetReservation4LeaseCount(
			ctx, &pbdhcpagent.GetReservation4LeaseCountRequest{
				SubnetId:  subnetIDStrToUint64(reservation.Subnet4),
				HwAddress: strings.ToLower(reservation.HwAddress),
				Hostname:  reservation.Hostname,
				IpAddress: reservation.IpAddress,
			})
		return err
	}); err != nil {
		return 0, errorno.ErrNetworkError(errorno.ErrNameLease, err.Error())
	}

	return resp.GetLeasesCount(), err
}

func (r *Reservation4Service) Delete(subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservation4CouldBeDeleted(tx, subnet, reservation); err != nil {
			return err
		}

		if err := updateSubnet4OrPool4CapacityWithReservation4(tx, subnet,
			reservation, false); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableReservation4,
			map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, reservation.GetID(), pg.Error(err).Error())
		}

		return sendDeleteReservation4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservation)
	})
}

func checkReservation4CouldBeDeleted(tx restdb.Transaction, subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	if err := setSubnet4FromDB(tx, subnet); err != nil {
		return err
	}

	if err := setReservation4FromDB(tx, reservation); err != nil {
		return err
	}

	if leasesCount, err := getReservation4LeaseCount(subnet, reservation); err != nil {
		return err
	} else if leasesCount != 0 {
		return errorno.ErrIPHasBeenAllocated(errorno.ErrNameDhcpReservation, reservation.GetID())
	}

	return nil
}

func setReservation4FromDB(tx restdb.Transaction, reservation *resource.Reservation4) error {
	var reservations []*resource.Reservation4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: reservation.GetID()},
		&reservations); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, reservation.GetID(), pg.Error(err).Error())
	} else if len(reservations) == 0 {
		return errorno.ErrNotFound(errorno.ErrNameDhcpReservation, reservation.GetID())
	}

	reservation.Subnet4 = reservations[0].Subnet4
	reservation.HwAddress = reservations[0].HwAddress
	reservation.Hostname = reservations[0].Hostname
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
		Hostname:  reservation.Hostname,
		IpAddress: reservation.IpAddress,
	}
}

func (r *Reservation4Service) Update(subnetId string, reservation *resource.Reservation4) error {
	if err := util.ValidateStrings(util.RegexpTypeComma, reservation.Comment); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservation4, map[string]interface{}{
			resource.SqlColumnComment: reservation.Comment,
		}, map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, reservation.GetID(), pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameDhcpReservation, reservation.GetID())
		}

		return nil
	})
}

func GetReservationPool4sByPrefix(prefix string) ([]*resource.Reservation4, error) {
	subnet4, err := GetSubnet4ByPrefix(prefix)
	if err != nil {
		return nil, err
	}

	if pools, err := listReservation4s(subnet4); err != nil {
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
			return err
		}
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
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
				return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
			}

			if err := sendCreateReservation4CmdToDHCPAgent(
				subnet.SubnetId, subnet.Nodes, reservation); err != nil {
				return err
			}
		}
		return nil
	})
}
