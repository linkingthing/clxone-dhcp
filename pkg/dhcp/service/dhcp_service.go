package service

import (
	"context"
	"fmt"
	"time"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"

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
		return nil, fmt.Errorf("get subnet4 %s leases count failed: %s",
			subnets[0].Subnet, err.Error())
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
		return nil, fmt.Errorf("get subnet6 %s leases count failed: %s",
			subnets[0].Subnet, err.Error())
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

	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		reservations, subnetLeases, err = GetReservation4sAndSubnetLease4sWithIp(tx, subnet.Id, ip)
		return err
	}); err != nil {
		log.Warnf("get reservations and reclaimed leases with subnet4 %s failed: %s",
			subnet.Subnet, err.Error())
		return subnet, nil, nil
	}

	lease4, err := GetSubnetLease4WithoutReclaimed(subnet.SubnetId, ip, reservations, subnetLeases)
	if err != nil {
		log.Warnf("get subnet lease with ip %s and subnet4 %s failed: %s",
			ip, subnet.Subnet, err.Error())
	}

	return subnet, pbdhcpLease4FromSubnetLease4(lease4), nil
}

func GetReservation4sAndSubnetLease4sWithIp(tx restdb.Transaction, subnetId, ip string) ([]*resource.Reservation4, []*resource.SubnetLease4, error) {
	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := tx.Fill(map[string]interface{}{"ip_address": ip, "subnet4": subnetId},
		&reservations); err != nil {
		return nil, nil, err
	}

	if err := tx.Fill(map[string]interface{}{"address": ip, "subnet4": subnetId},
		&subnetLeases); err != nil {
		return nil, nil, err
	}

	return reservations, subnetLeases, nil
}

func GetSubnetLease4WithoutReclaimed(subnetId uint64, ip string, reservations []*resource.Reservation4, subnetLeases []*resource.SubnetLease4) (*resource.SubnetLease4, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Lease(context.TODO(),
		&dhcpagent.GetSubnet4LeaseRequest{Id: subnetId, Address: ip})
	if err != nil {
		return nil, err
	}

	subnetLease4 := SubnetLease4FromPbLease4(resp.GetLease())
	for _, reclaimSubnetLease4 := range subnetLeases {
		if reclaimSubnetLease4.Equal(subnetLease4) {
			return nil, nil
		}
	}

	for _, reservation := range reservations {
		if reservation.IpAddress == subnetLease4.Address {
			subnetLease4.AddressType = resource.AddressTypeReservation
			break
		}
	}

	return subnetLease4, nil
}

func SubnetLease4FromPbLease4(lease *dhcpagent.DHCPLease4) *resource.SubnetLease4 {
	lease4 := &resource.SubnetLease4{
		Address:         lease.GetAddress(),
		AddressType:     resource.AddressTypeDynamic,
		HwAddress:       lease.GetHwAddress(),
		ClientId:        lease.GetClientId(),
		ValidLifetime:   lease.GetValidLifetime(),
		Expire:          timeFromUinx(lease.GetExpire()),
		Hostname:        lease.GetHostname(),
		Fingerprint:     lease.GetFingerprint(),
		VendorId:        lease.GetVendorId(),
		OperatingSystem: lease.GetOperatingSystem(),
		ClientType:      lease.GetClientType(),
		LeaseState:      lease.GetLeaseState().String(),
	}

	lease4.SetID(lease.GetAddress())
	return lease4
}

func timeFromUinx(t int64) string {
	return time.Unix(t, 0).Format(time.RFC3339)
}

func pbdhcpLease4FromSubnetLease4(lease4 *resource.SubnetLease4) *pbdhcp.Lease4 {
	if lease4 == nil {
		return nil
	}

	return &pbdhcp.Lease4{
		Address:         lease4.Address,
		AddressType:     string(lease4.AddressType),
		HwAddress:       lease4.HwAddress,
		ClientId:        lease4.ClientId,
		ValidLifetime:   lease4.ValidLifetime,
		Expire:          lease4.Expire,
		Hostname:        lease4.Hostname,
		VendorId:        lease4.VendorId,
		OperatingSystem: lease4.OperatingSystem,
		ClientType:      lease4.ClientType,
		LeaseState:      lease4.LeaseState,
	}
}

func (d *DHCPService) GetSubnet6AndLease6WithIp(ip string) (*pbdhcp.Subnet6, *pbdhcp.Lease6, error) {
	subnet, err := getSubnet6WithIp(ip)
	if err != nil {
		return nil, nil, err
	}

	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		reservations, subnetLeases, err = GetReservation6sAndSubnetLease6sWithIp(tx, subnet.Id, ip)
		return err
	}); err != nil {
		log.Warnf("get reservations and reclaimed leases with subnet6 %s failed: %s",
			subnet.Subnet, err.Error())
		return subnet, nil, nil
	}

	lease6, err := GetSubnetLease6WithoutReclaimed(subnet.SubnetId, ip, reservations, subnetLeases)
	if err != nil {
		log.Warnf("get subnet lease with ip %s and subnet6 %s failed: %s",
			ip, subnet.Subnet, err.Error())
	}

	return subnet, pbdhcpLease6FromSubnetLease6(lease6), nil
}

func GetReservation6sAndSubnetLease6sWithIp(tx restdb.Transaction, subnetId, ip string) ([]*resource.Reservation6, []*resource.SubnetLease6, error) {
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := tx.FillEx(&reservations,
		"select * from gr_reservation6 where subnet6 = $1 and $2::text = any(ip_addresses)",
		subnetId, ip); err != nil {
		return nil, nil, err
	}

	if err := tx.Fill(map[string]interface{}{"address": ip, "subnet6": subnetId},
		&subnetLeases); err != nil {
		return nil, nil, err
	}

	return reservations, subnetLeases, nil
}

func GetSubnetLease6WithoutReclaimed(subnetId uint64, ip string, reservations []*resource.Reservation6, subnetLeases []*resource.SubnetLease6) (*resource.SubnetLease6, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Lease(context.TODO(),
		&dhcpagent.GetSubnet6LeaseRequest{Id: subnetId, Address: ip})
	if err != nil {
		return nil, err
	}

	subnetLease6 := SubnetLease6FromPbLease6(resp.GetLease())
	for _, reclaimSubnetLease6 := range subnetLeases {
		if reclaimSubnetLease6.Equal(subnetLease6) {
			return nil, nil
		}
	}

	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			if ipAddress == subnetLease6.Address {
				subnetLease6.AddressType = resource.AddressTypeReservation
				break
			}
		}
	}

	return subnetLease6, nil
}

func SubnetLease6FromPbLease6(lease *dhcpagent.DHCPLease6) *resource.SubnetLease6 {
	lease6 := &resource.SubnetLease6{
		Address:           lease.GetAddress(),
		AddressType:       resource.AddressTypeDynamic,
		PrefixLen:         lease.GetPrefixLen(),
		Duid:              lease.GetDuid(),
		Iaid:              lease.GetIaid(),
		HwAddress:         lease.GetHwAddress(),
		HwAddressType:     lease.GetHwType(),
		HwAddressSource:   lease.GetHwAddressSource().String(),
		ValidLifetime:     lease.GetValidLifetime(),
		PreferredLifetime: lease.GetPreferredLifetime(),
		Expire:            timeFromUinx(lease.GetExpire()),
		LeaseType:         lease.GetLeaseType(),
		Hostname:          lease.GetHostname(),
		Fingerprint:       lease.GetFingerprint(),
		VendorId:          lease.GetVendorId(),
		OperatingSystem:   lease.GetOperatingSystem(),
		ClientType:        lease.GetClientType(),
		LeaseState:        lease.GetLeaseState().String(),
	}

	lease6.SetID(lease.GetAddress())
	return lease6
}

func pbdhcpLease6FromSubnetLease6(lease6 *resource.SubnetLease6) *pbdhcp.Lease6 {
	if lease6 == nil {
		return nil
	}

	return &pbdhcp.Lease6{
		Address:           lease6.Address,
		AddressType:       string(lease6.AddressType),
		PrefixLen:         lease6.PrefixLen,
		Duid:              lease6.Duid,
		Iaid:              lease6.Iaid,
		HwAddress:         lease6.HwAddress,
		HwAddressType:     lease6.HwAddressType,
		HwAddressSource:   lease6.HwAddressSource,
		ValidLifetime:     lease6.ValidLifetime,
		PreferredLifetime: lease6.PreferredLifetime,
		Expire:            lease6.Expire,
		LeaseType:         lease6.LeaseType,
		Hostname:          lease6.Hostname,
		VendorId:          lease6.VendorId,
		OperatingSystem:   lease6.OperatingSystem,
		ClientType:        lease6.ClientType,
		LeaseState:        lease6.LeaseState,
	}
}
