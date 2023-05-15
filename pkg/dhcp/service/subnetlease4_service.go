package service

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
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
		if reservation.IpAddress == lease4.Address {
			lease4.AddressType = resource.AddressTypeReservation
			break
		}
	}

	lease4.HwAddress = strings.ToUpper(lease4.HwAddress)
	return []*resource.SubnetLease4{lease4}, nil
}

func GetSubnetLease4WithoutReclaimed(subnetId uint64, ip string, subnetLeases []*resource.SubnetLease4) (*resource.SubnetLease4, error) {
	var err error
	var resp *pbdhcpagent.GetLease4Response
	if err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet4Lease(ctx,
			&pbdhcpagent.GetSubnet4LeaseRequest{Id: subnetId, Address: ip})
		if err != nil {
			err = errorno.ErrNetworkError(errorno.ErrNameLease, err.Error())
		}
		return err
	}); err != nil {
		return nil, err
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
	lease4 := &resource.SubnetLease4{
		Address:               lease.GetAddress(),
		AddressType:           resource.AddressTypeDynamic,
		HwAddress:             lease.GetHwAddress(),
		HwAddressOrganization: lease.GetHwAddressOrganization(),
		ClientId:              lease.GetClientId(),
		ValidLifetime:         lease.GetValidLifetime(),
		Expire:                TimeFromUinx(lease.GetExpire()),
		Hostname:              lease.GetHostname(),
		Fingerprint:           lease.GetFingerprint(),
		VendorId:              lease.GetVendorId(),
		OperatingSystem:       lease.GetOperatingSystem(),
		ClientType:            lease.GetClientType(),
		LeaseState:            lease.GetLeaseState().String(),
	}

	lease4.SetID(lease.GetAddress())
	return lease4
}

func TimeFromUinx(t int64) string {
	return time.Unix(t, 0).Format(time.RFC3339)
}

func getSubnetLease4s(subnetId uint64, reservations []*resource.Reservation4, subnetLeases []*resource.SubnetLease4) ([]*resource.SubnetLease4, error) {
	var err error
	var resp *pbdhcpagent.GetLeases4Response
	if err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet4Leases(ctx,
			&pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
		if err != nil {
			err = errorno.ErrNetworkError(errorno.ErrNameLease, err.Error())
		}
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
			lease4.HwAddress = strings.ToUpper(lease4.HwAddress)
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
	if _, ok := reservationMap[subnetLease4.Address]; ok {
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
				err = errorno.ErrNetworkError(errorno.ErrNameLease, err.Error())
			}
			return err
		})
	})
}
