package api

import (
	"context"
	"fmt"

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
	subnetId := ctx.Resource.GetParent().GetID()
	var subnets []*resource.Subnet6
	if _, err := restdb.GetResourceWithID(db.GetDB(), subnetId, &subnets); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet6 %s from db failed: %s", subnetId, err.Error()))
	}

	if address, ok := util.GetFilterValueWithEqModifierFromFilters(util.FilterNameIp, ctx.GetFilters()); ok {
		return getLease6sWithAddress(subnets[0].SubnetId, address)
	} else {
		return getLease6s(subnets[0].SubnetId)
	}
}

func getLease6sWithAddress(subnetId uint64, address string) (interface{}, *resterror.APIError) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Lease(context.TODO(),
		&dhcpagent.GetSubnet6LeaseRequest{Id: subnetId, Address: address})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet6 %d leases failed: %s", subnetId, err.Error()))
	}

	return []*resource.Lease6{lease6FromPbLease6(resp.GetLease())}, nil
}

func getLease6s(subnetId uint64) (interface{}, *resterror.APIError) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Leases(context.TODO(),
		&dhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet6 %d leases failed: %s", subnetId, err.Error()))
	}

	var leases []*resource.Lease6
	for _, lease := range resp.GetLeases() {
		leases = append(leases, lease6FromPbLease6(lease))
	}

	return leases, nil
}

func lease6FromPbLease6(lease *dhcpagent.DHCPLease6) *resource.Lease6 {
	lease6 := &resource.Lease6{
		Address:           lease.GetAddress(),
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
	lease6.SetID(lease.GetAddress())
	return lease6
}
