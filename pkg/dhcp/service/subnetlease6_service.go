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

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
)

type SubnetLease6Service struct{}

func NewSubnetLease6Service() *SubnetLease6Service {
	return &SubnetLease6Service{}
}

func (l *SubnetLease6Service) List(subnet *resource.Subnet6, ip string) ([]*resource.SubnetLease6, error) {
	return ListSubnetLease6(subnet, ip)
}

func (l *SubnetLease6Service) ActionListToReservation(subnet *resource.Subnet6, input *resource.ConvToReservationInput) (
	*resource.ConvToReservationOutput, error) {
	if len(input.Addresses) == 0 {
		return &resource.ConvToReservationOutput{Data: []resource.ConvToReservationItem{}}, nil
	}

	leases, err := ListSubnetLease6(subnet, "")
	if err != nil {
		return nil, err
	}

	leases = l.filterAbleToReservation(leases, input.Addresses)

	switch input.ReservationType {
	case resource.ReservationTypeMac:
		return l.listToReservationWithMac(leases)
	case resource.ReservationTypeHostname:
		return l.listToReservationWithHostname(leases)
	default:
		return nil, fmt.Errorf("unsupported type %q", input.ReservationType)
	}
}

func (l *SubnetLease6Service) filterAbleToReservation(leases []*resource.SubnetLease6, addresses []string) []*resource.SubnetLease6 {
	reservationLeases := make([]*resource.SubnetLease6, 0, len(leases))
	for _, lease := range leases {
		if lease.AddressType == resource.AddressTypeDynamic && slice.SliceIndex(addresses, lease.Address) >= 0 {
			reservationLeases = append(reservationLeases, lease)
		}
	}
	return reservationLeases
}

func (l *SubnetLease6Service) listToReservationWithMac(leases []*resource.SubnetLease6) (
	*resource.ConvToReservationOutput, error) {
	reservationLeases := make([]*resource.SubnetLease6, 0, len(leases))
	hwAddresses := make([]string, 0, len(leases))
	for _, lease := range leases {
		if lease.HwAddress != "" {
			reservationLeases = append(reservationLeases, lease)
			hwAddresses = append(hwAddresses, lease.HwAddress)
		}
	}

	lease4s, err := GetSubnets4LeasesWithMacs(hwAddresses)
	if err != nil {
		return nil, err
	}

	result := make([]resource.ConvToReservationItem, 0, len(reservationLeases))
	for _, lease := range reservationLeases {
		var dualStack []string
		for _, lease4 := range lease4s {
			if lease4.HwAddress == lease.HwAddress {
				dualStack = append(dualStack, lease4.Address)
			}
		}

		result = append(result, resource.ConvToReservationItem{
			Address:    lease.Address,
			DualStacks: dualStack,
			HwAddress:  lease.HwAddress,
			Hostname:   lease.Hostname,
		})
	}

	return &resource.ConvToReservationOutput{Data: result}, nil
}

func (l *SubnetLease6Service) listToReservationWithHostname(leases []*resource.SubnetLease6) (
	*resource.ConvToReservationOutput, error) {
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

	return &resource.ConvToReservationOutput{Data: result}, nil
}

func (l *SubnetLease6Service) ActionDynamicToReservation(subnet *resource.Subnet6, input *resource.ConvToReservationInput) error {
	if len(input.Addresses) == 0 {
		return nil
	}

	leases, err := ListSubnetLease6(subnet, "")
	if err != nil {
		return err
	}

	leases = l.filterAbleToReservation(leases, input.Addresses)
	reservations, err := l.getReservationFromLease(leases, input)
	if err != nil {
		return err
	}

	v6ReservationMap := map[string][]*resource.Reservation6{subnet.GetID(): reservations}

	if !input.BothV4V6 || input.ReservationType != resource.ReservationTypeMac {
		return createReservationsBySubnet(nil, v6ReservationMap)
	}

	hwAddresses := make([]string, len(reservations))
	for i, item := range reservations {
		hwAddresses[i] = item.HwAddress
	}

	lease4s, err := GetSubnets4LeasesWithMacs(hwAddresses)
	if err != nil {
		return err
	}

	v4ReservationMap := make(map[string][]*resource.Reservation4, len(lease4s))
	seenMac := make(map[string]bool, len(lease4s))
	for _, lease4 := range lease4s {
		if lease4.HwAddress == "" || seenMac[lease4.HwAddress] || lease4.AddressType != resource.AddressTypeDynamic {
			continue
		}
		seenMac[lease4.HwAddress] = true
		v4ReservationMap[lease4.Subnet4] = append(v4ReservationMap[lease4.Subnet4], &resource.Reservation4{
			IpAddress: lease4.Address,
			HwAddress: lease4.HwAddress,
		})
	}

	return createReservationsBySubnet(v4ReservationMap, v6ReservationMap)
}

func (l *SubnetLease6Service) getReservationFromLease(leases []*resource.SubnetLease6, input *resource.ConvToReservationInput) (
	[]*resource.Reservation6, error) {
	reservations := make([]*resource.Reservation6, 0, len(leases))
	seen := make(map[string]bool, len(leases))
	for _, lease := range leases {
		var hwAddress, hostname, key string
		switch input.ReservationType {
		case resource.ReservationTypeMac:
			hwAddress = lease.HwAddress
			if hwAddress == "" {
				return nil, fmt.Errorf("%s has no hwAddress", lease.Address)
			}
			key = hwAddress
		case resource.ReservationTypeHostname:
			hostname = lease.Hostname
			if hostname == "" {
				return nil, fmt.Errorf("%s has no hostname", lease.Address)
			}
			key = hostname
		default:
			return nil, fmt.Errorf("unsupported type %q", input.ReservationType)
		}

		if seen[key] {
			continue
		} else if key != "" {
			seen[key] = true
		}

		reservations = append(reservations, &resource.Reservation6{
			IpAddresses: []string{lease.Address},
			HwAddress:   hwAddress,
			Hostname:    hostname,
		})
	}

	return reservations, nil
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
	var subnetLeases []*resource.SubnetLease6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnet.GetID())
		if err != nil {
			return err
		} else if len(subnet6.Nodes) == 0 {
			return ErrorSubnetNotInNodes
		}

		subnet6SubnetId = subnet6.SubnetId
		if hasAddressFilter {
			reservations, subnetLeases, err = getReservation6sAndSubnetLease6sWithIp(
				tx, subnet6, ip)
		} else {
			reservations, subnetLeases, err = getReservation6sAndSubnetLease6s(tx, subnet.GetID())
		}
		return err
	}); err != nil {
		if err == ErrorIpNotBelongToSubnet || err == ErrorSubnetNotInNodes {
			return nil, nil
		} else {
			return nil, fmt.Errorf("get subnet6 %s from db failed: %s", subnet.GetID(), err.Error())
		}
	}

	if hasAddressFilter {
		return getSubnetLease6sWithIp(subnet6SubnetId, ip, reservations, subnetLeases)
	} else {
		return getSubnetLease6s(subnet6SubnetId, reservations, subnetLeases)
	}
}

func getReservation6sAndSubnetLease6sWithIp(tx restdb.Transaction, subnet6 *resource.Subnet6, ip string) ([]*resource.Reservation6, []*resource.SubnetLease6, error) {
	if !subnet6.Ipnet.Contains(net.ParseIP(ip)) {
		return nil, nil, ErrorIpNotBelongToSubnet
	}

	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := tx.FillEx(&reservations,
		"select * from gr_reservation6 where subnet6 = $1 and $2::text = any(ip_addresses)",
		subnet6.GetID(), ip); err != nil {
		return nil, nil, fmt.Errorf("get reservation6 %s failed: %s", ip, pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnAddress: ip,
		resource.SqlColumnSubnet6: subnet6.GetID()},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet6 lease6 %s failed: %s", ip, pg.Error(err).Error())
	}

	return reservations, subnetLeases, nil
}

func getReservation6sAndSubnetLease6s(tx restdb.Transaction, subnetId string) ([]*resource.Reservation6, []*resource.SubnetLease6, error) {
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&reservations); err != nil {
		return nil, nil, fmt.Errorf("get reservation6s failed: %s", pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet6 lease6s failed: %s", pg.Error(err).Error())
	}

	return reservations, subnetLeases, nil
}

func getSubnetLease6sWithIp(subnetId uint64, ip string, reservations []*resource.Reservation6,
	subnetLeases []*resource.SubnetLease6) ([]*resource.SubnetLease6, error) {
	lease6, err := GetSubnetLease6WithoutReclaimed(subnetId, ip,
		subnetLeases)
	if err != nil {
		log.Debugf("get subnet6 %d lease6s failed: %s", subnetId, err.Error())
		return nil, nil
	} else if lease6 == nil {
		return nil, nil
	}

	leasePrefix := prefixFromAddressAndPrefixLen(lease6.Address, lease6.PrefixLen)
	for _, reservation := range reservations {
		for _, ipaddress := range reservation.IpAddresses {
			if ipaddress == lease6.Address {
				lease6.AddressType = resource.AddressTypeReservation
				break
			}
		}

		for _, prefix := range reservation.Prefixes {
			if prefix == leasePrefix {
				lease6.AddressType = resource.AddressTypeReservation
				break
			}
		}

		if lease6.AddressType == resource.AddressTypeReservation {
			break
		}
	}

	lease6.HwAddress = strings.ToUpper(lease6.HwAddress)
	return []*resource.SubnetLease6{lease6}, nil
}

func GetSubnetLease6WithoutReclaimed(subnetId uint64, ip string, subnetLeases []*resource.SubnetLease6) (*resource.SubnetLease6, error) {
	var err error
	var resp *pbdhcpagent.GetLease6Response
	if err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet6Lease(ctx,
			&pbdhcpagent.GetSubnet6LeaseRequest{Id: subnetId, Address: ip})
		return err
	}); err != nil {
		return nil, err
	}

	subnetLease6 := SubnetLease6FromPbLease6(resp.GetLease())
	for _, reclaimSubnetLease6 := range subnetLeases {
		if reclaimSubnetLease6.Equal(subnetLease6) {
			return nil, nil
		}
	}

	return subnetLease6, nil
}

func getSubnetLease6s(subnetId uint64, reservations []*resource.Reservation6, subnetLeases []*resource.SubnetLease6) ([]*resource.SubnetLease6, error) {
	var err error
	var resp *pbdhcpagent.GetLeases6Response
	if err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet6Leases(ctx,
			&pbdhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
		return err
	}); err != nil {
		log.Debugf("get subnet6 %d lease6s failed: %s", subnetId, err.Error())
		return nil, nil
	}

	reservationMap := reservationMapFromReservation6s(reservations)
	reclaimedSubnetLeases := make(map[string]*resource.SubnetLease6)
	for _, subnetLease := range subnetLeases {
		reclaimedSubnetLeases[subnetLease.Address] = subnetLease
	}

	var leases []*resource.SubnetLease6
	var reclaimleasesForRetain []string
	for _, lease := range resp.GetLeases() {
		lease6 := subnetLease6FromPbLease6AndReservations(lease, reservationMap)
		if reclaimedLease, ok := reclaimedSubnetLeases[lease6.Address]; ok &&
			reclaimedLease.Equal(lease6) {
			reclaimleasesForRetain = append(reclaimleasesForRetain, reclaimedLease.GetID())
		} else {
			lease6.HwAddress = strings.ToUpper(lease6.HwAddress)
			leases = append(leases, lease6)
		}
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

func subnetLease6FromPbLease6AndReservations(lease *pbdhcpagent.DHCPLease6, reservationMap map[string]struct{}) *resource.SubnetLease6 {
	subnetLease6 := SubnetLease6FromPbLease6(lease)
	if _, ok := reservationMap[prefixFromAddressAndPrefixLen(subnetLease6.Address,
		subnetLease6.PrefixLen)]; ok {
		subnetLease6.AddressType = resource.AddressTypeReservation
	}
	return subnetLease6
}

func SubnetLease6FromPbLease6(lease *pbdhcpagent.DHCPLease6) *resource.SubnetLease6 {
	lease6 := &resource.SubnetLease6{
		Subnet6:               strconv.FormatUint(lease.GetSubnetId(), 10),
		Address:               lease.GetAddress(),
		AddressType:           resource.AddressTypeDynamic,
		PrefixLen:             lease.GetPrefixLen(),
		Duid:                  lease.GetDuid(),
		Iaid:                  lease.GetIaid(),
		HwAddress:             lease.GetHwAddress(),
		HwAddressType:         lease.GetHwAddressType(),
		HwAddressSource:       lease.GetHwAddressSource().String(),
		HwAddressOrganization: lease.GetHwAddressOrganization(),
		ValidLifetime:         lease.GetValidLifetime(),
		PreferredLifetime:     lease.GetPreferredLifetime(),
		Expire:                TimeFromUinx(lease.GetExpire()),
		LeaseType:             lease.GetLeaseType(),
		Hostname:              lease.GetHostname(),
		Fingerprint:           lease.GetFingerprint(),
		VendorId:              lease.GetVendorId(),
		OperatingSystem:       lease.GetOperatingSystem(),
		ClientType:            lease.GetClientType(),
		LeaseState:            lease.GetLeaseState().String(),
		RequestSourceAddr:     lease.GetRequestSourceAddr(),
		AddressCode:           lease.GetAddressCode(),
		AddressCodeBegin:      lease.GetAddressCodeBegin(),
		AddressCodeEnd:        lease.GetAddressCodeEnd(),
	}

	lease6.SetID(lease.GetAddress())
	return lease6
}

func (l *SubnetLease6Service) Delete(subnetId, leaseId string) error {
	_, err := gohelperip.ParseIPv6(leaseId)
	if err != nil {
		return fmt.Errorf("subnet6 %s lease6 id %s is invalid: %s", subnetId, leaseId, err.Error())
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnetId)
		if err != nil {
			return err
		}

		_, subnetLeases, err := getReservation6sAndSubnetLease6sWithIp(
			tx, subnet6, leaseId)
		if err != nil {
			return err
		}

		lease6, err := GetSubnetLease6WithoutReclaimed(subnet6.SubnetId, leaseId,
			subnetLeases)
		if err != nil {
			return err
		} else if lease6 == nil {
			return nil
		}

		lease6.LeaseState = pbdhcpagent.LeaseState_RECLAIMED.String()
		lease6.Subnet6 = subnetId
		if _, err := tx.Insert(lease6); err != nil {
			return pg.Error(err)
		}

		return transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
			_, err = client.DeleteLease6(ctx,
				&pbdhcpagent.DeleteLease6Request{SubnetId: subnet6.SubnetId,
					LeaseType: lease6.LeaseType, Address: leaseId})
			return err
		})
	}); err != nil {
		return fmt.Errorf("delete lease6 %s with subnet6 %s failed: %s", leaseId, subnetId, err.Error())
	}

	return nil
}

func (l *SubnetLease6Service) BatchDeleteLease6s(subnetId string, leaseIds []string) error {
	for _, leaseId := range leaseIds {
		_, err := gohelperip.ParseIPv6(leaseId)
		if err != nil {
			return fmt.Errorf("subnet6 %s lease6 id %s is invalid: %s", subnetId, leaseId, err.Error())
		}
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnetId)
		if err != nil {
			return err
		}

		for _, leaseId := range leaseIds {
			_, subnetLeases, err := getReservation6sAndSubnetLease6sWithIp(
				tx, subnet6, leaseId)
			if err != nil {
				return err
			}

			lease6, err := GetSubnetLease6WithoutReclaimed(subnet6.SubnetId, leaseId,
				subnetLeases)
			if err != nil {
				return err
			} else if lease6 == nil {
				return nil
			}

			lease6.LeaseState = pbdhcpagent.LeaseState_RECLAIMED.String()
			lease6.Subnet6 = subnetId
			if _, err = tx.Insert(lease6); err != nil {
				return pg.Error(err)
			}

			if err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
				_, err = client.DeleteLease6(ctx,
					&pbdhcpagent.DeleteLease6Request{SubnetId: subnet6.SubnetId,
						LeaseType: lease6.LeaseType, Address: leaseId})
				return err
			}); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("batch delete lease6  with subnet6 %s failed: %s", subnetId, err.Error())
	}

	return nil
}

func GetSubnets6LeasesWithMacs(hwAddresses []string) ([]*resource.SubnetLease6, error) {
	var err error
	var resp *pbdhcpagent.GetLeases6Response
	if err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnets6LeasesWithMacs(ctx,
			&pbdhcpagent.GetSubnets6LeasesWithMacsRequest{HwAddresses: hwAddresses})
		return err
	}); err != nil {
		return nil, fmt.Errorf("get lease6s by mac failed: %s", err.Error())
	}

	subnetIds := make([]string, 0, len(resp.Leases))
	for i, lease := range resp.Leases {
		subnetIds[i] = fmt.Sprintf("%d", lease.SubnetId)
	}

	var reservations []*resource.Reservation6
	if err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&reservations, `select * from gr_reservation6 where subnet6 = any($1::text[])`, subnetIds)
	}); err != nil {
		return nil, fmt.Errorf("list reservation6s failed: %s", err.Error())
	}

	reservationMap := reservationMapFromReservation6s(reservations)

	leases := make([]*resource.SubnetLease6, len(resp.Leases))
	for i, lease := range resp.Leases {
		leases[i] = subnetLease6FromPbLease6AndReservations(lease, reservationMap)
	}

	return leases, nil
}
