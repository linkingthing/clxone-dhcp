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

type SubnetLease6Service struct{}

func NewSubnetLease6Service() *SubnetLease6Service {
	return &SubnetLease6Service{}
}

func (l *SubnetLease6Service) List(subnet *resource.Subnet6, ip string) ([]*resource.SubnetLease6, error) {
	return ListSubnetLease6(subnet, ip)
}

func (l *SubnetLease6Service) ActionListToReservation(subnet *resource.Subnet6, input *resource.ConvToReservationInput) (
	*resource.ConvToReservationInput, error) {
	if len(input.Addresses) == 0 {
		return &resource.ConvToReservationInput{Data: []resource.ConvToReservationItem{}}, nil
	}

	leases, err := ListSubnetLease6(subnet, "")
	if err != nil {
		return nil, err
	}

	reservationLeases := make([]*resource.SubnetLease6, 0, len(leases))
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

func (l *SubnetLease6Service) filterAbleToReservation(subnetId string, leases []*resource.SubnetLease6,
	addresses []string) ([]*resource.SubnetLease6, []string, error) {
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet6: subnetId,
		}, &reservations)
	}); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	reservationLeases := make([]*resource.SubnetLease6, 0, len(addresses))
	hwAddresses := make([]string, 0, len(addresses))
outer:
	for _, address := range addresses {
		for _, lease := range leases {
			if lease.Address != address {
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
			for _, addr := range reservation.IpAddresses {
				if addr != address {
					continue
				} else if reservation.HwAddress != "" {
					hwAddresses = append(hwAddresses, reservation.HwAddress)
				}
				continue outer
			}
		}

		return nil, nil, errorno.ErrNotFound(errorno.ErrNameLease, address)
	}

	return reservationLeases, hwAddresses, nil
}

func (l *SubnetLease6Service) listToReservationWithMac(leases []*resource.SubnetLease6) (
	*resource.ConvToReservationInput, error) {
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
		dualStack := make([]string, 0, len(lease4s))
		for _, lease4 := range lease4s {
			if lease4.HwAddress == lease.HwAddress &&
				lease4.AddressType == resource.AddressTypeDynamic {
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

func (l *SubnetLease6Service) listToReservationWithHostname(leases []*resource.SubnetLease6) (
	*resource.ConvToReservationInput, error) {
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

	reservations, hwAddresses, ipv4MacMap, err := l.getReservationFromLease(input)
	if err != nil {
		return err
	}

	v6ReservationMap := map[string][]*resource.Reservation6{subnet.GetID(): reservations}

	if !input.BothV4V6 || input.ReservationType != resource.ReservationTypeMac {
		return createReservationsBySubnet(nil, v6ReservationMap)
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
		v4ReservationMap[lease4.Subnet4] = append(v4ReservationMap[lease4.Subnet4], &resource.Reservation4{
			IpAddress: lease4.Address,
			HwAddress: lease4.HwAddress,
		})
	}

	for ipv4, mac := range ipv4MacMap {
		return errorno.ErrNoResourceWith(errorno.ErrNameLease, errorno.ErrNameMac, ipv4, mac)
	}

	return createReservationsBySubnet(v4ReservationMap, v6ReservationMap)
}

func (l *SubnetLease6Service) getReservationFromLease(input *resource.ConvToReservationInput) (
	[]*resource.Reservation6, []string, map[string]string, error) {
	reservations := make([]*resource.Reservation6, 0, len(input.Data))
	hwAddresses := make([]string, 0, len(input.Data))
	ipv4MacMap := make(map[string]string, len(input.Data))
	seen := make(map[string][]string, len(input.Data))
	for _, item := range input.Data {
		var key string
		switch input.ReservationType {
		case resource.ReservationTypeMac:
			key = item.HwAddress
			if key == "" {
				return nil, nil, nil, errorno.ErrNotFound(errorno.ErrNameMac, item.Address)
			}
			hwAddresses = append(hwAddresses, key)
		case resource.ReservationTypeHostname:
			key = item.Hostname
			if key == "" {
				return nil, nil, nil, errorno.ErrNotFound(errorno.ErrNameHostname, item.Address)
			}
		default:
			return nil, nil, nil, fmt.Errorf("unsupported type %q", input.ReservationType)
		}

		seen[key] = append(seen[key], item.Address)
		for _, ipv4 := range item.DualStacks {
			ipv4MacMap[ipv4] = item.HwAddress
		}
	}

	for key, addresses := range seen {
		item := &resource.Reservation6{IpAddresses: addresses}

		switch input.ReservationType {
		case resource.ReservationTypeMac:
			item.HwAddress = key
		case resource.ReservationTypeHostname:
			item.Hostname = key
		}

		reservations = append(reservations, item)
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
			return nil, err
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
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnAddress: ip,
		resource.SqlColumnSubnet6: subnet6.GetID()},
		&subnetLeases); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkLease), pg.Error(err).Error())
	}

	return reservations, subnetLeases, nil
}

func getReservation6sAndSubnetLease6s(tx restdb.Transaction, subnetId string) ([]*resource.Reservation6, []*resource.SubnetLease6, error) {
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&reservations); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&subnetLeases); err != nil {
		return nil, nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameLease), pg.Error(err).Error())
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
			if ipaddress == lease6.Address &&
				(reservation.HwAddress != "" && strings.EqualFold(reservation.HwAddress, lease6.HwAddress) ||
					reservation.Hostname != "" && reservation.Hostname == lease6.Hostname ||
					reservation.Duid != "" && reservation.Duid == lease6.Duid) {
				lease6.AddressType = resource.AddressTypeReservation
				break
			}
		}

		for _, prefix := range reservation.Prefixes {
			if prefix == leasePrefix &&
				(reservation.HwAddress != "" && reservation.HwAddress == lease6.HwAddress ||
					reservation.Hostname != "" && reservation.Hostname == lease6.Hostname ||
					reservation.Duid != "" && reservation.Duid == lease6.Duid) {
				lease6.AddressType = resource.AddressTypeReservation
				break
			}
		}

		if lease6.AddressType == resource.AddressTypeReservation {
			break
		}
	}

	return []*resource.SubnetLease6{lease6}, nil
}

func GetSubnetLease6WithoutReclaimed(subnetId uint64, ip string, subnetLeases []*resource.SubnetLease6) (*resource.SubnetLease6, error) {
	var err error
	var resp *pbdhcpagent.GetLease6Response
	if err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet6Lease(ctx,
			&pbdhcpagent.GetSubnet6LeaseRequest{Id: subnetId, Address: ip})
		if err != nil {
			err = errorno.ErrNetworkError(errorno.ErrNameLease, err.Error())
		}
		return err
	}); err != nil {
		errMsg := err.Error()
		if s, ok := status.FromError(err); ok {
			errMsg = s.Message()
		}
		return nil, errorno.ErrNetworkError(errorno.ErrNameLease, errMsg)
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
		if err != nil {
			err = errorno.ErrNetworkError(errorno.ErrNameLease, err.Error())
		}
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

func subnetLease6FromPbLease6AndReservations(lease *pbdhcpagent.DHCPLease6, reservationMap map[string]*resource.Reservation6) *resource.SubnetLease6 {
	subnetLease6 := SubnetLease6FromPbLease6(lease)
	if reservation, ok := reservationMap[prefixFromAddressAndPrefixLen(subnetLease6.Address,
		subnetLease6.PrefixLen)]; ok &&
		(reservation.HwAddress != "" && strings.EqualFold(reservation.HwAddress, subnetLease6.HwAddress) ||
			reservation.Hostname != "" && reservation.Hostname == subnetLease6.Hostname ||
			reservation.Duid != "" && reservation.Duid == subnetLease6.Duid) {
		subnetLease6.AddressType = resource.AddressTypeReservation
	}
	return subnetLease6
}

func SubnetLease6FromPbLease6(lease *pbdhcpagent.DHCPLease6) *resource.SubnetLease6 {
	hwAddress, _ := util.NormalizeMac(lease.GetHwAddress())
	lease6 := &resource.SubnetLease6{
		Subnet6:               strconv.FormatUint(lease.GetSubnetId(), 10),
		Address:               lease.GetAddress(),
		AddressType:           resource.AddressTypeDynamic,
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
		AddressCodes:          lease.GetAddressCodes(),
		AddressCodeBegins:     lease.GetAddressCodeBegins(),
		AddressCodeEnds:       lease.GetAddressCodeEnds(),
		Subnet:                lease.GetSubnet(),
	}

	lease6.SetID(lease.GetAddress())
	return lease6
}

func (l *SubnetLease6Service) Delete(subnetId, leaseId string) error {
	_, err := gohelperip.ParseIPv6(leaseId)
	if err != nil {
		return errorno.ErrInvalidAddress(leaseId)
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
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
			return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameLease), pg.Error(err).Error())
		}

		return transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
			_, err = client.DeleteLease6(ctx,
				&pbdhcpagent.DeleteLease6Request{SubnetId: subnet6.SubnetId,
					LeaseType: lease6.LeaseType, Address: leaseId})
			if err != nil {
				errMsg := err.Error()
				if s, ok := status.FromError(err); ok {
					errMsg = s.Message()
					if strings.Contains(errMsg, "failed:") {
						msgs := strings.Split(errMsg, "failed:")
						errMsg = msgs[len(msgs)-1]
					}
				}
				err = errorno.ErrDBError(errorno.ErrDBNameDelete, string(errorno.ErrNameLease), errMsg)
			}
			return err
		})
	})
}

func (l *SubnetLease6Service) BatchDeleteLease6s(subnetId string, leaseIds []string) error {
	for _, leaseId := range leaseIds {
		_, err := gohelperip.ParseIPv6(leaseId)
		if err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameIp, leaseId)
		}
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnetId)
		if err != nil {
			return err
		}

		addrsByLeaseTypeMap := make(map[string][]string, len(leaseIds))
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

			addrsByLeaseTypeMap[lease6.LeaseType] = append(addrsByLeaseTypeMap[lease6.LeaseType], leaseId)
			lease6.LeaseState = pbdhcpagent.LeaseState_RECLAIMED.String()
			lease6.Subnet6 = subnetId
			if _, err = tx.Insert(lease6); err != nil {
				return errorno.ErrDBError(errorno.ErrDBNameInsert, string(errorno.ErrNameLease), pg.Error(err).Error())
			}
		}

		for leaseType, addrs := range addrsByLeaseTypeMap {
			if err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
				_, err = client.DeleteLeases6(ctx,
					&pbdhcpagent.DeleteLeases6Request{SubnetId: subnet6.SubnetId,
						LeaseType: leaseType, Addresses: addrs})
				return err
			}); err != nil {
				errMsg := err.Error()
				if s, ok := status.FromError(err); ok {
					errMsg = s.Message()
					if strings.Contains(errMsg, "failed:") {
						msgs := strings.Split(errMsg, "failed:")
						errMsg = msgs[len(msgs)-1]
					}
				}
				return errorno.ErrDBError(errorno.ErrDBNameDelete, string(errorno.ErrNameLease), errMsg)
			}
		}

		return nil
	})
}

func GetSubnets6LeasesWithMacs(hwAddresses []string) ([]*resource.SubnetLease6, error) {
	var err error
	var resp *pbdhcpagent.GetLeases6Response
	if err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnets6LeasesWithMacs(ctx,
			&pbdhcpagent.GetSubnets6LeasesWithMacsRequest{
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

	subnetIds := make([]string, 0, len(resp.Leases))
	seen := make(map[uint64]bool, len(resp.Leases))
	for _, lease := range resp.Leases {
		if !seen[lease.SubnetId] {
			seen[lease.SubnetId] = true
			subnetIds = append(subnetIds, fmt.Sprintf("%d", lease.SubnetId))
		}
	}

	var subnets []*resource.Subnet6
	var reservations []*resource.Reservation6
	if err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err = tx.FillEx(&subnets, `select * from gr_subnet6 where id = any($1::text[])`, subnetIds); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), err.Error())
		}
		if err = tx.FillEx(&reservations, `select * from gr_reservation6 where subnet6 = any($1::text[])`, subnetIds); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), err.Error())
		}
		return nil
	}); err != nil {
		return nil, err
	}

	reservationMap := reservationMapFromReservation6s(reservations)

	leases := make([]*resource.SubnetLease6, 0, len(resp.Leases))
	for _, lease := range resp.Leases {
		lease6 := subnetLease6FromPbLease6AndReservations(lease, reservationMap)
		for _, subnet := range subnets {
			if subnet.GetID() == lease6.Subnet6 {
				if subnet.UseEui64 {
					lease6.BelongEui64Subnet = true
				} else if subnet.AddressCode != "" {
					lease6.BelongAddrCodeSubnet = true
				}
				break
			}
		}

		leases = append(leases, lease6)
	}

	return leases, nil
}
