package service

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/log"
	"github.com/linkingthing/cement/slice"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type SubnetLease6Service struct{}

func NewSubnetLease6Service() *SubnetLease6Service {
	return &SubnetLease6Service{}
}

func (l *SubnetLease6Service) List(subnet *resource.Subnet6, ip string) ([]*resource.SubnetLease6, error) {
	return ListSubnetLease6(subnet, ip)
}

func (l *SubnetLease6Service) ActionListToReservation(subnet *resource.Subnet6, input *resource.ConvToReservationInput) (*resource.ConvToReservationInput, error) {
	if len(input.Addresses) == 0 {
		return &resource.ConvToReservationInput{Data: []resource.ConvToReservationItem{}}, nil
	}

	leases, err := ListSubnetLease6(subnet, "")
	if err != nil {
		return nil, err
	}

	toReservationLeases := make([]*resource.SubnetLease6, 0, len(leases))
	for _, lease := range leases {
		if lease.AllocateMode == pbdhcpagent.LeaseAllocateMode_DYNAMIC.String() &&
			slice.SliceIndex(input.Addresses, lease.Address) >= 0 {
			toReservationLeases = append(toReservationLeases, lease)
		}
	}

	switch input.ReservationType {
	case resource.ReservationTypeMac:
		return l.listToReservationWithMac(toReservationLeases)
	case resource.ReservationTypeHostname:
		return l.listToReservationWithHostname(toReservationLeases)
	default:
		return nil, fmt.Errorf("unsupported type %q", input.ReservationType)
	}
}

func (l *SubnetLease6Service) listToReservationWithMac(leases []*resource.SubnetLease6) (*resource.ConvToReservationInput, error) {
	reservationLeases := make([]*resource.SubnetLease6, 0, len(leases))
	hwAddresses := make([]string, 0, len(leases))
	uniqueHwAddresses := make(map[string]struct{}, len(leases))
	for _, lease := range leases {
		if lease.HwAddress != "" {
			reservationLeases = append(reservationLeases, lease)
			if _, ok := uniqueHwAddresses[lease.HwAddress]; !ok {
				uniqueHwAddresses[lease.HwAddress] = struct{}{}
				hwAddresses = append(hwAddresses, lease.HwAddress)
			}
		}
	}

	lease4s, err := GetSubnets4LeasesWithMacs(hwAddresses)
	if err != nil {
		log.Warnf("get leases of subnets with macs failed: %s", err.Error())
	}

	result := make([]resource.ConvToReservationItem, 0, len(reservationLeases))
	for _, lease := range reservationLeases {
		dualStack := make([]string, 0, len(lease4s))
		for _, lease4 := range lease4s {
			if lease4.HwAddress == lease.HwAddress &&
				lease4.AllocateMode == pbdhcpagent.LeaseAllocateMode_DYNAMIC.String() {
				dualStack = append(dualStack, lease4.Address)
			}
		}

		result = append(result, resource.ConvToReservationItem{
			Address:    lease.Address,
			DualStacks: dualStack,
			HwAddress:  lease.HwAddress,
			Hostname:   lease.Hostname,
			Duid:       lease.Duid,
		})
	}

	return &resource.ConvToReservationInput{Data: result}, nil
}

func (l *SubnetLease6Service) listToReservationWithHostname(leases []*resource.SubnetLease6) (*resource.ConvToReservationInput, error) {
	result := make([]resource.ConvToReservationItem, 0, len(leases))
	for _, lease := range leases {
		if lease.Hostname != "" {
			result = append(result, resource.ConvToReservationItem{
				Address:   lease.Address,
				HwAddress: lease.HwAddress,
				Hostname:  lease.Hostname,
				Duid:      lease.Duid,
			})
		}
	}

	return &resource.ConvToReservationInput{Data: result}, nil
}

func (l *SubnetLease6Service) ActionDynamicToReservation(subnet *resource.Subnet6, input *resource.ConvToReservationInput) error {
	if len(input.Data) == 0 {
		return nil
	}

	leases, err := ListSubnetLease6(subnet, "")
	if err != nil {
		return err
	}

	reservations, hwAddresses, ipv4MacMap, err := l.getReservationFromLease(leases, input)
	if err != nil {
		return err
	}

	v6ReservationMap := map[string][]*resource.Reservation6{subnet.GetID(): reservations}
	if !input.BothV4V6 || input.ReservationType != resource.ReservationTypeMac {
		return createReservationsFromDynamicLeases(nil, v6ReservationMap)
	}

	lease4s, err := GetSubnets4LeasesWithMacs(hwAddresses)
	if err != nil {
		return err
	}

	v4ReservationMap := make(map[string][]*resource.Reservation4, len(lease4s))
	for _, lease4 := range lease4s {
		expectMac, exist := ipv4MacMap[lease4.Address]
		if !exist {
			continue
		} else if lease4.HwAddress != expectMac {
			return errorno.ErrChanged(errorno.ErrNameMac, lease4.Address, expectMac, lease4.HwAddress)
		}

		delete(ipv4MacMap, lease4.Address)
		if lease4.AllocateMode == pbdhcpagent.LeaseAllocateMode_RESERVATION.String() {
			continue
		}

		v4ReservationMap[lease4.Subnet4] = append(v4ReservationMap[lease4.Subnet4], &resource.Reservation4{
			IpAddress: lease4.Address,
			HwAddress: lease4.HwAddress,
		})
	}

	for ipv4, mac := range ipv4MacMap {
		return errorno.ErrNoResourceWith(errorno.ErrNameLease, errorno.ErrNameMac, ipv4, mac)
	}

	return createReservationsFromDynamicLeases(v4ReservationMap, v6ReservationMap)
}

func (l *SubnetLease6Service) getReservationFromLease(leases []*resource.SubnetLease6, input *resource.ConvToReservationInput) ([]*resource.Reservation6, []string, map[string]string, error) {
	ipLeaseMap := make(map[string]*resource.SubnetLease6, len(leases))
	for _, item := range leases {
		ipLeaseMap[item.Address] = item
	}

	reservations := make([]*resource.Reservation6, 0, len(input.Data))
	hwAddresses := make([]string, 0, len(input.Data))
	ipv4MacMap := make(map[string]string, len(input.Data))
	reservationIdAndAddresses := make(map[string][]string, len(input.Data))
	for _, item := range input.Data {
		lease := ipLeaseMap[item.Address]
		if lease == nil {
			return nil, nil, nil, errorno.ErrNotFound(errorno.ErrNameLease, item.Address)
		}

		var reservationId string
		switch input.ReservationType {
		case resource.ReservationTypeMac:
			if reservationId = item.HwAddress; reservationId == "" {
				return nil, nil, nil, errorno.ErrNotFound(errorno.ErrNameMac, item.Address)
			} else if reservationId != lease.HwAddress {
				return nil, nil, nil, errorno.ErrChanged(errorno.ErrNameMac, item.Address, reservationId, lease.HwAddress)
			}
			hwAddresses = append(hwAddresses, reservationId)
		case resource.ReservationTypeHostname:
			if reservationId = item.Hostname; reservationId == "" {
				return nil, nil, nil, errorno.ErrNotFound(errorno.ErrNameHostname, item.Address)
			} else if reservationId != lease.Hostname {
				return nil, nil, nil, errorno.ErrChanged(errorno.ErrNameHostname, item.Address, reservationId, lease.Hostname)
			}
		default:
			return nil, nil, nil, fmt.Errorf("unsupported type %q", input.ReservationType)
		}

		reservationIdAndAddresses[reservationId] = append(reservationIdAndAddresses[reservationId], item.Address)
		for _, ipv4 := range item.DualStacks {
			ipv4MacMap[ipv4] = item.HwAddress
		}
	}

	for reservationId, addresses := range reservationIdAndAddresses {
		reservation6 := &resource.Reservation6{IpAddresses: addresses}
		switch input.ReservationType {
		case resource.ReservationTypeMac:
			reservation6.HwAddress = reservationId
		case resource.ReservationTypeHostname:
			reservation6.Hostname = reservationId
		}

		reservations = append(reservations, reservation6)
	}

	return reservations, hwAddresses, ipv4MacMap, nil
}

func ListSubnetLease6(subnet *resource.Subnet6, ip string) ([]*resource.SubnetLease6, error) {
	hasAddressFilter := ip != ""
	if hasAddressFilter {
		if _, err := gohelperip.ParseIPv6(ip); err != nil {
			return nil, nil
		}
	}

	var subnet6SubnetId uint64
	var reservations []*resource.Reservation6
	var reclaimedSubnetLeases []*resource.SubnetLease6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnet.GetID())
		if err != nil {
			return err
		} else if len(subnet6.Nodes) == 0 {
			return ErrorSubnetNotInNodes
		}

		subnet6SubnetId = subnet6.SubnetId
		if hasAddressFilter {
			reservations, reclaimedSubnetLeases, err = getReservation6sAndReclaimedSubnetLease6sWithIp(tx, subnet6, ip)
		} else {
			reservations, reclaimedSubnetLeases, err = getReservation6sAndReclaimedSubnetLease6s(tx, subnet.GetID())
		}

		return err
	}); err != nil {
		if err == ErrorIpNotBelongToSubnet || err == ErrorSubnetNotInNodes {
			return nil, nil
		} else {
			return nil, err
		}
	}

	if hasAddressFilter {
		return getSubnetLease6sWithIp(subnet6SubnetId, ip, reservations, reclaimedSubnetLeases)
	} else {
		return getSubnetLease6s(subnet6SubnetId, reservations, reclaimedSubnetLeases)
	}
}

func getReservation6sAndReclaimedSubnetLease6sWithIp(tx restdb.Transaction, subnet6 *resource.Subnet6, ip string) ([]*resource.Reservation6, []*resource.SubnetLease6, error) {
	if !subnet6.Ipnet.Contains(net.ParseIP(ip)) {
		return nil, nil, ErrorIpNotBelongToSubnet
	}

	var reservations []*resource.Reservation6
	if err := tx.FillEx(&reservations,
		"select * from gr_reservation6 where subnet6 = $1 and $2::text = any(ip_addresses)",
		subnet6.GetID(), ip); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	var subnetLeases []*resource.SubnetLease6
	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnAddress: ip,
		resource.SqlColumnSubnet6: subnet6.GetID()},
		&subnetLeases); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkLease), pg.Error(err).Error())
	}

	return reservations, subnetLeases, nil
}

func getReservation6sAndReclaimedSubnetLease6s(tx restdb.Transaction, subnetId string) ([]*resource.Reservation6, []*resource.SubnetLease6, error) {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&reservations); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	var subnetLeases []*resource.SubnetLease6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&subnetLeases); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameLease), pg.Error(err).Error())
	}

	return reservations, subnetLeases, nil
}

func getSubnetLease6sWithIp(subnetId uint64, ip string, reservations []*resource.Reservation6, reclaimedSubnetLeases []*resource.SubnetLease6) ([]*resource.SubnetLease6, error) {
	lease6, err := GetSubnetLease6WithoutReclaimed(subnetId, ip, reclaimedSubnetLeases)
	if err != nil {
		log.Debugf("get subnet6 %d lease6s failed: %s", subnetId, err.Error())
		return nil, nil
	} else if lease6 == nil {
		return nil, nil
	}

	if lease6.AllocateMode != pbdhcpagent.LeaseAllocateMode_DYNAMIC.String() {
		return []*resource.SubnetLease6{lease6}, nil
	}

	leasePrefix := prefixFromAddressAndPrefixLen(lease6.Address, lease6.PrefixLen)
	for _, reservation := range reservations {
		for _, ipaddress := range reservation.IpAddresses {
			if ipaddress == lease6.Address && subnetLease6AllocateToReservation6(reservation, lease6) {
				lease6.AllocateMode = pbdhcpagent.LeaseAllocateMode_RESERVATION.String()
				break
			}
		}

		for _, prefix := range reservation.Prefixes {
			if prefix == leasePrefix && subnetLease6AllocateToReservation6(reservation, lease6) {
				lease6.AllocateMode = pbdhcpagent.LeaseAllocateMode_RESERVATION.String()
				break
			}
		}

		if lease6.AllocateMode == pbdhcpagent.LeaseAllocateMode_RESERVATION.String() {
			break
		}
	}

	return []*resource.SubnetLease6{lease6}, nil
}

func GetSubnetLease6WithoutReclaimed(subnetId uint64, ip string, reclaimedSubnetLeases []*resource.SubnetLease6) (*resource.SubnetLease6, error) {
	var resp *pbdhcpagent.GetLease6Response
	if err := transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) (err error) {
		resp, err = client.GetSubnet6Lease(ctx, &pbdhcpagent.GetSubnet6LeaseRequest{Id: subnetId, Address: ip})
		return err
	}); err != nil {
		return nil, errorno.ErrNetworkError(errorno.ErrNameLease, formatError(err))
	}

	subnetLease6 := SubnetLease6FromPbLease6(resp.GetLease())
	for _, reclaimSubnetLease6 := range reclaimedSubnetLeases {
		if reclaimSubnetLease6.Equal(subnetLease6) {
			return nil, nil
		}
	}

	return subnetLease6, nil
}

func SubnetLease6FromPbLease6(lease *pbdhcpagent.DHCPLease6) *resource.SubnetLease6 {
	hwAddress, _ := util.NormalizeMac(lease.GetHwAddress())
	lease6 := &resource.SubnetLease6{
		Subnet6:               strconv.FormatUint(lease.GetSubnetId(), 10),
		Address:               lease.GetAddress(),
		Duid:                  lease.GetDuid(),
		HwAddress:             hwAddress,
		HwAddressType:         lease.GetHwAddressType(),
		HwAddressSource:       lease.GetHwAddressSource().String(),
		HwAddressOrganization: lease.GetHwAddressOrganization(),
		FqdnFwd:               lease.GetFqdnFwd(),
		FqdnRev:               lease.GetFqdnRev(),
		Hostname:              lease.GetHostname(),
		Iaid:                  lease.GetIaid(),
		LeaseState:            lease.GetLeaseState().String(),
		LeaseType:             lease.GetLeaseType(),
		PrefixLen:             lease.GetPrefixLen(),
		RequestType:           lease.GetRequestType(),
		RequestTime:           lease.GetRequestTime(),
		ValidLifetime:         lease.GetValidLifetime(),
		PreferredLifetime:     lease.GetPreferredLifetime(),
		ExpirationTime:        lease.GetExpirationTime(),
		Fingerprint:           lease.GetFingerprint(),
		VendorId:              lease.GetVendorId(),
		OperatingSystem:       lease.GetOperatingSystem(),
		ClientType:            lease.GetClientType(),
		RequestSourceAddr:     lease.GetRequestSourceAddr(),
		AddressCode:           lease.GetAddressCode(),
		AddressCodeBegin:      lease.GetAddressCodeBegin(),
		AddressCodeEnd:        lease.GetAddressCodeEnd(),
		Subnet:                lease.GetSubnet(),
		AllocateMode:          lease.GetAllocateMode().String(),
	}

	lease6.SetID(lease.GetAddress())
	return lease6
}

func subnetLease6AllocateToReservation6(reservation *resource.Reservation6, lease6 *resource.SubnetLease6) bool {
	return (reservation.HwAddress != "" && strings.EqualFold(reservation.HwAddress, lease6.HwAddress)) ||
		(reservation.Hostname != "" && reservation.Hostname == lease6.Hostname) ||
		(reservation.Duid != "" && reservation.Duid == lease6.Duid)
}

func getSubnetLease6s(subnetId uint64, reservations []*resource.Reservation6, reclaimedSubnetLeases []*resource.SubnetLease6) ([]*resource.SubnetLease6, error) {
	leases, reclaimleasesForRetain, err := getSubnetLease6sWithoutReclaimed(subnetId, reclaimedSubnetLeases,
		reservationMapFromReservation6s(reservations))
	if err != nil {
		return nil, err
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Exec("delete from gr_subnet_lease6 where id not in ('" +
			strings.Join(reclaimleasesForRetain, "','") + "')")
		return err
	}); err != nil {
		log.Warnf("delete reclaim lease6s failed: %s", pg.Error(err).Error())
	}

	return leases, nil
}

func getSubnetLease6sWithoutReclaimed(subnetId uint64, reclaimedSubnetLeases []*resource.SubnetLease6, reservationMap map[string]*resource.Reservation6, ips ...string) ([]*resource.SubnetLease6, []string, error) {
	var resp *pbdhcpagent.GetLeases6Response
	if err := transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) (err error) {
		if len(ips) != 0 {
			resp, err = client.GetSubnet6LeasesWithIps(ctx, ipsToPbGetSubnet6LeasesWithIpsRequest(subnetId, ips))
		} else {
			resp, err = client.GetSubnet6Leases(ctx, &pbdhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
		}
		return err
	}); err != nil {
		log.Debugf("get subnet6 %d lease6s failed: %s", subnetId, err.Error())
		return nil, nil, nil
	}

	reclaimedAddrAndLeases := make(map[string]*resource.SubnetLease6)
	for _, subnetLease := range reclaimedSubnetLeases {
		reclaimedAddrAndLeases[subnetLease.Address] = subnetLease
	}

	var leases []*resource.SubnetLease6
	var reclaimleasesForRetain []string
	for _, lease := range resp.GetLeases() {
		lease6 := subnetLease6FromPbLease6AndReservations(lease, reservationMap)
		if reclaimedLease, ok := reclaimedAddrAndLeases[lease6.Address]; ok && reclaimedLease.Equal(lease6) {
			reclaimleasesForRetain = append(reclaimleasesForRetain, reclaimedLease.GetID())
		} else {
			leases = append(leases, lease6)
		}
	}

	return leases, reclaimleasesForRetain, nil
}

func ipsToPbGetSubnet6LeasesWithIpsRequest(subnetId uint64, ips []string) *pbdhcpagent.GetSubnet6LeasesWithIpsRequest {
	reqs := make([]*pbdhcpagent.GetSubnet6LeaseRequest, 0, len(ips))
	for _, ip := range ips {
		reqs = append(reqs, &pbdhcpagent.GetSubnet6LeaseRequest{
			Id:      subnetId,
			Address: ip,
		})
	}

	return &pbdhcpagent.GetSubnet6LeasesWithIpsRequest{Addresses: reqs}
}

func subnetLease6FromPbLease6AndReservations(lease *pbdhcpagent.DHCPLease6, reservationMap map[string]*resource.Reservation6) *resource.SubnetLease6 {
	subnetLease6 := SubnetLease6FromPbLease6(lease)
	if subnetLease6.AllocateMode == pbdhcpagent.LeaseAllocateMode_DYNAMIC.String() {
		if reservation, ok := reservationMap[prefixFromAddressAndPrefixLen(subnetLease6.Address,
			subnetLease6.PrefixLen)]; ok && subnetLease6AllocateToReservation6(reservation, subnetLease6) {
			subnetLease6.AllocateMode = pbdhcpagent.LeaseAllocateMode_RESERVATION.String()
		}
	}

	return subnetLease6
}

func (l *SubnetLease6Service) Delete(subnetId, leaseId string) error {
	_, err := gohelperip.ParseIPv6(leaseId)
	if err != nil {
		return errorno.ErrInvalidAddress(leaseId)
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return l.batchDeleteLease6s(tx, subnetId, leaseId)
	})
}

func (l *SubnetLease6Service) batchDeleteLease6s(tx restdb.Transaction, subnetId string, leaseIds ...string) error {
	subnet6, err := getSubnet6FromDB(tx, subnetId)
	if err != nil {
		return err
	}

	reclaimedSubnetLeases, err := getReclaimedSubnetLease6sWithIps(tx, subnetId, leaseIds)
	if err != nil {
		return err
	}

	lease6s, _, err := getSubnetLease6sWithoutReclaimed(subnet6.SubnetId, reclaimedSubnetLeases, nil, leaseIds...)
	if err != nil {
		return err
	}

	if len(lease6s) == 0 {
		return nil
	}

	reclaimLease6sSql, leaseTypeAndAddrs := lease6sToInsertSubnetLease6Sql(subnetId, lease6s)
	if _, err := tx.Exec(reclaimLease6sSql); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameLease), pg.Error(err).Error())
	}

	for leaseType, addrs := range leaseTypeAndAddrs {
		return transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
			_, err = client.DeleteLeases6(ctx,
				&pbdhcpagent.DeleteLeases6Request{SubnetId: subnet6.SubnetId,
					LeaseType: leaseType, Addresses: addrs})
			if err != nil {
				err = errorno.ErrNetworkError(errorno.ErrNameLease, formatError(err))
			}
			return err
		})
	}

	return nil
}

func getReclaimedSubnetLease6sWithIps(tx restdb.Transaction, subnetId string, ips []string) ([]*resource.SubnetLease6, error) {
	var reclaimedSubnetLeases []*resource.SubnetLease6
	if err := tx.FillEx(&reclaimedSubnetLeases, "select * from gr_subnet_lease6 where id in ('"+
		strings.Join(ips, "','")+"') and subnet6 = $1", subnetId); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameLease), pg.Error(err).Error())
	}

	return reclaimedSubnetLeases, nil
}

func lease6sToInsertSubnetLease6Sql(subnetId string, lease6s []*resource.SubnetLease6) (string, map[string][]string) {
	var buf bytes.Buffer
	leaseTypeAndAddrs := make(map[string][]string, len(lease6s))
	buf.WriteString("INSERT INTO gr_subnet_lease6 VALUES ")
	for _, lease6 := range lease6s {
		buf.WriteString(subnetLease6ToInsertDBSqlString(subnetId, lease6))
		leaseTypeAndAddrs[lease6.LeaseType] = append(leaseTypeAndAddrs[lease6.LeaseType], lease6.Address)
	}

	return strings.TrimSuffix(buf.String(), ",") + ";", leaseTypeAndAddrs
}

func (l *SubnetLease6Service) BatchDeleteLease6s(subnetId string, leaseIds []string) error {
	if len(leaseIds) == 0 {
		return nil
	}

	for _, leaseId := range leaseIds {
		_, err := gohelperip.ParseIPv6(leaseId)
		if err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameIp, leaseId)
		}
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return l.batchDeleteLease6s(tx, subnetId, leaseIds...)
	})
}

func GetSubnets6LeasesWithMacs(hwAddresses []string) ([]*resource.SubnetLease6, error) {
	var err error
	var resp *pbdhcpagent.GetLeases6Response
	if err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnets6LeasesWithMacs(ctx,
			&pbdhcpagent.GetSubnets6LeasesWithMacsRequest{HwAddresses: util.ToLower(hwAddresses)})
		return err
	}); err != nil {
		return nil, errorno.ErrNetworkError(errorno.ErrNameLease, formatError(err))
	}

	subnetIds := make([]string, 0, len(resp.GetLeases()))
	uniqueSubnetIds := make(map[uint64]struct{}, len(resp.GetLeases()))
	hasDynamicLease := false
	for _, lease := range resp.GetLeases() {
		if _, ok := uniqueSubnetIds[lease.GetSubnetId()]; !ok {
			uniqueSubnetIds[lease.GetSubnetId()] = struct{}{}
			subnetIds = append(subnetIds, strconv.FormatUint(lease.GetSubnetId(), 10))
		}

		if !hasDynamicLease && lease.GetAllocateMode() == pbdhcpagent.LeaseAllocateMode_DYNAMIC {
			hasDynamicLease = true
		}
	}

	var reservationMap map[string]*resource.Reservation6
	if hasDynamicLease {
		if reservationMap, err = getAddrAndReservation6MapWithSubnetIds(subnetIds); err != nil {
			return nil, err
		}
	}

	leases := make([]*resource.SubnetLease6, 0, len(resp.GetLeases()))
	for _, lease := range resp.GetLeases() {
		leases = append(leases, subnetLease6FromPbLease6AndReservations(lease, reservationMap))
	}

	return leases, nil
}

func getAddrAndReservation6MapWithSubnetIds(subnetIds []string) (map[string]*resource.Reservation6, error) {
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&reservations, "select * from gr_reservation6 where subnet6 = any($1::text[])", subnetIds)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), err.Error())
	}

	return reservationMapFromReservation6s(reservations), nil
}
