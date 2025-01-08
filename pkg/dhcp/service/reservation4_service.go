package service

import (
	"context"
	"fmt"
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

type CreateReservationMode uint32

const (
	CreateReservationModeCreate  CreateReservationMode = 1
	CreateReservationModeImport  CreateReservationMode = 2
	CreateReservationModeConvert CreateReservationMode = 3
)

type ListResourceMode uint32

const (
	ListResourceModeAPI  ListResourceMode = 1
	ListResourceModeGRPC ListResourceMode = 2
)

type Reservation4Service struct{}

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
			return errorno.ErrDBError(errorno.ErrDBNameInsert,
				string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}

		return sendCreateReservation4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, reservation)
	})
}

func checkReservation4CouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	if err := setSubnet4FromDB(tx, subnet); err != nil {
		return err
	}

	if !subnet.Ipnet.Contains(reservation.Ip) {
		return errorno.ErrNotBelongTo(errorno.ErrNameDhcpReservation,
			errorno.ErrNameNetworkV4, reservation.IpAddress, subnet.Subnet)
	}

	if err := checkReservation4InUsed(tx, subnet.GetID(), reservation); err != nil {
		return err
	}

	return checkReservation4ConflictWithReservedPool4(tx, subnet.GetID(), reservation)
}

func checkReservation4InUsed(tx restdb.Transaction, subnetId string, reservation *resource.Reservation4) error {
	if count, err := tx.CountEx(resource.TableReservation4,
		`select count(*) from gr_reservation4 where subnet4 = $1 and 
			(hw_address = $2 and hostname = $3 or ip_address = $4)`, subnetId,
		reservation.HwAddress, reservation.Hostname, reservation.IpAddress); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	} else if count != 0 {
		return errorno.ErrUsedReservation(reservation.IpAddress)
	} else {
		return nil
	}
}

func checkReservation4ConflictWithReservedPool4(tx restdb.Transaction, subnetId string, reservation *resource.Reservation4) error {
	if reservedpools, err := getReservedPool4sWithBeginAndEndIp(tx, subnetId,
		reservation.Ip, reservation.Ip); err != nil {
		return err
	} else if len(reservedpools) != 0 {
		return errorno.ErrConflict(errorno.ErrNameDhcpReservation,
			errorno.ErrNameDhcpReservedPool, reservation.String(), reservedpools[0].String())
	} else {
		return nil
	}
}

func updateSubnet4OrPool4CapacityWithReservation4(tx restdb.Transaction, subnet *resource.Subnet4, reservation *resource.Reservation4, isCreate bool) error {
	affectPools, err := getPool4sWithBeginAndEndIp(tx, subnet.GetID(),
		reservation.Ip, reservation.Ip)
	if err != nil {
		return err
	}

	if len(affectPools) == 0 {
		if isCreate {
			subnet.Capacity += reservation.Capacity
		} else {
			subnet.Capacity -= reservation.Capacity
		}

		if err := updateResourceCapacity(tx, resource.TableSubnet4, subnet.GetID(),
			subnet.Capacity, errorno.ErrNameNetworkV4); err != nil {
			return err
		}
	} else {
		if isCreate {
			affectPools[0].Capacity -= reservation.Capacity
		} else {
			affectPools[0].Capacity += reservation.Capacity
		}

		if err := updateResourceCapacity(tx, resource.TablePool4, affectPools[0].GetID(),
			affectPools[0].Capacity, errorno.ErrName(affectPools[0].String())); err != nil {
			return err
		}
	}

	return nil
}

func updateResourceCapacity(tx restdb.Transaction, resourceType restdb.ResourceType, resourceId string, capacity interface{}, errName errorno.ErrName) error {
	if _, err := tx.Update(resourceType,
		map[string]interface{}{resource.SqlColumnCapacity: capacity},
		map[string]interface{}{restdb.IDField: resourceId}); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameUpdate, string(errName),
			pg.Error(err).Error())
	}

	return nil
}

func sendCreateReservation4CmdToDHCPAgent(subnetID uint64, nodes []string, reservation *resource.Reservation4) error {
	if len(nodes) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.CreateReservation4,
		reservation4ToCreateReservation4Request(subnetID, reservation),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservation4,
				reservation4ToDeleteReservation4Request(subnetID, reservation)); err != nil {
				log.Errorf("create subnet4 %d reservation4 %s failed, rollback %v failed: %s",
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

func reservation4ToDeleteReservation4Request(subnetID uint64, reservation *resource.Reservation4) *pbdhcpagent.DeleteReservation4Request {
	return &pbdhcpagent.DeleteReservation4Request{
		SubnetId:  subnetID,
		HwAddress: reservation.HwAddress,
		Hostname:  reservation.Hostname,
		IpAddress: reservation.IpAddress,
	}
}

func (r *Reservation4Service) List(subnet *resource.Subnet4) ([]*resource.Reservation4, error) {
	return listReservation4s(subnet, ListResourceModeAPI)
}

func listReservation4s(subnet *resource.Subnet4, mode ListResourceMode) ([]*resource.Reservation4, error) {
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		if mode == ListResourceModeAPI {
			if err = setSubnet4FromDB(tx, subnet); err != nil {
				return
			}
		}

		reservations, err = getReservation4sWithCondition(tx, map[string]interface{}{
			resource.SqlColumnSubnet4: subnet.GetID(),
			resource.SqlOrderBy:       resource.SqlColumnIp,
		})
		return
	}); err != nil {
		return nil, err
	}

	if len(reservations) != 0 && len(subnet.Nodes) != 0 {
		leasesCount := getReservation4sLeasesCount(subnet.SubnetId, reservations)
		for _, reservation := range reservations {
			setReservation4LeasesUsedRatio(reservation, leasesCount[reservation.IpAddress])
		}
	}

	return reservations, nil
}

func getReservation4sWithCondition(tx restdb.Transaction, condition map[string]interface{}) ([]*resource.Reservation4, error) {
	var reservations []*resource.Reservation4
	if err := tx.Fill(condition, &reservations); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	return reservations, nil
}

func getReservation4sLeasesCount(subnetId uint64, reservations []*resource.Reservation4) map[string]uint64 {
	resp, err := getSubnet4Leases(subnetId)
	if err != nil {
		log.Warnf("get subnet4 %d leases failed: %s", subnetId, err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := reservationMapFromReservation4s(reservations)
	leasesCount := make(map[string]uint64, len(resp.GetLeases()))
	for _, lease := range resp.GetLeases() {
		if reservation, ok := reservationMap[lease.GetAddress()]; ok &&
			leaseAllocateToReservation4(lease, reservation) {
			leasesCount[lease.GetAddress()] = 1
		}
	}

	return leasesCount
}

func reservationMapFromReservation4s(reservations []*resource.Reservation4) map[string]*resource.Reservation4 {
	reservationMap := make(map[string]*resource.Reservation4, len(reservations))
	for _, reservation := range reservations {
		reservationMap[reservation.IpAddress] = reservation
	}

	return reservationMap
}

func leaseAllocateToReservation4(lease *pbdhcpagent.DHCPLease4, reservation *resource.Reservation4) bool {
	return (reservation.HwAddress != "" &&
		strings.EqualFold(reservation.HwAddress, lease.GetHwAddress())) ||
		(reservation.Hostname != "" &&
			reservation.Hostname == lease.GetHostname())
}

func setReservation4LeasesUsedRatio(reservation *resource.Reservation4, leasesCount uint64) {
	if leasesCount != 0 {
		reservation.UsedCount = leasesCount
		reservation.UsedRatio = fmt.Sprintf("%.4f",
			float64(leasesCount)/float64(reservation.Capacity))
	}
}

func (r *Reservation4Service) Get(subnet *resource.Subnet4, reservationID string) (*resource.Reservation4, error) {
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		if err = setSubnet4FromDB(tx, subnet); err != nil {
			return
		}

		reservations, err = getReservation4sWithCondition(tx, map[string]interface{}{
			restdb.IDField: reservationID,
		})
		return
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

func getReservation4LeaseCount(subnet *resource.Subnet4, reservation *resource.Reservation4) (uint64, error) {
	if len(subnet.Nodes) == 0 {
		return 0, nil
	}

	var resp *pbdhcpagent.GetLeasesCountResponse
	var err error
	if err = transport.CallDhcpAgentGrpc4(func(ctx context.Context,
		client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetReservation4LeaseCount(
			ctx, &pbdhcpagent.GetReservation4LeaseCountRequest{
				SubnetId:  subnet.SubnetId,
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

		if _, err := tx.Delete(resource.TableReservation4, map[string]interface{}{
			restdb.IDField: reservation.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, reservation.GetID(),
				pg.Error(err).Error())
		}

		return sendDeleteReservation4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, reservation)
	})
}

func checkReservation4CouldBeDeleted(tx restdb.Transaction, subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	if err := setSubnet4FromDB(tx, subnet); err != nil {
		return err
	}

	if err := setReservation4FromDB(tx, reservation); err != nil {
		return err
	}

	return checkReservation4WithLease(subnet, reservation)
}

func setReservation4FromDB(tx restdb.Transaction, reservation *resource.Reservation4) error {
	reservations, err := getReservation4sWithCondition(tx, map[string]interface{}{
		restdb.IDField: reservation.GetID()})
	if err != nil {
		return err
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

func checkReservation4WithLease(subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	if leasesCount, err := getReservation4LeaseCount(subnet, reservation); err != nil {
		return err
	} else if leasesCount != 0 {
		return errorno.ErrIPHasBeenAllocated(errorno.ErrNameDhcpReservation,
			reservation.IpAddress)
	}

	return nil
}

func sendDeleteReservation4CmdToDHCPAgent(subnetID uint64, nodes []string, reservation *resource.Reservation4) error {
	if len(nodes) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.DeleteReservation4,
		reservation4ToDeleteReservation4Request(subnetID, reservation), nil)
}

func (r *Reservation4Service) Update(subnetId string, reservation *resource.Reservation4) error {
	if err := util.ValidateStrings(util.RegexpTypeComma, reservation.Comment); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservation4, map[string]interface{}{
			resource.SqlColumnComment: reservation.Comment,
		}, map[string]interface{}{restdb.IDField: reservation.GetID()}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, reservation.GetID(),
				pg.Error(err).Error())
		} else if rows == 0 {
			return errorno.ErrNotFound(errorno.ErrNameDhcpReservation, reservation.GetID())
		}

		return nil
	})
}

func GetReservation4sByPrefix(prefix string) ([]*resource.Reservation4, error) {
	if subnet4, err := GetSubnet4ByPrefix(prefix); err != nil {
		return nil, err
	} else {
		return listReservation4s(subnet4, ListResourceModeGRPC)
	}
}

func BatchCreateReservation4s(prefix string, reservations []*resource.Reservation4) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet, err := getSubnet4WithPrefix(tx, prefix)
		if err != nil {
			return err
		}

		_, err = batchCreateReservation4s(tx, subnet, reservations, CreateReservationModeCreate)
		return err
	})
}

func batchCreateReservation4s(tx restdb.Transaction, subnet *resource.Subnet4, reservations []*resource.Reservation4, mode CreateReservationMode) (*excel.ImportResult, error) {
	reservedpools, err := getReservedPool4sWithSubnetId(tx, subnet.GetID())
	if err != nil {
		return nil, err
	}

	oldReservations, err := getReservation4sWithSubnetId(tx, subnet.GetID())
	if err != nil {
		return nil, err
	}

	pools, err := getPool4sWithSubnetId(tx, subnet.GetID())
	if err != nil {
		return nil, err
	}

	reservation4Identifier := Reservation4IdentifierFromReservations(oldReservations)
	reservationValues := make([][]interface{}, 0, len(reservations))
	poolsCapacity := make(map[string]uint64, len(pools))
	result := &excel.ImportResult{}
	validReservations := make([]*resource.Reservation4, 0, len(reservations))
	for _, reservation := range reservations {
		if mode != CreateReservationModeImport {
			if err := reservation.Validate(); err != nil {
				return nil, err
			}

			if !subnet.Ipnet.Contains(reservation.Ip) {
				return nil, errorno.ErrNotBelongTo(errorno.ErrNameDhcpReservation,
					errorno.ErrNameNetworkV4, reservation.IpAddress, subnet.Subnet)
			}
		}

		if err := checkReservation4IpConflictWithReservedPool4s(reservation,
			reservedpools); err != nil {
			if mode == CreateReservationModeImport {
				addFailDataToResponse(result, TableHeaderReservation4FailLen,
					localizationReservation4ToStrSlice(reservation),
					errorno.TryGetErrorCNMsg(err))
				continue
			} else {
				return nil, err
			}
		}

		if err := reservation4Identifier.Add(reservation); err != nil {
			if mode == CreateReservationModeImport {
				addFailDataToResponse(result, TableHeaderReservation4FailLen,
					localizationReservation4ToStrSlice(reservation),
					errorno.TryGetErrorCNMsg(err))
				continue
			} else {
				return nil, err
			}
		}

		recalculateSubnetAndPoolsCapacityWithReservation4(subnet, pools,
			reservation, poolsCapacity, true)
		reservation.Subnet4 = subnet.GetID()
		reservationValues = append(reservationValues, reservation.GenCopyValues())
		validReservations = append(validReservations, reservation)
	}

	if err := updateSubnet4AndPool4sCapacity(tx, subnet, poolsCapacity); err != nil {
		return nil, err
	}

	if _, err := tx.CopyFromEx(resource.TableReservation4, resource.Reservation4Columns,
		reservationValues); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameInsert,
			string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	if err := sendCreateReservation4sCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
		validReservations); err != nil {
		return nil, err
	}

	return result, nil
}

func getReservedPool4sWithSubnetId(tx restdb.Transaction, subnetId string) ([]*resource.ReservedPool4, error) {
	return getReservedPool4sWithCondition(tx,
		map[string]interface{}{resource.SqlColumnSubnet4: subnetId})
}

func getReservation4sWithSubnetId(tx restdb.Transaction, subnetId string) ([]*resource.Reservation4, error) {
	return getReservation4sWithCondition(tx,
		map[string]interface{}{resource.SqlColumnSubnet4: subnetId})
}

func getPool4sWithSubnetId(tx restdb.Transaction, subnetId string) ([]*resource.Pool4, error) {
	return getPool4sWithCondition(tx,
		map[string]interface{}{resource.SqlColumnSubnet4: subnetId})
}

func checkReservation4IpConflictWithReservedPool4s(reservation *resource.Reservation4, reservedpools []*resource.ReservedPool4) error {
	for _, reservedpool := range reservedpools {
		if reservedpool.ContainsIp(reservation.Ip) {
			return errorno.ErrConflict(errorno.ErrNameDhcpReservation,
				errorno.ErrNameDhcpReservedPool, reservation.String(), reservedpool.String())
		}
	}

	return nil
}

func recalculateSubnetAndPoolsCapacityWithReservation4(subnet *resource.Subnet4, pools []*resource.Pool4, reservation *resource.Reservation4, poolsCapacity map[string]uint64, isCreate bool) {
	var conflictPool *resource.Pool4
	for _, pool := range pools {
		if pool.Capacity != 0 && pool.ContainsIp(reservation.Ip) {
			conflictPool = pool
			break
		}
	}

	if conflictPool == nil {
		if isCreate {
			subnet.Capacity += reservation.Capacity
		} else {
			subnet.Capacity -= reservation.Capacity
		}
	} else {
		capacity, ok := poolsCapacity[conflictPool.GetID()]
		if !ok {
			capacity = conflictPool.Capacity
		}

		if isCreate {
			poolsCapacity[conflictPool.GetID()] = capacity - reservation.Capacity
		} else {
			poolsCapacity[conflictPool.GetID()] = capacity + reservation.Capacity
		}
	}
}

func updateSubnet4AndPool4sCapacity(tx restdb.Transaction, subnet *resource.Subnet4, poolsCapacity map[string]uint64) error {
	if err := updateResourceCapacity(tx, resource.TableSubnet4, subnet.GetID(),
		subnet.Capacity, errorno.ErrNameNetworkV4); err != nil {
		return err
	}

	return batchUpdatePool4sCapacity(tx, poolsCapacity)
}

func sendCreateReservation4sCmdToDHCPAgent(subnetID uint64, nodes []string, reservations []*resource.Reservation4) error {
	if len(nodes) == 0 || len(reservations) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.CreateReservation4s,
		reservation4sToCreateReservations4Request(subnetID, reservations),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservation4s,
				reservation4sToDeleteReservations4Request(subnetID, reservations)); err != nil {
				log.Errorf("create subnet4 %d reservation4 %s failed, rollback %v failed: %s",
					subnetID, reservations[0].String(), nodesForSucceed, err.Error())
			}
		})
}

func reservation4sToCreateReservations4Request(subnetID uint64, reservations []*resource.Reservation4) *pbdhcpagent.CreateReservations4Request {
	pbReservations := make([]*pbdhcpagent.CreateReservation4Request, len(reservations))
	for i, reservation := range reservations {
		pbReservations[i] = reservation4ToCreateReservation4Request(subnetID, reservation)
	}

	return &pbdhcpagent.CreateReservations4Request{
		SubnetId:     subnetID,
		Reservations: pbReservations,
	}
}

func reservation4sToDeleteReservations4Request(subnetID uint64, reservations []*resource.Reservation4) *pbdhcpagent.DeleteReservations4Request {
	pbReservations := make([]*pbdhcpagent.DeleteReservation4Request, len(reservations))
	for i, reservation := range reservations {
		pbReservations[i] = reservation4ToDeleteReservation4Request(subnetID, reservation)
	}

	return &pbdhcpagent.DeleteReservations4Request{
		SubnetId:     subnetID,
		Reservations: pbReservations,
	}
}

func (s *Reservation4Service) BatchDeleteReservation4s(subnetId string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet, err := getSubnet4FromDB(tx, subnetId)
		if err != nil {
			return err
		}

		reservations, err := getReservation4sWithCondition(tx, map[string]interface{}{
			restdb.IDField: restdb.FillValue{Operator: restdb.OperatorAny, Value: ids}})
		if err != nil {
			return err
		} else if len(ids) != len(reservations) {
			return errorno.ErrResourceNotFound(errorno.ErrNameDhcpReservation)
		}

		pools, err := getPool4sWithSubnetId(tx, subnet.GetID())
		if err != nil {
			return err
		}

		leaseMap, err := getLease4MapFromSubnet4Leases(subnet)
		if err != nil {
			return err
		}

		poolsCapacity := make(map[string]uint64, len(pools))
		for _, reservation := range reservations {
			if lease, ok := leaseMap[reservation.IpAddress]; ok &&
				leaseAllocateToReservation4(lease, reservation) {
				return errorno.ErrIPHasBeenAllocated(errorno.ErrNameDhcpReservation,
					reservation.IpAddress)
			}

			recalculateSubnetAndPoolsCapacityWithReservation4(subnet, pools,
				reservation, poolsCapacity, false)
		}

		if err = updateSubnet4AndPool4sCapacity(tx, subnet, poolsCapacity); err != nil {
			return err
		}

		if _, err = tx.Delete(resource.TableReservation4, map[string]interface{}{
			restdb.IDField: restdb.FillValue{
				Operator: restdb.OperatorAny, Value: ids}}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete,
				string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}

		return sendDeleteReservation4sCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes,
			reservations)
	})
}

func getLease4MapFromSubnet4Leases(subnet *resource.Subnet4) (map[string]*pbdhcpagent.DHCPLease4, error) {
	resp, err := getSubnet4Leases(subnet.SubnetId)
	if err != nil {
		return nil, errorno.ErrNetworkError(errorno.ErrNameLease, err.Error())
	}

	if len(resp.GetLeases()) == 0 {
		return nil, nil
	}

	leaseMap := make(map[string]*pbdhcpagent.DHCPLease4, len(resp.GetLeases()))
	for _, lease := range resp.GetLeases() {
		leaseMap[lease.GetAddress()] = lease
	}

	return leaseMap, nil
}

func sendDeleteReservation4sCmdToDHCPAgent(subnetID uint64, nodes []string, reservations []*resource.Reservation4) error {
	if len(nodes) == 0 || len(reservations) == 0 {
		return nil
	}

	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.DeleteReservation4s,
		reservation4sToDeleteReservations4Request(subnetID, reservations), nil)
}

func (s *Reservation4Service) ImportExcel(file *excel.ImportFile, subnetId string) (interface{}, error) {
	var subnet4s []*resource.Subnet4
	if err := db.GetResources(map[string]interface{}{restdb.IDField: subnetId},
		&subnet4s); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
	} else if len(subnet4s) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetwork, subnetId)
	}

	subnet := subnet4s[0]
	response := &excel.ImportResult{}
	defer sendImportFieldResponse(Reservation4ImportFileNamePrefix,
		TableHeaderReservation4Fail, response)
	reservations, err := s.parseReservation4sFromFile(file.Name, subnet, response)
	if err != nil {
		return response, err
	}

	if len(reservations) == 0 {
		return response, nil
	}

	if err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		response, err = batchCreateReservation4s(tx, subnet, reservations,
			CreateReservationModeImport)
		return err
	}); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *Reservation4Service) parseReservation4sFromFile(fileName string, subnet4 *resource.Subnet4,
	response *excel.ImportResult) ([]*resource.Reservation4, error) {
	contents, err := excel.ReadExcelFile(fileName)
	if err != nil {
		return nil, errorno.ErrReadFile(fileName, err.Error())
	}

	if len(contents) < 2 {
		return nil, nil
	}

	tableHeaderFields, err := excel.ParseTableHeader(contents[0],
		TableHeaderReservation4, Reservation4MandatoryFields)
	if err != nil {
		return nil, errorno.ErrInvalidTableHeader()
	}

	response.InitData(len(contents) - 1)
	fieldcontents := contents[1:]
	reservations := make([]*resource.Reservation4, 0, len(fieldcontents))
	for j, fields := range fieldcontents {
		fields, missingMandatory, emptyLine := excel.ParseTableFields(fields,
			tableHeaderFields, Reservation4MandatoryFields)
		if emptyLine {
			continue
		} else if missingMandatory {
			addFailDataToResponse(response, TableHeaderReservation4FailLen,
				localizationReservation4ToStrSlice(&resource.Reservation4{}),
				errorno.ErrMissingMandatory(j+2, Reservation4MandatoryFields).ErrorCN())
			continue
		}

		reservation4, err := s.parseReservation4sFromFields(fields, tableHeaderFields)
		if err != nil {
			addFailDataToResponse(response, TableHeaderReservation4FailLen,
				localizationReservation4ToStrSlice(reservation4), errorno.TryGetErrorCNMsg(err))
			continue
		}

		if err = reservation4.Validate(); err != nil {
			addFailDataToResponse(response, TableHeaderReservation4FailLen,
				localizationReservation4ToStrSlice(reservation4), errorno.TryGetErrorCNMsg(err))
			continue
		}

		if !subnet4.Ipnet.Contains(reservation4.Ip) {
			addFailDataToResponse(response, TableHeaderReservation4FailLen,
				localizationReservation4ToStrSlice(reservation4),
				errorno.ErrNotBelongTo(errorno.ErrNameIp, errorno.ErrNameNetwork,
					reservation4.Ip.String(), subnet4.Ipnet.String()).ErrorCN())
			continue
		}

		reservations = append(reservations, reservation4)
	}

	return reservations, nil
}

func (s *Reservation4Service) parseReservation4sFromFields(fields, tableHeaderFields []string) (*resource.Reservation4, error) {
	reservation4 := &resource.Reservation4{}
	var deviceFlag string
	var err error
	for i, field := range fields {
		if excel.IsSpaceField(field) {
			continue
		}

		field = strings.TrimSpace(field)
		switch tableHeaderFields[i] {
		case FieldNameIpAddress:
			reservation4.IpAddress = field
		case FieldNameReservation4DeviceFlag:
			deviceFlag = field
		case FieldNameReservation4DeviceFlagValue:
			if deviceFlag == ReservationFlagMac {
				reservation4.HwAddress = field
			} else if deviceFlag == ReservationFlagHostName {
				reservation4.Hostname = field
			} else {
				err = errorno.ErrInvalidParams(errorno.ErrNameDeviceFlag, field)
			}
		case FieldNameComment:
			reservation4.Comment = field
		}
	}

	return reservation4, err
}

func (s *Reservation4Service) ExportExcel(subnetId string) (*excel.ExportFile, error) {
	var reservation4s []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnetId},
			&reservation4s)
		return err
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	strMatrix := make([][]string, 0, len(reservation4s))
	for _, reservation4 := range reservation4s {
		strMatrix = append(strMatrix, localizationReservation4ToStrSlice(reservation4))
	}

	if filepath, err := excel.WriteExcelFile(Reservation4FileNamePrefix+
		time.Now().Format(excel.TimeFormat), TableHeaderReservation4, strMatrix,
		getOpt(Reservation4DropList, len(strMatrix)+1)); err != nil {
		return nil, errorno.ErrExport(errorno.ErrNameDhcpReservation, err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func (s *Reservation4Service) ExportExcelTemplate() (*excel.ExportFile, error) {
	if filepath, err := excel.WriteExcelFile(Reservation4TemplateFileName,
		TableHeaderReservation4, TemplateReservation4, getOpt(Reservation4DropList,
			len(TemplateReservation4)+1)); err != nil {
		return nil, errorno.ErrExportTmp(errorno.ErrNameDhcpReservation, err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func createReservationsFromDynamicLeases(v4Map map[string][]*resource.Reservation4, v6Map map[string][]*resource.Reservation6) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		v4SubnetMap := make(map[string]*resource.Subnet4, len(v4Map))
		for subnetId, reservation4s := range v4Map {
			subnet4, err := getSubnet4FromDB(tx, subnetId)
			if err != nil {
				return err
			}

			if _, err = batchCreateReservation4s(tx, subnet4, reservation4s,
				CreateReservationModeConvert); err != nil {
				return err
			}

			v4SubnetMap[subnetId] = subnet4
		}

		v6SubnetMap := make(map[string]*resource.Subnet6, len(v6Map))
		for subnetId, reservation6s := range v6Map {
			subnet6, err := getSubnet6FromDB(tx, subnetId)
			if err != nil {
				return err
			}

			if _, err = batchCreateReservation6s(tx, subnet6, reservation6s,
				CreateReservationModeConvert); err != nil {
				return err
			}

			v6SubnetMap[subnetId] = subnet6
		}

		for subnetId, subnet4 := range v4SubnetMap {
			if err := sendCreateReservation4sCmdToDHCPAgent(subnet4.SubnetId, subnet4.Nodes,
				v4Map[subnetId]); err != nil {
				return err
			}
		}

		for subnetId, subnet6 := range v6SubnetMap {
			if err := sendCreateReservation6sCmdToDHCPAgent(subnet6.SubnetId, subnet6.Nodes,
				v6Map[subnetId]); err != nil {
				return err
			}
		}

		return nil
	})
}
