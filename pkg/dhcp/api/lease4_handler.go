package api

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	dhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var ErrorIpNotBelongToSubnet = fmt.Errorf("ip not belongs to subnet")

type Lease4Handler struct{}

func NewLease4Handler() *Lease4Handler {
	return &Lease4Handler{}
}

func (l *Lease4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	address, hasAddressFilter := util.GetFilterValueWithEqModifierFromFilters(
		util.FilterNameIp, ctx.GetFilters())
	if hasAddressFilter {
		if _, isv4, err := util.ParseIP(address); err != nil || isv4 == false {
			return nil, nil
		}
	}

	subnetId := ctx.Resource.GetParent().GetID()
	var subnets []*resource.Subnet4
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{restdb.IDField: subnetId}, &subnets); err != nil {
			return err
		}

		if len(subnets) == 0 {
			return fmt.Errorf("no found subnet4 %s from db", subnetId)
		}

		if hasAddressFilter {
			if subnets[0].Ipnet.Contains(net.ParseIP(address)) == false {
				return ErrorIpNotBelongToSubnet
			}

			return tx.Fill(map[string]interface{}{
				"ip_address": address, "subnet4": subnetId}, &reservations)
		} else {
			return tx.Fill(map[string]interface{}{"subnet4": subnetId}, &reservations)
		}
	}); err != nil {
		if err == ErrorIpNotBelongToSubnet {
			return nil, nil
		} else {
			return nil, resterror.NewAPIError(resterror.ServerError,
				fmt.Sprintf("get subnet4 %s from db failed: %s", subnetId, err.Error()))
		}
	}

	reservationMap := reservationMapFromReservation4s(reservations)
	if hasAddressFilter {
		return getLease4sWithAddress(subnets[0].SubnetId, address, reservationMap)
	} else {
		return getLease4s(subnets[0].SubnetId, reservationMap)
	}
}

func getLease4sWithAddress(subnetId uint64, address string, reservationMap map[string]string) (interface{}, *resterror.APIError) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Lease(context.TODO(),
		&dhcpagent.GetSubnet4LeaseRequest{Id: subnetId, Address: address})
	if err != nil {
		log.Debugf("get subnet4 %d leases failed: %s", subnetId, err.Error())
		return nil, nil
	}

	return []*resource.Lease4{lease4FromPbLease4(resp.GetLease(), reservationMap)}, nil
}

func getLease4s(subnetId uint64, reservationMap map[string]string) (interface{}, *resterror.APIError) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Leases(context.TODO(),
		&dhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
	if err != nil {
		log.Debugf("get subnet4 %d leases failed: %s", subnetId, err.Error())
		return nil, nil
	}

	var leases []*resource.Lease4
	for _, lease := range resp.GetLeases() {
		leases = append(leases, lease4FromPbLease4(lease, reservationMap))
	}

	return leases, nil
}

func lease4FromPbLease4(lease *dhcpagent.DHCPLease4, reservationMap map[string]string) *resource.Lease4 {
	lease4 := &resource.Lease4{
		Address:         lease.GetAddress(),
		AddressType:     resource.AddressTypeDynamic,
		HwAddress:       lease.GetHwAddress(),
		ClientId:        lease.GetClientId(),
		ValidLifetime:   lease.GetValidLifetime(),
		Expire:          isoTimeFromUinx(lease.GetExpire()),
		Hostname:        lease.GetHostname(),
		Fingerprint:     lease.GetFingerprint(),
		VendorId:        lease.GetVendorId(),
		OperatingSystem: lease.GetOperatingSystem(),
		ClientType:      lease.GetClientType(),
		State:           lease.GetState(),
	}

	if _, ok := reservationMap[lease4.Address]; ok {
		lease4.AddressType = resource.AddressTypeReservation
	}

	lease4.SetID(lease.GetAddress())
	return lease4
}

func isoTimeFromUinx(t int64) restresource.ISOTime {
	return restresource.ISOTime(time.Unix(t, 0))
}

func (l *Lease4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnetId := ctx.Resource.GetParent().GetID()
	leaseId := ctx.Resource.GetID()
	ip, isv4, err := util.ParseIP(leaseId)
	if err != nil || isv4 == false {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("subnet %s lease4 id %s is invalid ipv4", subnetId, leaseId))
	}

	var subnets []*resource.Subnet4
	if _, err := restdb.GetResourceWithID(db.GetDB(), subnetId, &subnets); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet4 %s from db failed: %s", subnetId, err.Error()))
	}

	if subnets[0].Ipnet.Contains(ip) == false {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete lease %s failed: address not belongs to subnet4 %s",
				leaseId, subnets[0].Subnet))
	}

	if _, err := grpcclient.GetDHCPAgentGrpcClient().DeleteLease4(context.TODO(),
		&dhcpagent.DeleteLease4Request{SubnetId: subnets[0].SubnetId, Address: leaseId}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete lease %s with subnet4 %s failed: %s", leaseId, subnetId, err.Error()))
	} else {
		return nil
	}
}
