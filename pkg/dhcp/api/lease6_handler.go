package api

import (
	"context"
	"fmt"
	"net"

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

type Lease6Handler struct{}

func NewLease6Handler() *Lease6Handler {
	return &Lease6Handler{}
}

func (l *Lease6Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	address, hasAddressFilter := util.GetFilterValueWithEqModifierFromFilters(
		util.FilterNameIp, ctx.GetFilters())
	if hasAddressFilter {
		if _, isv4, err := util.ParseIP(address); err != nil || isv4 {
			return nil, nil
		}
	}

	subnetId := ctx.Resource.GetParent().GetID()
	var subnets []*resource.Subnet6
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{restdb.IDField: subnetId}, &subnets); err != nil {
			return err
		}

		if len(subnets) == 0 {
			return fmt.Errorf("no found subnet6 %s from db", subnetId)
		}

		if hasAddressFilter {
			if subnets[0].Ipnet.Contains(net.ParseIP(address)) == false {
				return ErrorIpNotBelongToSubnet
			}

			return tx.FillEx(&reservations,
				"select * from gr_reservation6 where subnet6 = $1 and $2::text = any(ip_addresses)",
				subnetId, address)
		} else {
			return tx.Fill(map[string]interface{}{"subnet6": subnetId}, &reservations)
		}
	}); err != nil {
		if err == ErrorIpNotBelongToSubnet {
			return nil, nil
		} else {
			return nil, resterror.NewAPIError(resterror.ServerError,
				fmt.Sprintf("get subnet6 %s from db failed: %s", subnetId, err.Error()))
		}
	}

	reservationMap := reservationMapFromReservation6s(reservations)
	if hasAddressFilter {
		return getLease6sWithAddress(subnets[0].SubnetId, address, reservationMap)
	} else {
		return getLease6s(subnets[0].SubnetId, reservationMap)
	}
}

func getLease6sWithAddress(subnetId uint64, address string, reservationMap map[string]struct{}) (interface{}, *resterror.APIError) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Lease(context.TODO(),
		&dhcpagent.GetSubnet6LeaseRequest{Id: subnetId, Address: address})
	if err != nil {
		log.Debugf("get subnet6 %d leases failed: %s", subnetId, err.Error())
		return nil, nil
	}

	return []*resource.Lease6{lease6FromPbLease6(resp.GetLease(), reservationMap)}, nil
}

func getLease6s(subnetId uint64, reservationMap map[string]struct{}) (interface{}, *resterror.APIError) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Leases(context.TODO(),
		&dhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
	if err != nil {
		log.Debugf("get subnet6 %d leases failed: %s", subnetId, err.Error())
		return nil, nil
	}

	var leases []*resource.Lease6
	for _, lease := range resp.GetLeases() {
		leases = append(leases, lease6FromPbLease6(lease, reservationMap))
	}

	return leases, nil
}

func lease6FromPbLease6(lease *dhcpagent.DHCPLease6, reservationMap map[string]struct{}) *resource.Lease6 {
	lease6 := &resource.Lease6{
		Address:           lease.GetAddress(),
		AddressType:       resource.AddressTypeDynamic,
		PrefixLen:         lease.GetPrefixLen(),
		Duid:              lease.GetDuid(),
		Iaid:              lease.GetIaid(),
		HwAddress:         lease.GetHwAddress(),
		HwAddressType:     lease.GetHwType(),
		HwAddressSource:   lease.GetHwAddressSource(),
		ValidLifetime:     lease.GetValidLifetime(),
		PreferredLifetime: lease.GetPreferredLifetime(),
		Expire:            isoTimeFromUinx(lease.GetExpire()),
		LeaseType:         lease.GetLeaseType().String(),
		Hostname:          lease.GetHostname(),
		Fingerprint:       lease.GetFingerprint(),
		VendorId:          lease.GetVendorId(),
		OperatingSystem:   lease.GetOperatingSystem(),
		ClientType:        lease.GetClientType(),
		State:             lease.GetState(),
	}

	if _, ok := reservationMap[lease6.Address]; ok {
		lease6.AddressType = resource.AddressTypeReservation
	}
	lease6.SetID(lease.GetAddress())
	return lease6
}

func (l *Lease6Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnetId := ctx.Resource.GetParent().GetID()
	leaseId := ctx.Resource.GetID()
	ip, isv4, err := util.ParseIP(leaseId)
	if err != nil || isv4 {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("subnet %s lease6 id %s is invalid ipv6", subnetId, leaseId))
	}

	var subnets []*resource.Subnet6
	if _, err := restdb.GetResourceWithID(db.GetDB(), subnetId, &subnets); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet6 %s from db failed: %s", subnetId, err.Error()))
	}

	if subnets[0].Ipnet.Contains(ip) == false {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete lease %s failed: address not belongs to subnet6 %s",
				leaseId, subnets[0].Subnet))
	}

	leaseType := dhcpagent.DHCPv6LeaseType_IA_NA
	if ones, _ := subnets[0].Ipnet.Mask.Size(); ones < 64 {
		leaseType = dhcpagent.DHCPv6LeaseType_IA_PD
	}

	if _, err := grpcclient.GetDHCPAgentGrpcClient().DeleteLease6(context.TODO(),
		&dhcpagent.DeleteLease6Request{SubnetId: subnets[0].SubnetId,
			LeaseType: leaseType, Address: leaseId}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete lease %s with subnet6 %s failed: %s", leaseId, subnetId, err.Error()))
	} else {
		return nil
	}
}
