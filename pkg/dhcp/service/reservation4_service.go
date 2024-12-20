package service

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
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

const (
	InvalidHeaderPrefix = "the file table header field "
	InvalidHeaderSuffix = " is invalid"
)

type Reservation4Service struct{}

func NewReservation4Service() *Reservation4Service {
	return &Reservation4Service{}
}

func (r *Reservation4Service) Create(subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := batchCreateReservationV4s(tx, subnet, reservation); err != nil {
			return err
		}
		return sendCreateReservation4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, reservation)
	})
}

func checkReservation4CouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet4, reservations ...*resource.Reservation4) error {
	for _, reservation := range reservations {
		if !subnet.Ipnet.Contains(reservation.Ip) {
			return errorno.ErrNotBelongTo(errorno.ErrNameDhcpReservation, errorno.ErrNameNetworkV4,
				reservation.IpAddress, subnet.Subnet)
		}
	}

	if err := checkReservation4ConflictWithReservedPool4(tx, subnet.GetID(), reservations...); err != nil {
		return err
	}

	return nil
}

func checkReservation4sInUsed(tx restdb.Transaction, subnetId string, reservations ...*resource.Reservation4) error {
	reservationInfoMap, reservationAddressMap, err := getReservation4UniqueMap(tx, subnetId)
	if err != nil {
		return err
	}

	for _, reservation := range reservations {
		if err = reservation.CheckUnique(reservationInfoMap, reservationAddressMap); err != nil {
			return err
		}
	}
	return nil
}

func getReservation4UniqueMap(tx restdb.Transaction, subnetId string) (map[string]struct{}, map[string]struct{}, error) {
	var dbReservations resource.Reservation4s
	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnSubnet4: subnetId,
	}, &dbReservations); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	reservationInfoMap := make(map[string]struct{}, len(dbReservations))
	reservationAddressMap := make(map[string]struct{}, len(dbReservations))
	for _, reservation := range dbReservations {
		reservationInfoMap[reservation.GetUniqueKey()] = struct{}{}
		reservationAddressMap[reservation.IpAddress] = struct{}{}
	}
	return reservationInfoMap, reservationAddressMap, nil
}

func checkReservation4ConflictWithReservedPool4(tx restdb.Transaction, subnetId string, reservations ...*resource.Reservation4) error {
	var reservedpools []*resource.ReservedPool4
	if err := tx.FillEx(&reservedpools,
		"select * from gr_reserved_pool4 where subnet4 = $1",
		subnetId); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
	}

	for _, reservation := range reservations {
		for _, reservationPool := range reservedpools {
			if isIPInRange(reservation.Ip, reservationPool.BeginIp, reservationPool.EndIp) {
				return errorno.ErrConflict(errorno.ErrNameDhcpReservation, errorno.ErrNameDhcpReservedPool,
					reservation.String(), reservationPool.String())
			}
		}
	}
	return nil
}

func ipToUint32(ip net.IP) uint32 {
	if ipv4_ := ip.To4(); ipv4_ == nil {
		return 0
	} else {
		return binary.BigEndian.Uint32(ipv4_)
	}
}

func isIPInRange(ip, start, end net.IP) bool {
	ipNum := ipToUint32(ip)
	startNum := ipToUint32(start)
	endNum := ipToUint32(end)

	return ipNum >= startNum && ipNum <= endNum
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

func updateSubnet4OrPool4CapacityWithReservation4s(tx restdb.Transaction, subnet *resource.Subnet4,
	reservation *resource.Reservation4, isCreate bool, updatePool4Map map[string]*resource.Pool4) error {
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
	} else {
		if _, ok := updatePool4Map[conflictPools[0].GetID()]; !ok {
			updatePool4Map[conflictPools[0].GetID()] = conflictPools[0]
		}
		if isCreate {
			updatePool4Map[conflictPools[0].GetID()].Capacity -= reservation.Capacity
		} else {
			updatePool4Map[conflictPools[0].GetID()].Capacity += reservation.Capacity
		}
	}

	return nil
}

func sendCreateReservation4CmdToDHCPAgent(subnetID uint64, nodes []string, reservations ...*resource.Reservation4) error {
	if len(reservations) == 0 {
		return nil
	}
	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.CreateReservation4s,
		reservation4sToCreateReservations4Request(subnetID, reservations),
		func(nodesForSucceed []string) {
			if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
				nodesForSucceed, kafka.DeleteReservation4s,
				reservation4sToDeleteReservations4Request(subnetID, reservations)); err != nil {
				log.Errorf("create subnet4 %d reservation4 %s failed, rollback with nodes %v failed: %s",
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
			resource.SqlOrderBy:       resource.SqlColumnIp}, &reservations); err != nil {
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
		log.Warnf("get subnet4 %d leases failed: %s", subnetId, err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := reservationMapFromReservation4s(reservations)
	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if reservation, ok := reservationMap[lease.GetAddress()]; ok &&
			(reservation.HwAddress == "" || strings.EqualFold(reservation.HwAddress, lease.GetHwAddress())) &&
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

func checkReservation4WithLease(subnet *resource.Subnet4, reservation *resource.Reservation4) error {
	if leasesCount, err := getReservation4LeaseCount(subnet, reservation); err != nil {
		return err
	} else if leasesCount != 0 {
		return errorno.ErrIPHasBeenAllocated(errorno.ErrNameDhcpReservation, reservation.IpAddress)
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

func sendDeleteReservation4CmdToDHCPAgent(subnetID uint64, nodes []string, reservations ...*resource.Reservation4) error {
	return kafka.SendDHCPCmdWithNodes(true, nodes, kafka.DeleteReservation4s,
		reservation4sToDeleteReservations4Request(subnetID, reservations), nil)
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

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err = batchCreateReservationV4s(tx, subnet, reservations...); err != nil {
			return err
		}
		return sendCreateReservation4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, reservations...)
	})
}

func batchCreateReservationV4s(tx restdb.Transaction, subnet *resource.Subnet4, reservations ...*resource.Reservation4) error {
	for _, reservation := range reservations {
		if err := reservation.Validate(); err != nil {
			return err
		}
	}

	if err := checkReservation4CouldBeCreated(tx, subnet, reservations...); err != nil {
		return err
	}

	if err := checkReservation4sInUsed(tx, subnet.GetID(), reservations...); err != nil {
		return err
	}

	updatePool4Map := make(map[string]*resource.Pool4, len(reservations))
	values := make([][]interface{}, 0, len(reservations))
	for _, reservation := range reservations {
		if err := updateSubnet4OrPool4CapacityWithReservation4s(tx, subnet,
			reservation, true, updatePool4Map); err != nil {
			return err
		}

		reservation.Subnet4 = subnet.GetID()
		values = append(values, reservation.GenCopyValues())
	}

	if err := updateSubnet4AndPool4Capacity(tx, subnet, updatePool4Map); err != nil {
		return err
	}

	if _, err := tx.CopyFromEx(resource.TableReservation4, resource.Reservation4Columns, values); err != nil {
		log.Warnf("%s", err.Error())
		return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	return nil
}

func (s *Reservation4Service) BatchDeleteReservation4s(subnetId string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	var reservations resource.Reservation4s
	updatePool4Map := make(map[string]*resource.Pool4, len(ids))
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet, err := getSubnet4FromDB(tx, subnetId)
		if err != nil {
			return err
		}

		if err = tx.Fill(map[string]interface{}{restdb.IDField: restdb.FillValue{
			Operator: restdb.OperatorAny, Value: ids}},
			&reservations); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		} else if len(ids) != len(reservations) {
			return errorno.ErrResourceNotFound(errorno.ErrNameDhcpReservation)
		}

		for _, reservation := range reservations {
			if err = checkReservation4WithLease(subnet, reservation); err != nil {
				return err
			}

			if err = updateSubnet4OrPool4CapacityWithReservation4s(tx, subnet, reservation, false, updatePool4Map); err != nil {
				return err
			}
		}

		if err = updateSubnet4AndPool4Capacity(tx, subnet, updatePool4Map); err != nil {
			return err
		}

		if _, err = tx.Delete(resource.TableReservation4,
			map[string]interface{}{restdb.IDField: restdb.FillValue{
				Operator: restdb.OperatorAny,
				Value:    reservations.GetIds(),
			}}); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameDelete, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}

		return sendDeleteReservation4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, reservations...)
	})
}

func updateSubnet4AndPool4Capacity(tx restdb.Transaction, subnet *resource.Subnet4,
	updatePool4Map map[string]*resource.Pool4) error {
	if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
		resource.SqlColumnCapacity: subnet.Capacity,
	}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameUpdate, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
	}

	if len(updatePool4Map) > 0 {
		updateMap := make(map[string]string, len(updatePool4Map))
		for _, pool := range updatePool4Map {
			updateMap[pool.GetID()] = strings.Join([]string{"('", pool.GetID(), "',",
				strconv.FormatUint(pool.Capacity, 10), ")"}, "")
		}
		if err := util.BatchUpdateById(tx, string(resource.TablePool4), []string{resource.SqlColumnCapacity}, updateMap); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameUpdate, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
		}
	}
	return nil
}

func (s *Reservation4Service) ImportExcel(file *excel.ImportFile, subnetId string) (interface{}, error) {
	var subnet4s []*resource.Subnet4
	if err := db.GetResources(map[string]interface{}{restdb.IDField: subnetId},
		&subnet4s); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
	} else if len(subnet4s) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetwork, subnetId)
	}

	response := &excel.ImportResult{}
	defer sendImportFieldResponse(Reservation4ImportFileNamePrefix, TableHeaderReservation4Fail, response)
	reservations, err := s.parseReservation4sFromFile(file.Name, subnet4s[0], response)
	if err != nil {
		return response, err
	}

	if len(reservations) == 0 {
		return response, nil
	}

	var reservationInfoMap, reservationAddressMap map[string]struct{}
	validReservations := make([]*resource.Reservation4, 0, len(reservations))
	updatePool4Map := make(map[string]*resource.Pool4, len(reservations))
	values := make([][]interface{}, 0, len(reservations))
	if err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if reservationInfoMap, reservationAddressMap, err = getReservation4UniqueMap(tx, subnetId); err != nil {
			return err
		}

		for _, reservation := range reservations {
			if err = checkReservation4CouldBeCreated(tx, subnet4s[0], reservation); err != nil {
				addFailDataToResponse(response, TableHeaderReservation4FailLen,
					localizationReservation4ToStrSlice(reservation), errorno.TryGetErrorCNMsg(err))
				continue
			} else if err = reservation.CheckUnique(reservationInfoMap, reservationAddressMap); err != nil {
				addFailDataToResponse(response, TableHeaderReservation4FailLen,
					localizationReservation4ToStrSlice(reservation), errorno.TryGetErrorCNMsg(err))
				continue
			}

			if err = updateSubnet4OrPool4CapacityWithReservation4s(tx, subnet4s[0],
				reservation, true, updatePool4Map); err != nil {
				return err
			}

			reservation.Subnet4 = subnet4s[0].GetID()
			values = append(values, reservation.GenCopyValues())
			validReservations = append(validReservations, reservation)
		}

		if err = updateSubnet4AndPool4Capacity(tx, subnet4s[0], updatePool4Map); err != nil {
			return err
		}

		if _, err = tx.CopyFromEx(resource.TableReservation4, resource.Reservation4Columns, values); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
		}

		return sendCreateReservation4CmdToDHCPAgent(subnet4s[0].SubnetId, subnet4s[0].Nodes, validReservations...)
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
	subnetReservations := make([]*resource.Reservation4, 0, len(fieldcontents))
	reservationMap := make(map[string]struct{}, len(fieldcontents))
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

		if _, ok := reservationMap[reservation4.IpAddress]; ok {
			addFailDataToResponse(response, TableHeaderReservation4FailLen,
				localizationReservation4ToStrSlice(reservation4), errorno.ErrDuplicate(errorno.ErrNameIp, reservation4.IpAddress).ErrorCN())
			continue
		}

		reservationMap[reservation4.IpAddress] = struct{}{}
		subnetReservations = append(subnetReservations, reservation4)
	}

	return subnetReservations, nil
}

func getInvalidHeader(errMsg string) string {
	return strings.TrimSuffix(strings.TrimPrefix(errMsg, InvalidHeaderPrefix), InvalidHeaderSuffix)
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
		err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnetId}, &reservation4s)
		return err
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
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
		TableHeaderReservation4, TemplateReservation4, getOpt(Reservation4DropList, len(TemplateReservation4)+1)); err != nil {
		return nil, errorno.ErrExportTmp(errorno.ErrNameDhcpReservation, err.Error())
	} else {
		return &excel.ExportFile{Path: filepath}, nil
	}
}

func createReservationsBySubnet(v4Map map[string][]*resource.Reservation4, v6Map map[string][]*resource.Reservation6) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		v4SubnetMap := make(map[string]*resource.Subnet4, len(v4Map))
		for subnetId, reservation4s := range v4Map {
			subnet4, err := getSubnet4FromDB(tx, subnetId)
			if err != nil {
				return err
			}
			if err = batchCreateReservationV4s(tx, subnet4, reservation4s...); err != nil {
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
			if err = batchCreateReservation6s(tx, subnet6, reservation6s); err != nil {
				return err
			}
			v6SubnetMap[subnetId] = subnet6
		}

		for subnetId, subnet4 := range v4SubnetMap {
			if err := sendCreateReservation4CmdToDHCPAgent(subnet4.SubnetId, subnet4.Nodes, v4Map[subnetId]...); err != nil {
				return err
			}
		}
		for subnetId, subnet6 := range v6SubnetMap {
			if err := sendCreateReservation6CmdToDHCPAgent(subnet6.SubnetId, subnet6.Nodes, v6Map[subnetId]...); err != nil {
				return err
			}
		}

		return nil
	})
}
