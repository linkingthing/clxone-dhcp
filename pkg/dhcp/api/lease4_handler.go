package api

import (
	"context"
	"fmt"
	"time"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	dhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Lease4Handler struct{}

func NewLease4Handler() *Lease4Handler {
	return &Lease4Handler{}
}

func (l *Lease4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetId := ctx.Resource.GetParent().GetID()
	var subnets []*resource.Subnet4
	if _, err := restdb.GetResourceWithID(db.GetDB(), subnetId, &subnets); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet4 %s from db failed: %s", subnetId, err.Error()))
	}

	if address, ok := util.GetFilterValueWithEqModifierFromFilters(util.FilterNameIp, ctx.GetFilters()); ok {
		return getLease4sWithAddress(subnets[0].SubnetId, address)
	} else {
		return getLease4s(subnets[0].SubnetId)
	}
}

func getLease4sWithAddress(subnetId uint64, address string) (interface{}, *resterror.APIError) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Lease(context.TODO(),
		&dhcpagent.GetSubnet4LeaseRequest{Id: subnetId, Address: address})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet4 %d leases failed: %s", subnetId, err.Error()))
	}

	return []*resource.Lease4{lease4FromPbLease4(resp.GetLease())}, nil
}

func getLease4s(subnetId uint64) (interface{}, *resterror.APIError) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Leases(context.TODO(),
		&dhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet4 %d leases failed: %s", subnetId, err.Error()))
	}

	var leases []*resource.Lease4
	for _, lease := range resp.GetLeases() {
		leases = append(leases, lease4FromPbLease4(lease))
	}

	return leases, nil
}

func lease4FromPbLease4(lease *dhcpagent.DHCPLease4) *resource.Lease4 {
	lease4 := &resource.Lease4{
		Address:         lease.GetAddress(),
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
	lease4.SetID(lease.GetAddress())
	return lease4
}

func isoTimeFromUinx(t int64) restresource.ISOTime {
	return restresource.ISOTime(time.Unix(t, 0))
}

func (l *Lease4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnetId := ctx.Resource.GetParent().GetID()
	leaseId := ctx.Resource.GetID()
	var subnets []*resource.Subnet4
	if _, err := restdb.GetResourceWithID(db.GetDB(), subnetId, &subnets); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet4 %s from db failed: %s", subnetId, err.Error()))
	}

	if _, err := grpcclient.GetDHCPAgentGrpcClient().DeleteLease4(context.TODO(),
		&dhcpagent.DeleteLease4Request{SubnetId: subnets[0].SubnetId, Address: leaseId}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete lease %s with subnet4 %s failed: %s", leaseId, subnetId, err.Error()))
	} else {
		return nil
	}
}
