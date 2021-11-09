package service

import (
	"context"
	"fmt"
	"time"

	restdb "github.com/zdnscloud/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
	dhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

func init() {
	globalDHCPService = &DHCPService{}
}

var globalDHCPService *DHCPService

type DHCPService struct {
}

func GetDHCPService() *DHCPService {
	return globalDHCPService
}

func (d *DHCPService) GetSubnet4WithIp(ip string) (*pbdhcp.Subnet4, error) {
	return getSubnet4WithIp(ip)
}

func getSubnet4WithIp(ip string) (*pbdhcp.Subnet4, error) {
	var subnets []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "select * from gr_subnet4 where ipnet >> $1", ip)
	}); err != nil {
		return nil, fmt.Errorf("get subnet4 from db failed: %s", err.Error())
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet4 with ip %s", ip)
	}

	if leasesCount, err := GetSubnet4LeasesCount(subnets[0]); err != nil {
		return nil, fmt.Errorf("get subnet4 %s leases count failed: %s", subnets[0].Subnet, err.Error())
	} else {
		return pbdhcpSubnet4FromSubnet4(subnets[0], leasesCount), nil
	}
}

func GetSubnet4LeasesCount(subnet *resource.Subnet4) (uint64, error) {
	if subnet.Capacity == 0 {
		return 0, nil
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4LeasesCount(context.TODO(),
		&dhcpagent.GetSubnet4LeasesCountRequest{Id: subnet.SubnetId})
	return resp.GetLeasesCount(), err
}

func pbdhcpSubnet4FromSubnet4(subnet *resource.Subnet4, leasesCount uint64) *pbdhcp.Subnet4 {
	return &pbdhcp.Subnet4{
		Id:            subnet.GetID(),
		Subnet:        subnet.Subnet,
		SubnetId:      subnet.SubnetId,
		Capacity:      subnet.Capacity,
		UsedCount:     leasesCount,
		DomainServers: subnet.DomainServers,
		Routers:       subnet.Routers,
	}
}

func (d *DHCPService) GetSubnet6WithIp(ip string) (*pbdhcp.Subnet6, error) {
	return getSubnet6WithIp(ip)
}

func getSubnet6WithIp(ip string) (*pbdhcp.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "select * from gr_subnet6 where ipnet >> $1", ip)
	}); err != nil {
		return nil, fmt.Errorf("get subnet6 from db failed: %s", err.Error())
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet6 with ip %s", ip)
	}

	if leasesCount, err := GetSubnet6LeasesCount(subnets[0]); err != nil {
		return nil, fmt.Errorf("get subnet6 %s leases count failed: %s", subnets[0].Subnet, err.Error())
	} else {
		return pbdhcpSubnet6FromSubnet6(subnets[0], leasesCount), nil
	}
}

func GetSubnet6LeasesCount(subnet *resource.Subnet6) (uint64, error) {
	if subnet.Capacity == 0 {
		return 0, nil
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6LeasesCount(context.TODO(),
		&dhcpagent.GetSubnet6LeasesCountRequest{Id: subnet.SubnetId})
	return resp.GetLeasesCount(), err
}

func pbdhcpSubnet6FromSubnet6(subnet *resource.Subnet6, leasesCount uint64) *pbdhcp.Subnet6 {
	return &pbdhcp.Subnet6{
		Id:            subnet.GetID(),
		Subnet:        subnet.Subnet,
		SubnetId:      subnet.SubnetId,
		Capacity:      subnet.Capacity,
		UsedCount:     leasesCount,
		DomainServers: subnet.DomainServers,
	}
}

func (d *DHCPService) GetSubnet4AndLease4WithIp(ip string) (*pbdhcp.Subnet4, *pbdhcp.Lease4, error) {
	subnet, err := getSubnet4WithIp(ip)
	if err != nil {
		return nil, nil, err
	}

	if resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Lease(context.TODO(),
		&dhcpagent.GetSubnet4LeaseRequest{Id: subnet.GetSubnetId(), Address: ip}); err != nil {
		return nil, nil, fmt.Errorf("get lease %s with subnet %s failed: %s",
			ip, subnet.GetSubnet(), err.Error())
	} else {
		return subnet, pbdhcpLease4FromDHCPAgentDHCPLease4(resp.GetLease()), nil
	}
}

func pbdhcpLease4FromDHCPAgentDHCPLease4(lease *dhcpagent.DHCPLease4) *pbdhcp.Lease4 {
	return &pbdhcp.Lease4{
		Address:         lease.GetAddress(),
		HwAddress:       lease.GetHwAddress(),
		ClientId:        lease.GetClientId(),
		ValidLifetime:   lease.GetValidLifetime(),
		Expire:          time.Unix(lease.GetExpire(), 0).Format(time.RFC3339),
		Hostname:        lease.GetHostname(),
		VendorId:        lease.GetVendorId(),
		OperatingSystem: lease.GetOperatingSystem(),
		ClientType:      lease.GetClientType(),
		State:           lease.GetState(),
	}
}

func (d *DHCPService) GetSubnet6AndLease6WithIp(ip string) (*pbdhcp.Subnet6, *pbdhcp.Lease6, error) {
	subnet, err := getSubnet6WithIp(ip)
	if err != nil {
		return nil, nil, err
	}

	if resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Lease(context.TODO(),
		&dhcpagent.GetSubnet6LeaseRequest{Id: subnet.GetSubnetId(), Address: ip}); err != nil {
		return nil, nil, fmt.Errorf("get lease %s with subnet %s failed: %s",
			ip, subnet.GetSubnet(), err.Error())
	} else {
		return subnet, pbdhcpLease6FromDHCPAgentDHCPLease6(resp.GetLease()), nil
	}
}

func pbdhcpLease6FromDHCPAgentDHCPLease6(lease *dhcpagent.DHCPLease6) *pbdhcp.Lease6 {
	return &pbdhcp.Lease6{
		Address:           lease.GetAddress(),
		PrefixLen:         lease.GetPrefixLen(),
		Duid:              lease.GetDuid(),
		Iaid:              lease.GetIaid(),
		HwAddress:         lease.GetHwAddress(),
		HwAddressType:     lease.GetHwType(),
		HwAddressSource:   lease.GetHwAddressSource(),
		ValidLifetime:     lease.GetValidLifetime(),
		PreferredLifetime: lease.GetPreferredLifetime(),
		Expire:            time.Unix(lease.GetExpire(), 0).Format(time.RFC3339),
		LeaseType:         lease.GetLeaseType().String(),
		Hostname:          lease.GetHostname(),
		VendorId:          lease.GetVendorId(),
		OperatingSystem:   lease.GetOperatingSystem(),
		ClientType:        lease.GetClientType(),
		State:             lease.GetState(),
	}
}
