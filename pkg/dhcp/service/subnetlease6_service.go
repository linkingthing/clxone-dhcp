package service

import (
	"context"
	"fmt"
	"net"
	"strings"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	grpcservice "github.com/linkingthing/clxone-dhcp/pkg/grpc/service"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type SubnetLease6Service struct{}

func NewSubnetLease6Service() *SubnetLease6Service {
	return &SubnetLease6Service{}
}

func (l *SubnetLease6Service) List(ctx *restresource.Context) (interface{}, error) {
	ip, hasAddressFilter := util.GetFilterValueWithEqModifierFromFilters(
		util.FilterNameIp, ctx.GetFilters())
	if hasAddressFilter {
		if _, err := gohelperip.ParseIPv6(ip); err != nil {
			return nil, nil
		}
	}

	subnetId := ctx.Resource.GetParent().GetID()
	var subnet6SubnetId uint64
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnetId)
		if err != nil {
			return err
		}

		subnet6SubnetId = subnet6.SubnetId
		if hasAddressFilter {
			reservations, subnetLeases, err = getReservation6sAndSubnetLease6sWithIp(
				tx, subnet6, ip)
		} else {
			reservations, subnetLeases, err = getReservation6sAndSubnetLease6s(tx, subnetId)
		}
		return err
	}); err != nil {
		if err == ErrorIpNotBelongToSubnet {
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
	if subnet6.Ipnet.Contains(net.ParseIP(ip)) == false {
		return nil, nil, ErrorIpNotBelongToSubnet
	}

	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := tx.FillEx(&reservations,
		"select * from gr_reservation6 where subnet6 = $1 and $2::text = any(ip_addresses)",
		subnet6.GetID(), ip); err != nil {
		return nil, nil, fmt.Errorf("get reservation6 %s failed: %s", ip, err.Error())
	}

	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnAddress: ip,
		resource.SqlColumnSubnet6: subnet6.GetID()},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease6 %s failed: %s", ip, err.Error())
	}

	return reservations, subnetLeases, nil
}

func getReservation6sAndSubnetLease6s(tx restdb.Transaction, subnetId string) ([]*resource.Reservation6, []*resource.SubnetLease6, error) {
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&reservations); err != nil {
		return nil, nil, fmt.Errorf("get reservation6s failed: %s", err.Error())
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease6s failed: %s", err.Error())
	}

	return reservations, subnetLeases, nil
}

func getSubnetLease6sWithIp(subnetId uint64, ip string, reservations []*resource.Reservation6,
	subnetLeases []*resource.SubnetLease6) (interface{}, error) {
	lease6, err := grpcservice.GetSubnetLease6WithoutReclaimed(subnetId, ip,
		subnetLeases)
	if err != nil {
		log.Debugf("get subnet6 %d leases failed: %s", subnetId, err.Error())
		return nil, nil
	} else if lease6 == nil {
		return nil, nil
	}

	for _, reservation := range reservations {
		for _, ipaddress := range reservation.IpAddresses {
			if ipaddress == lease6.Address {
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

func getSubnetLease6s(subnetId uint64, reservations []*resource.Reservation6,
	subnetLeases []*resource.SubnetLease6) (interface{}, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Leases(context.TODO(),
		&pbdhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
	if err != nil {
		log.Debugf("get subnet6 %d leases failed: %s", subnetId, err.Error())
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
		log.Warnf("delete reclaim leases failed: %s", err.Error())
	}

	return leases, nil
}

func subnetLease6FromPbLease6AndReservations(lease *pbdhcpagent.DHCPLease6, reservationMap map[string]struct{}) *resource.SubnetLease6 {
	subnetLease6 := grpcservice.SubnetLease6FromPbLease6(lease)
	if _, ok := reservationMap[subnetLease6.Address]; ok {
		subnetLease6.AddressType = resource.AddressTypeReservation
	}
	return subnetLease6
}

func (l *SubnetLease6Service) Delete(subnetId, leaseId string) error {
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

		lease6, err := grpcservice.GetSubnetLease6WithoutReclaimed(subnet6.SubnetId, leaseId,
			subnetLeases)
		if err != nil {
			return err
		} else if lease6 == nil {
			return nil
		}

		lease6.LeaseState = pbdhcpagent.LeaseState_RECLAIMED.String()
		lease6.Subnet6 = subnetId
		if _, err := tx.Insert(lease6); err != nil {
			return err
		}

		_, err = grpcclient.GetDHCPAgentGrpcClient().DeleteLease6(context.TODO(),
			&pbdhcpagent.DeleteLease6Request{SubnetId: subnet6.SubnetId,
				LeaseType: lease6.LeaseType, Address: leaseId})
		return err
	}); err != nil {
		return err
	}

	return nil
}
