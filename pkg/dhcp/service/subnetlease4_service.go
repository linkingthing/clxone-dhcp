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

var ErrorIpNotBelongToSubnet = fmt.Errorf("ip not belongs to subnet")

type SubnetLease4Service struct{}

func NewSubnetLease4Service() *SubnetLease4Service {
	return &SubnetLease4Service{}
}

func (l *SubnetLease4Service) List(ctx *restresource.Context) (interface{}, error) {
	ip, hasAddressFilter := util.GetFilterValueWithEqModifierFromFilters(
		util.FilterNameIp, ctx.GetFilters())
	if hasAddressFilter {
		if _, err := gohelperip.ParseIPv4(ip); err != nil {
			return nil, nil
		}
	}

	subnetId := ctx.Resource.GetParent().GetID()
	var subnet4SubnetId uint64
	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, subnetId)
		if err != nil {
			return err
		}

		subnet4SubnetId = subnet4.SubnetId
		if hasAddressFilter {
			reservations, subnetLeases, err = getReservation4sAndSubnetLease4sWithIp(
				tx, subnet4, ip)
		} else {
			reservations, subnetLeases, err = getReservation4sAndSubnetLease4s(
				tx, subnetId)
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
		return getSubnetLease4sWithIp(subnet4SubnetId, ip, reservations, subnetLeases)
	} else {
		return getSubnetLease4s(subnet4SubnetId, reservations, subnetLeases)
	}
}

func getReservation4sAndSubnetLease4sWithIp(tx restdb.Transaction, subnet4 *resource.Subnet4, ip string) ([]*resource.Reservation4, []*resource.SubnetLease4, error) {
	if subnet4.Ipnet.Contains(net.ParseIP(ip)) == false {
		return nil, nil, ErrorIpNotBelongToSubnet
	}

	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnAddress: ip,
		resource.SqlColumnSubnet4: subnet4.GetID()},
		&reservations); err != nil {
		return nil, nil, fmt.Errorf("get reservation4 %s failed: %s", ip, err.Error())
	}

	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnAddress: ip,
		resource.SqlColumnSubnet4: subnet4.GetID()},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease4 %s failed: %s", ip, err.Error())
	}

	return reservations, subnetLeases, nil
}

func getReservation4sAndSubnetLease4s(tx restdb.Transaction, subnetId string) ([]*resource.Reservation4, []*resource.SubnetLease4, error) {
	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnetId},
		&reservations); err != nil {
		return nil, nil, fmt.Errorf("get reservation4s failed: %s", err.Error())
	}

	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnetId},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease4s failed: %s", err.Error())
	}

	return reservations, subnetLeases, nil
}

func getSubnetLease4sWithIp(subnetId uint64, ip string, reservations []*resource.Reservation4,
	subnetLeases []*resource.SubnetLease4) (interface{}, error) {
	lease4, err := grpcservice.GetSubnetLease4WithoutReclaimed(subnetId, ip,
		subnetLeases)
	if err != nil {
		log.Debugf("get subnet4 %d leases failed: %s", subnetId, err.Error())
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

	return []*resource.SubnetLease4{lease4}, nil
}

func getSubnetLease4s(subnetId uint64, reservations []*resource.Reservation4,
	subnetLeases []*resource.SubnetLease4) (interface{}, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Leases(context.TODO(),
		&pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
	if err != nil {
		log.Debugf("get subnet4 %d leases failed: %s", subnetId, err.Error())
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
		log.Warnf("delete reclaim leases failed: %s", err.Error())
	}

	return leases, nil
}

func subnetLease4FromPbLease4AndReservations(lease *pbdhcpagent.DHCPLease4, reservationMap map[string]string) *resource.SubnetLease4 {
	subnetLease4 := grpcservice.SubnetLease4FromPbLease4(lease)
	if _, ok := reservationMap[subnetLease4.Address]; ok {
		subnetLease4.AddressType = resource.AddressTypeReservation
	}
	return subnetLease4
}

func (l *SubnetLease4Service) Delete(subnetId, leaseId string) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, subnetId)
		if err != nil {
			return err
		}

		_, subnetLeases, err := getReservation4sAndSubnetLease4sWithIp(
			tx, subnet4, leaseId)
		if err != nil {
			return err
		}

		lease4, err := grpcservice.GetSubnetLease4WithoutReclaimed(subnet4.SubnetId, leaseId,
			subnetLeases)
		if err != nil {
			return err
		} else if lease4 == nil {
			return nil
		}

		lease4.LeaseState = pbdhcpagent.LeaseState_RECLAIMED.String()
		lease4.Subnet4 = subnetId
		if _, err := tx.Insert(lease4); err != nil {
			return err
		}

		_, err = grpcclient.GetDHCPAgentGrpcClient().DeleteLease4(context.TODO(),
			&pbdhcpagent.DeleteLease4Request{SubnetId: subnet4.SubnetId, Address: leaseId})
		return err
	}); err != nil {
		return err
	}

	return nil
}
