package service

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/log"
	"github.com/linkingthing/cement/slice"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	"google.golang.org/grpc/status"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var (
	ErrorIpNotBelongToSubnet = fmt.Errorf("ip not belongs to subnet")
	ErrorSubnetNotInNodes    = fmt.Errorf("subnet is not in any nodes")
)

const NeedFilterDeclineLeases = true

type SubnetLease4Service struct{}

func NewSubnetLease4Service() *SubnetLease4Service {
	return &SubnetLease4Service{}
}

func (l *SubnetLease4Service) List(subnet *resource.Subnet4, ip string) ([]*resource.SubnetLease4, error) {
	return ListSubnetLease4(subnet, ip)
}

func (l *SubnetLease4Service) ActionListToReservation(subnet *resource.Subnet4, input *resource.ConvToReservationInput) (*resource.ConvToReservationInput, error) {
	if len(input.Addresses) == 0 {
		return &resource.ConvToReservationInput{Data: []resource.ConvToReservationItem{}}, nil
	}

	leases, err := ListSubnetLease4(subnet, "", NeedFilterDeclineLeases)
	if err != nil {
		return nil, err
	}

	toReservationLeases := make([]*resource.SubnetLease4, 0, len(leases))
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

func (l *SubnetLease4Service) listToReservationWithMac(leases []*resource.SubnetLease4) (*resource.ConvToReservationInput, error) {
	reservationLeases := make([]*resource.SubnetLease4, 0, len(leases))
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

	lease6s, err := GetSubnets6LeasesWithMacs(hwAddresses, NeedFilterDeclineLeases)
	if err != nil {
		log.Warnf("get leases of subnets with macs failed: %s", err.Error())
	}

	result := make([]resource.ConvToReservationItem, 0, len(reservationLeases))
	for _, lease := range reservationLeases {
		dualStack := make([]string, 0, len(lease6s))
		for _, lease6 := range lease6s {
			if lease6.HwAddress == lease.HwAddress &&
				lease6.AllocateMode == pbdhcpagent.LeaseAllocateMode_DYNAMIC.String() {
				dualStack = append(dualStack, lease6.Address)
			}
		}

		result = append(result, resource.ConvToReservationItem{
			Address:    lease.Address,
			DualStacks: dualStack,
			HwAddress:  lease.HwAddress,
			Hostname:   lease.Hostname,
		})
	}

	return &resource.ConvToReservationInput{Data: result}, nil
}

func (l *SubnetLease4Service) listToReservationWithHostname(leases []*resource.SubnetLease4) (*resource.ConvToReservationInput, error) {
	result := make([]resource.ConvToReservationItem, 0, len(leases))
	for _, lease := range leases {
		if lease.Hostname != "" {
			result = append(result, resource.ConvToReservationItem{
				Address:   lease.Address,
				HwAddress: lease.HwAddress,
				Hostname:  lease.Hostname,
			})
		}
	}

	return &resource.ConvToReservationInput{Data: result}, nil
}

func (l *SubnetLease4Service) ActionDynamicToReservation(subnet *resource.Subnet4, input *resource.ConvToReservationInput) error {
	if len(input.Data) == 0 {
		return nil
	}

	leases, err := ListSubnetLease4(subnet, "", NeedFilterDeclineLeases)
	if err != nil {
		return err
	}

	reservations, hwAddresses, ipv6MacMap, err := l.getReservationFromLease(leases, input)
	if err != nil {
		return err
	}

	v4ReservationMap := map[string][]*resource.Reservation4{subnet.GetID(): reservations}
	if !input.BothV4V6 || input.ReservationType != resource.ReservationTypeMac {
		return createReservationsFromDynamicLeases(v4ReservationMap, nil)
	}

	lease6s, err := GetSubnets6LeasesWithMacs(hwAddresses, NeedFilterDeclineLeases)
	if err != nil {
		return err
	}

	macAndSubnetLeases := make(map[string]map[string][]string, len(lease6s))
	for _, lease6 := range lease6s {
		expectMac, exist := ipv6MacMap[lease6.Address]
		if !exist {
			continue
		} else if lease6.HwAddress != expectMac {
			return errorno.ErrChanged(errorno.ErrNameMac, lease6.Address, expectMac, lease6.HwAddress)
		}

		if lease6.AllocateMode != pbdhcpagent.LeaseAllocateMode_DYNAMIC.String() &&
			lease6.AllocateMode != pbdhcpagent.LeaseAllocateMode_RESERVATION.String() {
			return errorno.ErrAddressAutoGenerated(lease6.Address)
		}

		delete(ipv6MacMap, lease6.Address)
		if lease6.AllocateMode == pbdhcpagent.LeaseAllocateMode_RESERVATION.String() {
			continue
		}

		if _, ok := macAndSubnetLeases[lease6.HwAddress]; !ok {
			macAndSubnetLeases[lease6.HwAddress] = make(map[string][]string)
		}

		macAndSubnetLeases[lease6.HwAddress][lease6.Subnet6] = append(
			macAndSubnetLeases[lease6.HwAddress][lease6.Subnet6], lease6.Address)
	}

	for ipv6, mac := range ipv6MacMap {
		return errorno.ErrNoResourceWith(errorno.ErrNameLease, errorno.ErrNameMac, ipv6, mac)
	}

	v6ReservationMap := make(map[string][]*resource.Reservation6, len(lease6s))
	for hwAddress, subnetMap := range macAndSubnetLeases {
		for subnetId, addresses := range subnetMap {
			v6ReservationMap[subnetId] = append(v6ReservationMap[subnetId], &resource.Reservation6{
				IpAddresses: addresses,
				HwAddress:   hwAddress,
			})
		}
	}

	return createReservationsFromDynamicLeases(v4ReservationMap, v6ReservationMap)
}

func (l *SubnetLease4Service) getReservationFromLease(leases []*resource.SubnetLease4, input *resource.ConvToReservationInput) ([]*resource.Reservation4, []string, map[string]string, error) {
	ipLeaseMap := make(map[string]*resource.SubnetLease4, len(leases))
	for _, lease := range leases {
		ipLeaseMap[lease.Address] = lease
	}

	reservations := make([]*resource.Reservation4, 0, len(input.Data))
	hwAddresses := make([]string, 0, len(input.Data))
	ipv6MacMap := make(map[string]string, len(input.Data))
	for _, item := range input.Data {
		lease := ipLeaseMap[item.Address]
		if lease == nil {
			return nil, nil, nil, errorno.ErrNotFound(errorno.ErrNameLease, item.Address)
		}

		var hwAddress, hostname string
		switch input.ReservationType {
		case resource.ReservationTypeMac:
			if hwAddress = item.HwAddress; hwAddress == "" {
				return nil, nil, nil, errorno.ErrNotFound(errorno.ErrNameMac, item.Address)
			} else if hwAddress != lease.HwAddress {
				return nil, nil, nil, errorno.ErrChanged(errorno.ErrNameMac, item.Address, item.HwAddress, lease.HwAddress)
			} else {
				hwAddresses = append(hwAddresses, hwAddress)
			}
		case resource.ReservationTypeHostname:
			if hostname = item.Hostname; hostname == "" {
				return nil, nil, nil, errorno.ErrNotFound(errorno.ErrNameHostname, item.Address)
			} else if hostname != lease.Hostname {
				return nil, nil, nil, errorno.ErrChanged(errorno.ErrNameHostname, item.Address, item.Hostname, lease.Hostname)
			}
		default:
			return nil, nil, nil, fmt.Errorf("unsupported type %q", input.ReservationType)
		}

		reservations = append(reservations, &resource.Reservation4{
			IpAddress: item.Address,
			HwAddress: hwAddress,
			Hostname:  hostname,
		})

		for _, ipv6 := range item.DualStacks {
			ipv6MacMap[ipv6] = item.HwAddress
		}
	}

	return reservations, hwAddresses, ipv6MacMap, nil
}

func ListSubnetLease4(subnet *resource.Subnet4, ipstr string, needFilterDeclineLeases ...bool) ([]*resource.SubnetLease4, error) {
	var ips []string
	var ip net.IP
	if ipstr != "" {
		if ip_, err := gohelperip.ParseIPv4(ipstr); err != nil {
			return nil, nil
		} else {
			ips = []string{ipstr}
			ip = ip_
		}
	}

	var subnet4SubnetId uint64
	var reservations []*resource.Reservation4
	var reclaimedSubnetLeases []*resource.SubnetLease4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, subnet.GetID())
		if err != nil {
			return err
		} else if len(subnet4.Nodes) == 0 {
			return ErrorSubnetNotInNodes
		}

		subnet4SubnetId = subnet4.SubnetId
		reservations, reclaimedSubnetLeases, err = getReservation4sAndReclaimedSubnetLease4s(tx, subnet4, ip)
		return err
	}); err != nil {
		if err == ErrorIpNotBelongToSubnet || err == ErrorSubnetNotInNodes {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return getSubnetLease4s(subnet4SubnetId, reservations, reclaimedSubnetLeases, ips, needFilterDeclineLeases...)
}

func getReservation4sAndReclaimedSubnetLease4s(tx restdb.Transaction, subnet4 *resource.Subnet4, ip net.IP) ([]*resource.Reservation4, []*resource.SubnetLease4, error) {
	reservationCondition := map[string]interface{}{resource.SqlColumnSubnet4: subnet4.GetID()}
	subnetLeaseCondition := map[string]interface{}{resource.SqlColumnSubnet4: subnet4.GetID()}
	if ip != nil {
		if !subnet4.Ipnet.Contains(ip) {
			return nil, nil, ErrorIpNotBelongToSubnet
		}

		reservationCondition[resource.SqlColumnIpAddress] = ip.String()
		subnetLeaseCondition[resource.SqlColumnAddress] = ip.String()
	}

	var reservations []*resource.Reservation4
	if err := tx.Fill(reservationCondition, &reservations); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	var subnetLeases []*resource.SubnetLease4
	if err := tx.Fill(subnetLeaseCondition, &subnetLeases); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameLease), pg.Error(err).Error())
	}

	return reservations, subnetLeases, nil
}

func GetSubnetLease4WithoutReclaimed(subnetId uint64, ip string, reclaimedSubnetLeases []*resource.SubnetLease4) (*resource.SubnetLease4, error) {
	var resp *pbdhcpagent.GetLease4Response
	if err := transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) (err error) {
		resp, err = client.GetSubnet4Lease(ctx, &pbdhcpagent.GetSubnet4LeaseRequest{Id: subnetId, Address: ip})
		return err
	}); err != nil {
		return nil, errorno.ErrNetworkError(errorno.ErrNameLease, formatError(err))
	}

	subnetLease4 := SubnetLease4FromPbLease4(resp.GetLease())
	for _, reclaimSubnetLease4 := range reclaimedSubnetLeases {
		if reclaimSubnetLease4.Equal(subnetLease4) {
			return nil, nil
		}
	}

	return subnetLease4, nil
}

func formatError(err error) string {
	errMsg := err.Error()
	if s, ok := status.FromError(err); ok {
		errMsg = s.Message()
	}

	if strings.Contains(errMsg, ":") {
		msgs := strings.Split(errMsg, ":")
		errMsg = msgs[len(msgs)-1]
	}

	return errMsg
}

func SubnetLease4FromPbLease4(lease *pbdhcpagent.DHCPLease4) *resource.SubnetLease4 {
	hwAddress, _ := util.NormalizeMac(lease.GetHwAddress())
	lease4 := &resource.SubnetLease4{
		Subnet4:               strconv.FormatUint(lease.GetSubnetId(), 10),
		Address:               lease.GetAddress(),
		HwAddress:             hwAddress,
		HwAddressOrganization: lease.GetHwAddressOrganization(),
		ClientId:              lease.GetClientId(),
		FqdnFwd:               lease.GetFqdnFwd(),
		FqdnRev:               lease.GetFqdnRev(),
		Hostname:              lease.GetHostname(),
		LeaseState:            lease.GetLeaseState().String(),
		RequestType:           lease.GetRequestType(),
		RequestTime:           lease.GetRequestTime(),
		ValidLifetime:         lease.GetValidLifetime(),
		ExpirationTime:        lease.GetExpirationTime(),
		Fingerprint:           lease.GetFingerprint(),
		VendorId:              lease.GetVendorId(),
		OperatingSystem:       lease.GetOperatingSystem(),
		ClientType:            lease.GetClientType(),
		Subnet:                lease.GetSubnet(),
		AllocateMode:          lease.GetAllocateMode().String(),
	}

	lease4.SetID(lease.GetAddress())
	return lease4
}

func subnetLease4AllocateToReservation4(reservation *resource.Reservation4, lease4 *resource.SubnetLease4) bool {
	return (reservation.HwAddress != "" && strings.EqualFold(reservation.HwAddress, lease4.HwAddress)) ||
		(reservation.Hostname != "" && reservation.Hostname == lease4.Hostname)
}

func getSubnetLease4s(subnetId uint64, reservations []*resource.Reservation4, reclaimedSubnetLeases []*resource.SubnetLease4, ips []string, needFilterDeclineLeases ...bool) ([]*resource.SubnetLease4, error) {
	leases, reclaimleasesForRetain, err := getSubnetLease4sWithoutReclaimed(subnetId, reclaimedSubnetLeases,
		reservationMapFromReservation4s(reservations), ips, needFilterDeclineLeases...)
	if err != nil {
		return nil, err
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Exec("delete from gr_subnet_lease4 where id not in ('" +
			strings.Join(reclaimleasesForRetain, "','") + "')")
		return err
	}); err != nil {
		log.Warnf("delete reclaim lease4s failed: %s", pg.Error(err).Error())
	}

	return leases, nil
}

func getSubnetLease4sWithoutReclaimed(subnetId uint64, reclaimedSubnetLeases []*resource.SubnetLease4, reservationMap map[string]*resource.Reservation4, ips []string, needFilterDeclineLeases ...bool) ([]*resource.SubnetLease4, []string, error) {
	var resp *pbdhcpagent.GetLeases4Response
	if err := transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) (err error) {
		if len(ips) != 0 {
			resp, err = client.GetSubnet4LeasesWithIps(ctx, ipsToPbGetSubnet4LeasesWithIpsRequest(subnetId, ips))
		} else {
			resp, err = client.GetSubnet4Leases(ctx, &pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
		}
		return err
	}); err != nil {
		log.Debugf("get subnet4 %d lease4s failed: %s", subnetId, err.Error())
		return nil, nil, nil
	}

	reclaimedAddrAndLeases := make(map[string]*resource.SubnetLease4)
	for _, subnetLease := range reclaimedSubnetLeases {
		reclaimedAddrAndLeases[subnetLease.Address] = subnetLease
	}

	var leases []*resource.SubnetLease4
	var reclaimleasesForRetain []string
	needFilterDeclineLease := len(needFilterDeclineLeases) != 0 && needFilterDeclineLeases[0]
	for _, lease := range resp.GetLeases() {
		if needFilterDeclineLease && lease.GetLeaseState() != pbdhcpagent.LeaseState_NORMAL {
			continue
		}

		lease4 := subnetLease4FromPbLease4AndReservations(lease, reservationMap)
		if reclaimedLease, ok := reclaimedAddrAndLeases[lease4.Address]; ok && reclaimedLease.Equal(lease4) {
			reclaimleasesForRetain = append(reclaimleasesForRetain, reclaimedLease.GetID())
		} else {
			leases = append(leases, lease4)
		}
	}

	return leases, reclaimleasesForRetain, nil
}

func ipsToPbGetSubnet4LeasesWithIpsRequest(subnetId uint64, ips []string) *pbdhcpagent.GetSubnet4LeasesWithIpsRequest {
	reqs := make([]*pbdhcpagent.GetSubnet4LeaseRequest, 0, len(ips))
	for _, ip := range ips {
		reqs = append(reqs, &pbdhcpagent.GetSubnet4LeaseRequest{
			Id:      subnetId,
			Address: ip,
		})
	}

	return &pbdhcpagent.GetSubnet4LeasesWithIpsRequest{Addresses: reqs}
}

func subnetLease4FromPbLease4AndReservations(lease *pbdhcpagent.DHCPLease4, reservationMap map[string]*resource.Reservation4) *resource.SubnetLease4 {
	subnetLease4 := SubnetLease4FromPbLease4(lease)
	if subnetLease4.AllocateMode == pbdhcpagent.LeaseAllocateMode_DYNAMIC.String() {
		if reservation4, ok := reservationMap[subnetLease4.Address]; ok &&
			subnetLease4AllocateToReservation4(reservation4, subnetLease4) {
			subnetLease4.AllocateMode = pbdhcpagent.LeaseAllocateMode_RESERVATION.String()
		}
	}

	return subnetLease4
}

func (l *SubnetLease4Service) Delete(subnet *resource.Subnet4, leaseId string) error {
	if _, err := gohelperip.ParseIPv4(leaseId); err != nil {
		return errorno.ErrInvalidAddress(leaseId)
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return l.batchDeleteLease4s(tx, subnet.GetID(), leaseId)
	})
}

func (l *SubnetLease4Service) batchDeleteLease4s(tx restdb.Transaction, subnetId string, leaseIds ...string) error {
	subnet4, err := getSubnet4FromDB(tx, subnetId)
	if err != nil {
		return err
	}

	reclaimedSubnetLeases, err := getReclaimedSubnetLease4sWithIps(tx, subnetId, leaseIds)
	if err != nil {
		return err
	}

	lease4s, _, err := getSubnetLease4sWithoutReclaimed(subnet4.SubnetId, reclaimedSubnetLeases, nil, leaseIds)
	if err != nil {
		return err
	}

	if len(lease4s) == 0 {
		return nil
	}

	reclaimLease4sSql, addrs := lease4sToInsertSubnetLease4Sql(subnetId, lease4s)
	if _, err := tx.Exec(reclaimLease4sSql); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameLease), pg.Error(err).Error())
	}

	return transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		_, err := client.DeleteLeases4(ctx, &pbdhcpagent.DeleteLeases4Request{SubnetId: subnet4.SubnetId, Addresses: addrs})
		if err != nil {
			err = errorno.ErrNetworkError(errorno.ErrNameLease, formatError(err))
		}
		return err
	})
}

func getReclaimedSubnetLease4sWithIps(tx restdb.Transaction, subnetId string, ips []string) ([]*resource.SubnetLease4, error) {
	var reclaimedSubnetLeases []*resource.SubnetLease4
	if err := tx.FillEx(&reclaimedSubnetLeases, "select * from gr_subnet_lease4 where id in ('"+
		strings.Join(ips, "','")+"') and subnet4 = $1", subnetId); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameLease), pg.Error(err).Error())
	}

	return reclaimedSubnetLeases, nil
}

func lease4sToInsertSubnetLease4Sql(subnetId string, lease4s []*resource.SubnetLease4) (string, []string) {
	var buf bytes.Buffer
	addrs := make([]string, 0, len(lease4s))
	buf.WriteString("INSERT INTO gr_subnet_lease4 VALUES ")
	for _, lease4 := range lease4s {
		buf.WriteString(subnetLease4ToInsertDBSqlString(subnetId, lease4))
		addrs = append(addrs, lease4.Address)
	}

	return strings.TrimSuffix(buf.String(), ",") + ";", addrs
}

func (l *SubnetLease4Service) BatchDeleteLease4s(subnetId string, leaseIds []string) error {
	if len(leaseIds) == 0 {
		return nil
	}

	for _, leaseId := range leaseIds {
		if _, err := gohelperip.ParseIPv4(leaseId); err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameIp, leaseId)
		}
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return l.batchDeleteLease4s(tx, subnetId, leaseIds...)
	})
}

func GetSubnets4LeasesWithMacs(hwAddresses []string, needFilterDeclineLeases ...bool) ([]*resource.SubnetLease4, error) {
	var err error
	var resp *pbdhcpagent.GetLeases4Response
	if err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnets4LeasesWithMacs(ctx,
			&pbdhcpagent.GetSubnets4LeasesWithMacsRequest{HwAddresses: util.ToLower(hwAddresses)})
		return err
	}); err != nil {
		return nil, errorno.ErrNetworkError(errorno.ErrNameLease, formatError(err))
	}

	addresses := make([]string, 0, len(resp.GetLeases()))
	hasDynamicLease := false
	needFilterDeclineLease := len(needFilterDeclineLeases) != 0 && needFilterDeclineLeases[0]
	normalLeases := make([]*pbdhcpagent.DHCPLease4, 0, len(resp.GetLeases()))
	for _, lease := range resp.GetLeases() {
		if needFilterDeclineLease && lease.GetLeaseState() != pbdhcpagent.LeaseState_NORMAL {
			continue
		}

		normalLeases = append(normalLeases, lease)
		addresses = append(addresses, lease.GetAddress())
		if !hasDynamicLease && lease.GetAllocateMode() == pbdhcpagent.LeaseAllocateMode_DYNAMIC {
			hasDynamicLease = true
		}
	}

	var reservationMap map[string]*resource.Reservation4
	if hasDynamicLease {
		if reservationMap, err = getAddrAndReservationMapWithAddresses(addresses); err != nil {
			return nil, err
		}
	}

	leases := make([]*resource.SubnetLease4, 0, len(normalLeases))
	for _, lease := range normalLeases {
		leases = append(leases, subnetLease4FromPbLease4AndReservations(lease, reservationMap))
	}

	return leases, nil
}

func getAddrAndReservationMapWithAddresses(addresses []string) (map[string]*resource.Reservation4, error) {
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&reservations, "select * from gr_reservation4 where ip_address = any($1::text[])", addresses)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), err.Error())
	}

	return reservationMapFromReservation4s(reservations), nil
}

func (s *SubnetLease4Service) ActionFingerprintStatistics(subnet4 *resource.Subnet4) ([]*resource.FingerprintStatistics, error) {
	var subnetId uint64
	var subnetLeases []*resource.SubnetLease4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet, err := getSubnet4FromDB(tx, subnet4.GetID())
		if err != nil {
			return err
		} else if len(subnet.Nodes) == 0 {
			return ErrorSubnetNotInNodes
		}

		subnetId = subnet.SubnetId
		err = tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnet4.GetID()}, &subnetLeases)
		return err
	}); err != nil {
		if err == ErrorSubnetNotInNodes {
			return nil, nil
		} else {
			return nil, err
		}
	}

	var resp *pbdhcpagent.GetLeases4Response
	if err := transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) (err error) {
		resp, err = client.GetSubnet4Leases(ctx, &pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
		return err
	}); err != nil {
		return nil, nil
	}

	leases := resp.GetLeases()
	if len(leases) == 0 {
		return nil, nil
	}

	reclaimedAddrAndLeases := make(map[string]*resource.SubnetLease4)
	for _, subnetLease := range subnetLeases {
		reclaimedAddrAndLeases[subnetLease.Address] = subnetLease
	}

	clientTypes := make(map[string]uint64, len(leases))
	for _, lease := range leases {
		if lease.GetLeaseState() != pbdhcpagent.LeaseState_NORMAL || lease.GetClientType() == "" {
			continue
		}

		if reclaimedLease, ok := reclaimedAddrAndLeases[lease.GetAddress()]; !ok ||
			reclaimedLease.ExpirationTime != lease.GetExpirationTime() ||
			!strings.EqualFold(reclaimedLease.HwAddress, lease.GetHwAddress()) ||
			reclaimedLease.ClientId != lease.GetClientId() ||
			reclaimedLease.Hostname != lease.GetHostname() {
			leaseCount := clientTypes[lease.GetClientType()]
			clientTypes[lease.GetClientType()] = leaseCount + 1
		}
	}

	var stats resource.FingerprintStatisticses
	for clientType, leaseCount := range clientTypes {
		stats = append(stats, &resource.FingerprintStatistics{
			ClientType: clientType,
			LeaseCount: leaseCount,
		})
	}

	sort.Sort(stats)
	return stats, nil
}
