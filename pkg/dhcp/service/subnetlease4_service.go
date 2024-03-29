package service

import (
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

type SubnetLease4Service struct{}

func NewSubnetLease4Service() *SubnetLease4Service {
	return &SubnetLease4Service{}
}

func (l *SubnetLease4Service) List(subnet *resource.Subnet4, ip string) ([]*resource.SubnetLease4, error) {
	return ListSubnetLease4(subnet, ip)
}

func (l *SubnetLease4Service) ActionListToReservation(subnet *resource.Subnet4, input *resource.ConvToReservationInput) (
	*resource.ConvToReservationInput, error) {
	if len(input.Addresses) == 0 {
		return &resource.ConvToReservationInput{Data: []resource.ConvToReservationItem{}}, nil
	}

	leases, err := ListSubnetLease4(subnet, "")
	if err != nil {
		return nil, err
	}

	reservationLeases := make([]*resource.SubnetLease4, 0, len(leases))
	for _, lease := range leases {
		if lease.AddressType == resource.AddressTypeDynamic && slice.SliceIndex(input.Addresses, lease.Address) >= 0 {
			reservationLeases = append(reservationLeases, lease)
		}
	}

	switch input.ReservationType {
	case resource.ReservationTypeMac:
		return l.listToReservationWithMac(reservationLeases)
	case resource.ReservationTypeHostname:
		return l.listToReservationWithHostname(reservationLeases)
	default:
		return nil, fmt.Errorf("unsupported type %q", input.ReservationType)
	}
}

func (l *SubnetLease4Service) filterAbleToReservation(subnetId string, leases []*resource.SubnetLease4,
	items []resource.ConvToReservationItem) ([]*resource.SubnetLease4, []string, error) {
	addresses := make([]string, len(items))
	for i, item := range items {
		addresses[i] = item.Address
	}

	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&reservations, `select * from gr_reservation4 where subnet4 = $1 and ip_address = any($2::text[])`,
			subnetId, addresses)
	}); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	reservationLeases := make([]*resource.SubnetLease4, 0, len(items))
	hwAddresses := make([]string, 0, len(items))
outer:
	for _, item := range items {
		for _, lease := range leases {
			if lease.Address != item.Address {
				continue
			}
			if lease.AddressType == resource.AddressTypeDynamic {
				reservationLeases = append(reservationLeases, lease)
			}
			if lease.HwAddress != "" {
				hwAddresses = append(hwAddresses, lease.HwAddress)
			}
			continue outer
		}

		for _, reservation := range reservations {
			if reservation.IpAddress != item.Address {
				continue
			} else if reservation.HwAddress != "" {
				hwAddresses = append(hwAddresses, reservation.HwAddress)
			}
			continue outer
		}

		return nil, nil, errorno.ErrNotFound(errorno.ErrNameLease, item.Address)
	}

	return reservationLeases, hwAddresses, nil
}

func (l *SubnetLease4Service) listToReservationWithMac(leases []*resource.SubnetLease4) (
	*resource.ConvToReservationInput, error) {
	reservationLeases := make([]*resource.SubnetLease4, 0, len(leases))
	hwAddresses := make([]string, 0, len(leases))
	seen := make(map[string]bool, len(leases))
	for _, lease := range leases {
		if lease.HwAddress != "" {
			reservationLeases = append(reservationLeases, lease)
			if !seen[lease.HwAddress] {
				seen[lease.HwAddress] = true
				hwAddresses = append(hwAddresses, lease.HwAddress)
			}
		}
	}

	lease6s, err := GetSubnets6LeasesWithMacs(hwAddresses)
	if err != nil {
		log.Warnf("get leases of subnets with macs failed: %s", err.Error())
	}

	result := make([]resource.ConvToReservationItem, 0, len(reservationLeases))
	for _, lease := range reservationLeases {
		dualStack := make([]string, 0, len(lease6s))
		for _, lease6 := range lease6s {
			if lease6.HwAddress == lease.HwAddress &&
				lease6.AddressType == resource.AddressTypeDynamic &&
				!lease6.BelongEui64Subnet && !lease6.BelongAddrCodeSubnet {
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

func (l *SubnetLease4Service) listToReservationWithHostname(leases []*resource.SubnetLease4) (
	*resource.ConvToReservationInput, error) {
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

	leases, err := ListSubnetLease4(subnet, "")
	if err != nil {
		return err
	}

	reservations, hwAddresses, ipv6MacMap, err := l.getReservationFromLease(leases, input)
	if err != nil {
		return err
	}

	v4ReservationMap := map[string][]*resource.Reservation4{subnet.GetID(): reservations}

	if !input.BothV4V6 || input.ReservationType != resource.ReservationTypeMac {
		return createReservationsBySubnet(v4ReservationMap, nil)
	}

	lease6s, err := GetSubnets6LeasesWithMacs(hwAddresses)
	if err != nil {
		return err
	}

	seenMac := make(map[string]map[string][]string, len(lease6s))
	for _, lease6 := range lease6s {
		expectMac, exist := ipv6MacMap[lease6.Address]
		if !exist {
			continue
		} else if lease6.HwAddress != expectMac {
			return errorno.ErrChanged(errorno.ErrNameMac, lease6.Address, expectMac, lease6.HwAddress)
		}

		if lease6.BelongEui64Subnet || lease6.BelongAddrCodeSubnet {
			return errorno.ErrAddressWithEui64OrCode(lease6.Address)
		}

		delete(ipv6MacMap, lease6.Address)
		if _, ok := seenMac[lease6.HwAddress]; !ok {
			seenMac[lease6.HwAddress] = map[string][]string{}
		}
		seenMac[lease6.HwAddress][lease6.Subnet6] = append(seenMac[lease6.HwAddress][lease6.Subnet6], lease6.Address)
	}

	for ipv6, mac := range ipv6MacMap {
		return errorno.ErrNoResourceWith(errorno.ErrNameLease, errorno.ErrNameMac, ipv6, mac)
	}

	v6ReservationMap := make(map[string][]*resource.Reservation6, len(lease6s))
	for hwAddress, subnetMap := range seenMac {
		for subnetId, addresses := range subnetMap {
			v6ReservationMap[subnetId] = append(v6ReservationMap[subnetId], &resource.Reservation6{
				IpAddresses: addresses,
				HwAddress:   hwAddress,
			})
		}
	}

	return createReservationsBySubnet(v4ReservationMap, v6ReservationMap)
}

func (l *SubnetLease4Service) getReservationFromLease(leases []*resource.SubnetLease4, input *resource.ConvToReservationInput) (
	[]*resource.Reservation4, []string, map[string]string, error) {
	ipLeaseMap := make(map[string]*resource.SubnetLease4, len(leases))
	for _, item := range leases {
		ipLeaseMap[item.Address] = item
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
			hwAddress = item.HwAddress
			if hwAddress == "" {
				return nil, nil, nil, errorno.ErrNotFound(errorno.ErrNameMac, item.Address)
			} else if hwAddress != lease.HwAddress {
				return nil, nil, nil, errorno.ErrChanged(errorno.ErrNameMac, item.Address, hwAddress, lease.HwAddress)
			}
			hwAddresses = append(hwAddresses, hwAddress)
		case resource.ReservationTypeHostname:
			hostname = item.Hostname
			if hostname == "" {
				return nil, nil, nil, errorno.ErrNotFound(errorno.ErrNameHostname, item.Address)
			} else if hostname != lease.Hostname {
				return nil, nil, nil, errorno.ErrChanged(errorno.ErrNameHostname, item.Address, hostname, lease.Hostname)
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

func ListSubnetLease4(subnet *resource.Subnet4, ip string) ([]*resource.SubnetLease4, error) {
	hasAddressFilter := ip != ""
	if hasAddressFilter {
		if _, err := gohelperip.ParseIPv4(ip); err != nil {
			return nil, nil
		}
	}

	var subnet4SubnetId uint64
	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, subnet.GetID())
		if err != nil {
			return err
		} else if len(subnet4.Nodes) == 0 {
			return ErrorSubnetNotInNodes
		}

		subnet4SubnetId = subnet4.SubnetId
		if hasAddressFilter {
			reservations, subnetLeases, err = getReservation4sAndSubnetLease4sWithIp(
				tx, subnet4, ip)
		} else {
			reservations, subnetLeases, err = getReservation4sAndSubnetLease4s(
				tx, subnet.GetID())
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
		return getSubnetLease4sWithIp(subnet4SubnetId, ip, reservations, subnetLeases)
	} else {
		return getSubnetLease4s(subnet4SubnetId, reservations, subnetLeases)
	}
}

func getReservation4sAndSubnetLease4sWithIp(tx restdb.Transaction, subnet4 *resource.Subnet4, ip string) ([]*resource.Reservation4, []*resource.SubnetLease4, error) {
	if !subnet4.Ipnet.Contains(net.ParseIP(ip)) {
		return nil, nil, ErrorIpNotBelongToSubnet
	}

	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnIpAddress: ip,
		resource.SqlColumnSubnet4:   subnet4.GetID()},
		&reservations); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnAddress: ip,
		resource.SqlColumnSubnet4: subnet4.GetID()},
		&subnetLeases); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameLease), pg.Error(err).Error())
	}

	return reservations, subnetLeases, nil
}

func getReservation4sAndSubnetLease4s(tx restdb.Transaction, subnetId string) ([]*resource.Reservation4, []*resource.SubnetLease4, error) {
	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnetId},
		&reservations); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnetId},
		&subnetLeases); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameLease), pg.Error(err).Error())
	}

	return reservations, subnetLeases, nil
}

func getSubnetLease4sWithIp(subnetId uint64, ip string, reservations []*resource.Reservation4,
	subnetLeases []*resource.SubnetLease4) ([]*resource.SubnetLease4, error) {
	lease4, err := GetSubnetLease4WithoutReclaimed(subnetId, ip,
		subnetLeases)
	if err != nil {
		log.Debugf("get subnet4 %d lease4s failed: %s", subnetId, err.Error())
		return nil, nil
	} else if lease4 == nil {
		return nil, nil
	}

	for _, reservation := range reservations {
		if reservation.IpAddress == lease4.Address &&
			(reservation.HwAddress != "" && strings.EqualFold(reservation.HwAddress, lease4.HwAddress) ||
				reservation.Hostname != "" && reservation.Hostname == lease4.Hostname) {
			lease4.AddressType = resource.AddressTypeReservation
			break
		}
	}

	return []*resource.SubnetLease4{lease4}, nil
}

func GetSubnetLease4WithoutReclaimed(subnetId uint64, ip string, subnetLeases []*resource.SubnetLease4) (*resource.SubnetLease4, error) {
	var err error
	var resp *pbdhcpagent.GetLease4Response
	if err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet4Lease(ctx,
			&pbdhcpagent.GetSubnet4LeaseRequest{Id: subnetId, Address: ip})
		return err
	}); err != nil {
		errMsg := err.Error()
		if s, ok := status.FromError(err); ok {
			errMsg = s.Message()
		}
		return nil, errorno.ErrNetworkError(errorno.ErrNameLease, errMsg)
	}

	subnetLease4 := SubnetLease4FromPbLease4(resp.GetLease())
	for _, reclaimSubnetLease4 := range subnetLeases {
		if reclaimSubnetLease4.Equal(subnetLease4) {
			return nil, nil
		}
	}

	return subnetLease4, nil
}

func SubnetLease4FromPbLease4(lease *pbdhcpagent.DHCPLease4) *resource.SubnetLease4 {
	hwAddress, _ := util.NormalizeMac(lease.GetHwAddress())
	lease4 := &resource.SubnetLease4{
		Subnet4:               strconv.FormatUint(lease.GetSubnetId(), 10),
		Address:               lease.GetAddress(),
		AddressType:           resource.AddressTypeDynamic,
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
	}

	lease4.SetID(lease.GetAddress())
	return lease4
}

func getSubnetLease4s(subnetId uint64, reservations []*resource.Reservation4, subnetLeases []*resource.SubnetLease4) ([]*resource.SubnetLease4, error) {
	var err error
	var resp *pbdhcpagent.GetLeases4Response
	if err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet4Leases(ctx,
			&pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
		return err
	}); err != nil {
		log.Debugf("get subnet4 %d lease4s failed: %s", subnetId, err.Error())
		return nil, nil
	}

	reservationMap := reservationMapFromReservation4s(reservations)
	reclaimedSubnetLeases := make(map[string]*resource.SubnetLease4)
	for _, subnetLease := range subnetLeases {
		reclaimedSubnetLeases[subnetLease.Address] = subnetLease
	}

	var leases []*resource.SubnetLease4
	var reclaimleasesForRetain []string
	for _, lease := range resp.GetLeases() {
		lease4 := subnetLease4FromPbLease4AndReservations(lease, reservationMap)
		if reclaimedLease, ok := reclaimedSubnetLeases[lease4.Address]; ok &&
			reclaimedLease.Equal(lease4) {
			reclaimleasesForRetain = append(reclaimleasesForRetain, reclaimedLease.GetID())
			continue
		} else {
			leases = append(leases, lease4)
		}
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

func subnetLease4FromPbLease4AndReservations(lease *pbdhcpagent.DHCPLease4, reservationMap map[string]*resource.Reservation4) *resource.SubnetLease4 {
	subnetLease4 := SubnetLease4FromPbLease4(lease)
	if r4, ok := reservationMap[subnetLease4.Address]; ok &&
		(r4.HwAddress != "" && strings.EqualFold(r4.HwAddress, lease.HwAddress) ||
			r4.Hostname != "" && r4.Hostname == lease.Hostname) {
		subnetLease4.AddressType = resource.AddressTypeReservation
	}
	return subnetLease4
}

func (l *SubnetLease4Service) Delete(subnet *resource.Subnet4, leaseId string) error {
	if _, err := gohelperip.ParseIPv4(leaseId); err != nil {
		return errorno.ErrInvalidAddress(leaseId)
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, subnet.GetID())
		if err != nil {
			return err
		}

		_, subnetLeases, err := getReservation4sAndSubnetLease4sWithIp(
			tx, subnet4, leaseId)
		if err != nil {
			return err
		}

		lease4, err := GetSubnetLease4WithoutReclaimed(subnet4.SubnetId, leaseId,
			subnetLeases)
		if err != nil {
			return err
		} else if lease4 == nil {
			return nil
		}

		lease4.LeaseState = pbdhcpagent.LeaseState_RECLAIMED.String()
		lease4.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(lease4); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameLease), pg.Error(err).Error())
		}

		return transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
			_, err = client.DeleteLease4(ctx,
				&pbdhcpagent.DeleteLease4Request{SubnetId: subnet4.SubnetId, Address: leaseId})
			if err != nil {
				errMsg := err.Error()
				if s, ok := status.FromError(err); ok {
					errMsg = s.Message()
				}
				err = errorno.ErrNetworkError(errorno.ErrNameLease, errMsg)
			}
			return err
		})
	})
}

func (l *SubnetLease4Service) BatchDeleteLease4s(subnetId string, leaseIds []string) error {
	for _, leaseId := range leaseIds {
		if _, err := gohelperip.ParseIPv4(leaseId); err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameIp, leaseId)
		}
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, subnetId)
		if err != nil {
			return err
		}

		for _, leaseId := range leaseIds {
			_, subnetLeases, err := getReservation4sAndSubnetLease4sWithIp(
				tx, subnet4, leaseId)
			if err != nil {
				return err
			}

			lease4, err := GetSubnetLease4WithoutReclaimed(subnet4.SubnetId, leaseId,
				subnetLeases)
			if err != nil {
				return err
			} else if lease4 == nil {
				return nil
			}

			lease4.LeaseState = pbdhcpagent.LeaseState_RECLAIMED.String()
			lease4.Subnet4 = subnetId
			if _, err = tx.Insert(lease4); err != nil {
				return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameLease), pg.Error(err).Error())
			}
		}

		if err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
			_, err = client.DeleteLeases4(ctx,
				&pbdhcpagent.DeleteLeases4Request{SubnetId: subnet4.SubnetId, Addresses: leaseIds})
			return err
		}); err != nil {
			errMsg := err.Error()
			if s, ok := status.FromError(err); ok {
				errMsg = s.Message()
				if strings.Contains(errMsg, ":") {
					msgs := strings.Split(errMsg, ":")
					errMsg = msgs[len(msgs)-1]
				}
			}
			return errorno.ErrDBError(errorno.ErrDBNameDelete, string(errorno.ErrNameLease), errMsg)
		}

		return nil
	})
}

func GetSubnets4LeasesWithMacs(hwAddresses []string) ([]*resource.SubnetLease4, error) {
	var err error
	var resp *pbdhcpagent.GetLeases4Response
	if err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnets4LeasesWithMacs(ctx,
			&pbdhcpagent.GetSubnets4LeasesWithMacsRequest{
				HwAddresses: util.ToLower(hwAddresses),
			})
		return err
	}); err != nil {
		errMsg := err.Error()
		if s, ok := status.FromError(err); ok {
			errMsg = s.Message()
		}
		return nil, errorno.ErrNetworkError(errorno.ErrNameLease, errMsg)
	}

	addresses := make([]string, len(resp.Leases))
	for i, lease := range resp.Leases {
		addresses[i] = lease.Address
	}

	var reservations []*resource.Reservation4
	if err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&reservations, `select * from gr_reservation4 where ip_address = any($1::text[])`, addresses)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), err.Error())
	}

	reservationMap := reservationMapFromReservation4s(reservations)

	leases := make([]*resource.SubnetLease4, len(resp.Leases))
	for i, lease := range resp.Leases {
		leases[i] = subnetLease4FromPbLease4AndReservations(lease, reservationMap)
	}

	return leases, nil
}
