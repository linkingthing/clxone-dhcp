package service

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
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

func (d *DHCPService) GetSubnet4WithIp(ip string) (map[string]*pbdhcp.Subnet4, error) {
	return getSubnet4WithIp(ip)
}

func getSubnet4WithIp(ip string) (map[string]*pbdhcp.Subnet4, error) {
	if _, err := gohelperip.ParseIPv4(ip); err != nil {
		return nil, err
	}

	var subnets []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "select * from gr_subnet4 where ipnet >> $1", ip)
	}); err != nil {
		return nil, fmt.Errorf("get subnet4 from db failed: %s", err.Error())
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet4 with ip %s", ip)
	}

	return getPbDHCPSubnet4sFromSubnet4s(map[string]*resource.Subnet4{ip: subnets[0]})
}

func getPbDHCPSubnet4sFromSubnet4s(subnets map[string]*resource.Subnet4) (map[string]*pbdhcp.Subnet4, error) {
	if leasesCount, err := GetSubnet4sLeasesCount(subnets); err != nil {
		return nil, fmt.Errorf("get subnet4s leases count failed: %s", err.Error())
	} else {
		return pbdhcpSubnet4sFromSubnet4s(subnets, leasesCount), nil
	}
}

func GetSubnet4sLeasesCount(subnets map[string]*resource.Subnet4) (map[uint64]uint64, error) {
	var subnetIds []uint64
	for _, subnet := range subnets {
		if subnet.Capacity != 0 {
			subnetIds = append(subnetIds, subnet.SubnetId)
		}
	}

	if len(subnetIds) == 0 {
		return nil, nil
	} else {
		resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnets4LeasesCountWithIds(context.TODO(),
			&pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: subnetIds})
		return resp.GetSubnetsLeasesCount(), err
	}
}

func pbdhcpSubnet4sFromSubnet4s(subnets map[string]*resource.Subnet4, leasesCount map[uint64]uint64) map[string]*pbdhcp.Subnet4 {
	pbsubnets := make(map[string]*pbdhcp.Subnet4)
	for ip, subnet := range subnets {
		pbsubnets[ip] = pbdhcpSubnet4FromSubnet4(subnet, leasesCount[subnet.SubnetId])
	}

	return pbsubnets
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

func (d *DHCPService) GetSubnet6WithIp(ip string) (map[string]*pbdhcp.Subnet6, error) {
	return getSubnet6WithIp(ip)
}

func getSubnet6WithIp(ip string) (map[string]*pbdhcp.Subnet6, error) {
	if _, err := gohelperip.ParseIPv6(ip); err != nil {
		return nil, err
	}

	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, "select * from gr_subnet6 where ipnet >> $1", ip)
	}); err != nil {
		return nil, fmt.Errorf("get subnet6 from db failed: %s", err.Error())
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet6 with ip %s", ip)
	}

	return getPbDHCPSubnet6sFromSubnet6s(map[string]*resource.Subnet6{ip: subnets[0]})
}

func getPbDHCPSubnet6sFromSubnet6s(subnets map[string]*resource.Subnet6) (map[string]*pbdhcp.Subnet6, error) {
	if leasesCount, err := GetSubnet6sLeasesCount(subnets); err != nil {
		return nil, fmt.Errorf("get subnet6 leases count failed: %s", err.Error())
	} else {
		return pbdhcpSubnet6sFromSubnet6s(subnets, leasesCount), nil
	}
}

func GetSubnet6sLeasesCount(subnets map[string]*resource.Subnet6) (map[uint64]uint64, error) {
	var subnetIds []uint64
	for _, subnet := range subnets {
		if subnet.Capacity != 0 {
			subnetIds = append(subnetIds, subnet.SubnetId)
		}
	}

	if len(subnetIds) == 0 {
		return nil, nil
	} else {
		resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnets6LeasesCountWithIds(context.TODO(),
			&pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: subnetIds})
		return resp.GetSubnetsLeasesCount(), err
	}
}

func pbdhcpSubnet6sFromSubnet6s(subnets map[string]*resource.Subnet6, leasesCount map[uint64]uint64) map[string]*pbdhcp.Subnet6 {
	pbsubnets := make(map[string]*pbdhcp.Subnet6)
	for ip, subnet := range subnets {
		pbsubnets[ip] = pbdhcpSubnet6FromSubnet6(subnet, leasesCount[subnet.SubnetId])
	}

	return pbsubnets
}

func pbdhcpSubnet6FromSubnet6(subnet *resource.Subnet6, leasesCount uint64) *pbdhcp.Subnet6 {
	return &pbdhcp.Subnet6{
		Id:            subnet.GetID(),
		Subnet:        subnet.Subnet,
		SubnetId:      subnet.SubnetId,
		Capacity:      subnet.Capacity,
		UsedCount:     leasesCount,
		DomainServers: subnet.DomainServers,
		UseEui64:      subnet.UseEui64,
	}
}

func (d *DHCPService) GetSubnets4WithIps(ips []string) (map[string]*pbdhcp.Subnet4, error) {
	return getSubnets4WithIps(ips)
}

func getSubnets4WithIps(ips []string) (map[string]*pbdhcp.Subnet4, error) {
	if err := gohelperip.CheckIPv4sValid(ips...); err != nil {
		return nil, err
	}

	var subnets []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &subnets)
	}); err != nil {
		return nil, fmt.Errorf("get subnet4s from db failed: %s", err.Error())
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no subnet4s")
	}

	closestSubnets := make(map[string]*resource.Subnet4)
	for _, ip := range ips {
		for _, subnet := range subnets {
			if subnet.Contains(ip) {
				closestSubnets[ip] = subnet
				break
			}
		}
	}

	return getPbDHCPSubnet4sFromSubnet4s(closestSubnets)
}

func (d *DHCPService) GetSubnets6WithIps(ips []string) (map[string]*pbdhcp.Subnet6, error) {
	return getSubnets6WithIps(ips)
}

func getSubnets6WithIps(ips []string) (map[string]*pbdhcp.Subnet6, error) {
	if err := gohelperip.CheckIPv6sValid(ips...); err != nil {
		return nil, nil
	}

	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &subnets)
	}); err != nil {
		return nil, fmt.Errorf("get subnet6s from db failed: %s", err.Error())
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("get subnet6s from db is nil")
	}

	closestSubnets := make(map[string]*resource.Subnet6)
	for _, ip := range ips {
		for _, subnet := range subnets {
			if subnet.Contains(ip) {
				closestSubnets[ip] = subnet
				break
			}
		}
	}

	return getPbDHCPSubnet6sFromSubnet6s(closestSubnets)
}

func (d *DHCPService) GetSubnet4AndLease4WithIp(ip string) (map[string]*pbdhcp.Ipv4Information, error) {
	subnets, err := getSubnet4WithIp(ip)
	if err != nil {
		return nil, err
	}

	subnet := subnets[ip]
	ipv4Info := map[string]*pbdhcp.Ipv4Information{
		ip: &pbdhcp.Ipv4Information{
			Address: ip,
			Subnet:  subnet,
		}}

	var addressType resource.AddressType
	var subnetLeases []*resource.SubnetLease4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		addressType, subnetLeases, err = GetAddressTypeAndSubnetLease4sWithIp(tx, subnet.Id, ip)
		return err
	}); err != nil {
		log.Warnf("get address type and reclaimed leases with subnet4 %s failed: %s",
			subnet.Subnet, err.Error())
		return ipv4Info, nil
	}

	lease4, err := GetSubnetLease4WithoutReclaimed(subnet.SubnetId, ip, subnetLeases)
	if err != nil {
		log.Warnf("get subnet lease with ip %s and subnet4 %s failed: %s",
			ip, subnet.Subnet, err.Error())
	}

	ipv4Info[ip].AddressType = addressType.String()
	ipv4Info[ip].Lease = pbdhcpLease4FromSubnetLease4(lease4)
	return ipv4Info, nil
}

func GetAddressTypeAndSubnetLease4sWithIp(tx restdb.Transaction, subnetId, ip string) (resource.AddressType, []*resource.SubnetLease4, error) {
	addressType, err := GetIPv4AddressType(tx, subnetId, ip)
	if err != nil {
		return addressType, nil, err
	}

	var subnetLeases []*resource.SubnetLease4
	err = tx.Fill(map[string]interface{}{"address": ip, "subnet4": subnetId}, &subnetLeases)
	return addressType, subnetLeases, err
}

func GetIPv4AddressType(tx restdb.Transaction, subnetId, ip string) (resource.AddressType, error) {
	addressType := resource.AddressTypeExclusion
	if exists, err := tx.Exists(resource.TableReservation4,
		map[string]interface{}{"ip_address": ip, "subnet4": subnetId}); err != nil {
		return addressType, fmt.Errorf("check ip %s in reservation4 failed: %s", ip, err.Error())
	} else if exists {
		return resource.AddressTypeReservation, nil
	}

	if count, err := tx.CountEx(resource.TableReservedPool4,
		"select count(*) from gr_reserved_pool4 where subnet4 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetId, ip, ip); err != nil {
		return addressType, fmt.Errorf("check ip %s in reserved pool4 failed: %s", ip, err.Error())
	} else if count != 0 {
		return resource.AddressTypeReserve, nil
	}

	if count, err := tx.CountEx(resource.TablePool4,
		"select count(*) from gr_pool4 where subnet4 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetId, ip, ip); err != nil {
		return addressType, fmt.Errorf("check ip %s in pool4 failed: %s", ip, err.Error())
	} else if count != 0 {
		return resource.AddressTypeDynamic, nil
	}

	return addressType, nil
}

func GetSubnetLease4WithoutReclaimed(subnetId uint64, ip string, subnetLeases []*resource.SubnetLease4) (*resource.SubnetLease4, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Lease(context.TODO(),
		&pbdhcpagent.GetSubnet4LeaseRequest{Id: subnetId, Address: ip})
	if err != nil {
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
		Expire:                timeFromUinx(lease.GetExpire()),
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

func timeFromUinx(t int64) string {
	return time.Unix(t, 0).Format(time.RFC3339)
}

func pbdhcpLease4FromSubnetLease4(lease4 *resource.SubnetLease4) *pbdhcp.Lease4 {
	if lease4 == nil {
		return nil
	}

	return &pbdhcp.Lease4{
		Address:               lease4.Address,
		HwAddress:             lease4.HwAddress,
		HwAddressOrganization: lease4.HwAddressOrganization,
		ClientId:              lease4.ClientId,
		ValidLifetime:         lease4.ValidLifetime,
		Expire:                lease4.Expire,
		Hostname:              lease4.Hostname,
		VendorId:              lease4.VendorId,
		OperatingSystem:       lease4.OperatingSystem,
		ClientType:            lease4.ClientType,
		LeaseState:            lease4.LeaseState,
	}
}

func (d *DHCPService) GetSubnet6AndLease6WithIp(ip string) (map[string]*pbdhcp.Ipv6Information, error) {
	subnets, err := getSubnet6WithIp(ip)
	if err != nil {
		return nil, err
	}

	subnet := subnets[ip]
	ipv6Info := map[string]*pbdhcp.Ipv6Information{
		ip: &pbdhcp.Ipv6Information{
			Address: ip,
			Subnet:  subnet,
		}}

	var addressType resource.AddressType
	var subnetLeases []*resource.SubnetLease6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		addressType, subnetLeases, err = GetAddressTypeAndSubnetLease6sWithIp(tx, subnet.Id, ip)
		return err
	}); err != nil {
		log.Warnf("get address type and reclaimed leases with subnet6 %s failed: %s",
			subnet.Subnet, err.Error())
		return ipv6Info, nil
	}

	lease6, err := GetSubnetLease6WithoutReclaimed(subnet.SubnetId, ip, subnetLeases)
	if err != nil {
		log.Warnf("get subnet lease with ip %s and subnet6 %s failed: %s",
			ip, subnet.Subnet, err.Error())
	}

	ipv6Info[ip].AddressType = addressType.String()
	ipv6Info[ip].Lease = pbdhcpLease6FromSubnetLease6(lease6)
	return ipv6Info, nil
}

func GetAddressTypeAndSubnetLease6sWithIp(tx restdb.Transaction, subnetId, ip string) (resource.AddressType, []*resource.SubnetLease6, error) {
	addressType, err := GetIPv6AddressType(tx, subnetId, ip)
	if err != nil {
		return addressType, nil, err
	}

	var subnetLeases []*resource.SubnetLease6
	if err := tx.Fill(map[string]interface{}{"address": ip, "subnet6": subnetId},
		&subnetLeases); err != nil {
		return addressType, nil, err
	}

	return addressType, subnetLeases, nil
}

func GetIPv6AddressType(tx restdb.Transaction, subnetId, ip string) (resource.AddressType, error) {
	addressType := resource.AddressTypeExclusion
	if count, err := tx.CountEx(resource.TableReservation6,
		"select count(*) from gr_reservation6 where subnet6 = $1 and $2::text = any(ip_addresses)",
		subnetId, ip); err != nil {
		return addressType, fmt.Errorf("check ip %s in reservation6 failed: %s", ip, err.Error())
	} else if count != 0 {
		return resource.AddressTypeReservation, nil
	}

	if count, err := tx.CountEx(resource.TableReservedPool6,
		"select count(*) from gr_reserved_pool6 where subnet6 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetId, ip, ip); err != nil {
		return addressType, fmt.Errorf("check ip %s in reserved pool6 failed: %s", ip, err.Error())
	} else if count != 0 {
		return resource.AddressTypeReserve, nil
	}

	if count, err := tx.CountEx(resource.TablePool6,
		"select count(*) from gr_pool6 where subnet6 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetId, ip, ip); err != nil {
		return addressType, fmt.Errorf("check ip %s in pool6 failed: %s", ip, err.Error())
	} else if count != 0 {
		return resource.AddressTypeDynamic, nil
	}

	return addressType, nil
}

func GetSubnetLease6WithoutReclaimed(subnetId uint64, ip string, subnetLeases []*resource.SubnetLease6) (*resource.SubnetLease6, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Lease(context.TODO(),
		&pbdhcpagent.GetSubnet6LeaseRequest{Id: subnetId, Address: ip})
	if err != nil {
		return nil, err
	}

	subnetLease6 := SubnetLease6FromPbLease6(resp.GetLease())
	for _, reclaimSubnetLease6 := range subnetLeases {
		if reclaimSubnetLease6.Equal(subnetLease6) {
			return nil, nil
		}
	}

	return subnetLease6, nil
}

func SubnetLease6FromPbLease6(lease *pbdhcpagent.DHCPLease6) *resource.SubnetLease6 {
	lease6 := &resource.SubnetLease6{
		Address:               lease.GetAddress(),
		AddressType:           resource.AddressTypeDynamic,
		PrefixLen:             lease.GetPrefixLen(),
		Duid:                  lease.GetDuid(),
		Iaid:                  lease.GetIaid(),
		HwAddress:             lease.GetHwAddress(),
		HwAddressType:         lease.GetHwAddressType(),
		HwAddressSource:       lease.GetHwAddressSource().String(),
		HwAddressOrganization: lease.GetHwAddressOrganization(),
		ValidLifetime:         lease.GetValidLifetime(),
		PreferredLifetime:     lease.GetPreferredLifetime(),
		Expire:                timeFromUinx(lease.GetExpire()),
		LeaseType:             lease.GetLeaseType(),
		Hostname:              lease.GetHostname(),
		Fingerprint:           lease.GetFingerprint(),
		VendorId:              lease.GetVendorId(),
		OperatingSystem:       lease.GetOperatingSystem(),
		ClientType:            lease.GetClientType(),
		LeaseState:            lease.GetLeaseState().String(),
	}

	lease6.SetID(lease.GetAddress())
	return lease6
}

func pbdhcpLease6FromSubnetLease6(lease6 *resource.SubnetLease6) *pbdhcp.Lease6 {
	if lease6 == nil {
		return nil
	}

	return &pbdhcp.Lease6{
		Address:               lease6.Address,
		PrefixLen:             lease6.PrefixLen,
		Duid:                  lease6.Duid,
		Iaid:                  lease6.Iaid,
		HwAddress:             lease6.HwAddress,
		HwAddressType:         lease6.HwAddressType,
		HwAddressSource:       lease6.HwAddressSource,
		HwAddressOrganization: lease6.HwAddressOrganization,
		ValidLifetime:         lease6.ValidLifetime,
		PreferredLifetime:     lease6.PreferredLifetime,
		Expire:                lease6.Expire,
		LeaseType:             lease6.LeaseType,
		Hostname:              lease6.Hostname,
		VendorId:              lease6.VendorId,
		OperatingSystem:       lease6.OperatingSystem,
		ClientType:            lease6.ClientType,
		LeaseState:            lease6.LeaseState,
	}
}

func (d *DHCPService) GetSubnets4AndLeases4WithIps(ips []string) (map[string]*pbdhcp.Ipv4Information, error) {
	subnets, err := getSubnets4WithIps(ips)
	if err != nil {
		return nil, err
	}

	ipv4Infos := make(map[string]*pbdhcp.Ipv4Information)
	for ip, subnet := range subnets {
		ipv4Infos[ip] = &pbdhcp.Ipv4Information{
			Address:     ip,
			AddressType: resource.AddressTypeExclusion.String(),
			Subnet:      subnet,
		}
	}

	var subnetLeases map[string]*resource.SubnetLease4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		subnetLeases, err = setIpv4InfosAddressTypeAndGetSubnetLeases(tx, ips, ipv4Infos)
		return
	}); err != nil {
		log.Warnf("get address type and reclaimed lease4s failed: %s", err.Error())
		return ipv4Infos, nil
	}

	if err := setSubnetLease4sWithoutReclaimed(ipv4Infos, subnetLeases); err != nil {
		log.Warnf("get subnet4 leases failed: %s", err.Error())
	}

	return ipv4Infos, nil
}

func setIpv4InfosAddressTypeAndGetSubnetLeases(tx restdb.Transaction, ips []string, ipv4Infos map[string]*pbdhcp.Ipv4Information) (map[string]*resource.SubnetLease4, error) {
	var subnetIds []string
	unfoundIps := make(map[string]struct{})
	for ip, ipv4Info := range ipv4Infos {
		subnetIds = append(subnetIds, ipv4Info.Subnet.Id)
		unfoundIps[ip] = struct{}{}
	}

	subnetIdsArgs := strings.Join(subnetIds, "','")
	ipsArgs := strings.Join(ips, "','")
	if err := setIpv4InfosAddressType(tx, subnetIdsArgs, ipsArgs, unfoundIps, ipv4Infos); err != nil {
		return nil, err
	}

	return getClaimedSubnetLease4s(tx, subnetIdsArgs, ipsArgs)
}

func setIpv4InfosAddressType(tx restdb.Transaction, subnetIdsArgs, ipsArgs string, unfoundIps map[string]struct{}, ipv4Infos map[string]*pbdhcp.Ipv4Information) error {
	var reservations []*resource.Reservation4
	if err := tx.FillEx(reservations,
		"select * from gr_reservation4 where ip_address in "+
			ipsArgs+"and subnet4 in ('"+subnetIdsArgs+"')"); err != nil {
		return fmt.Errorf("get reservations failed: %s", err.Error())
	}

	for _, reservation := range reservations {
		if ipv4Info, ok := ipv4Infos[reservation.IpAddress]; ok {
			ipv4Info.AddressType = resource.AddressTypeReservation.String()
			delete(unfoundIps, ipv4Info.Address)
		}
	}

	if len(unfoundIps) == 0 {
		return nil
	}

	var reservedPools []*resource.ReservedPool4
	if err := tx.FillEx(reservedPools,
		"select * from gr_reserved_pool4 where subnet4 in ('"+
			subnetIdsArgs+"')"); err != nil {
		return fmt.Errorf("get reserved pool4s failed: %s", err.Error())
	}

	for ip := range unfoundIps {
		for _, reservedPool := range reservedPools {
			if reservedPool.Contains(ip) {
				ipv4Infos[ip].AddressType = resource.AddressTypeReserve.String()
				delete(unfoundIps, ip)
				break
			}
		}
	}

	if len(unfoundIps) == 0 {
		return nil
	}

	var pools []*resource.Pool4
	if err := tx.FillEx(pools,
		"select * from gr_pool4 where subnet4 in ('"+
			subnetIdsArgs+"')"); err != nil {
		return fmt.Errorf("get pool4s failed: %s", err.Error())
	}

	for ip := range unfoundIps {
		for _, pool := range pools {
			if pool.Contains(ip) {
				ipv4Infos[ip].AddressType = resource.AddressTypeDynamic.String()
				delete(unfoundIps, ip)
				break
			}
		}
	}

	return nil
}

func getClaimedSubnetLease4s(tx restdb.Transaction, subnetIdsArgs, ipsArgs string) (map[string]*resource.SubnetLease4, error) {
	var subnetLeases []*resource.SubnetLease4
	if err := tx.FillEx(&subnetLeases,
		"select * from gr_subnet_lease4 where address in ('"+
			ipsArgs+" and subnet4 in ('"+subnetIdsArgs+"')"); err != nil {
		return nil, fmt.Errorf("get subnet reclaimed lease4s failed: %s", err.Error())
	}

	subnetLeasesMap := make(map[string]*resource.SubnetLease4)
	for _, subnetLease := range subnetLeases {
		subnetLeasesMap[subnetLease.Address] = subnetLease
	}

	return subnetLeasesMap, nil
}

func setSubnetLease4sWithoutReclaimed(ipv4Infos map[string]*pbdhcp.Ipv4Information, subnetLeases map[string]*resource.SubnetLease4) error {
	var reqs []*pbdhcpagent.GetSubnet4LeaseRequest
	for ip, info := range ipv4Infos {
		reqs = append(reqs, &pbdhcpagent.GetSubnet4LeaseRequest{
			Id:      info.Subnet.SubnetId,
			Address: ip,
		})
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4LeasesWithIps(context.TODO(),
		&pbdhcpagent.GetSubnet4LeasesWithIpsRequest{Addresses: reqs})
	if err != nil {
		return fmt.Errorf("get subnet4 leases failed: %s", err.Error())
	}

	for ip, lease4 := range resp.GetLeases() {
		subnetLease4 := SubnetLease4FromPbLease4(lease4)
		if subnetLease, ok := subnetLeases[ip]; ok && subnetLease.Equal(subnetLease4) {
			continue
		}

		ipv4Infos[ip].Lease = pbdhcpLease4FromSubnetLease4(subnetLease4)
	}

	return nil
}

func (d *DHCPService) GetSubnets6AndLeases6WithIps(ips []string) (map[string]*pbdhcp.Ipv6Information, error) {
	subnets, err := getSubnets6WithIps(ips)
	if err != nil {
		return nil, err
	}

	ipv6Infos := make(map[string]*pbdhcp.Ipv6Information)
	for ip, subnet := range subnets {
		ipv6Infos[ip] = &pbdhcp.Ipv6Information{
			Address:     ip,
			AddressType: resource.AddressTypeExclusion.String(),
			Subnet:      subnet,
		}
	}

	var subnetLeases map[string]*resource.SubnetLease6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		subnetLeases, err = setIpv6InfosAddressTypeAndGetSubnetLeases(tx, ips, ipv6Infos)
		return
	}); err != nil {
		log.Warnf("get address type and reclaimed lease6s failed: %s", err.Error())
		return ipv6Infos, nil
	}

	if err := setSubnetLease6sWithoutReclaimed(ipv6Infos, subnetLeases); err != nil {
		log.Warnf("get subnet6 leases failed: %s", err.Error())
	}

	return ipv6Infos, nil
}

func setIpv6InfosAddressTypeAndGetSubnetLeases(tx restdb.Transaction, ips []string, ipv6Infos map[string]*pbdhcp.Ipv6Information) (map[string]*resource.SubnetLease6, error) {
	var subnetIds []string
	unfoundIps := make(map[string]struct{})
	for ip, ipv6Info := range ipv6Infos {
		subnetIds = append(subnetIds, ipv6Info.Subnet.Id)
		unfoundIps[ip] = struct{}{}
	}

	subnetIdsArgs := strings.Join(subnetIds, "','")
	if err := setIpv6InfosAddressType(tx, subnetIdsArgs, unfoundIps, ipv6Infos); err != nil {
		return nil, err
	}

	return getClaimedSubnetLease6s(tx, subnetIdsArgs, strings.Join(ips, "','"))
}

func setIpv6InfosAddressType(tx restdb.Transaction, subnetIdsArgs string, unfoundIps map[string]struct{}, ipv6Infos map[string]*pbdhcp.Ipv6Information) error {
	var reservations []*resource.Reservation6
	if err := tx.FillEx(reservations,
		"select * from gr_reservation6 where subnet6 in ('"+
			subnetIdsArgs+"')"); err != nil {
		return fmt.Errorf("get reservations failed: %s", err.Error())
	}

	for ip := range unfoundIps {
		for _, reservation := range reservations {
			oldLen := len(unfoundIps)
			for _, ipaddress := range reservation.IpAddresses {
				if ipaddress == ip {
					ipv6Infos[ip].AddressType = resource.AddressTypeReservation.String()
					delete(unfoundIps, ip)
					break
				}
			}
			if len(unfoundIps) != oldLen {
				break
			}
		}
	}

	if len(unfoundIps) == 0 {
		return nil
	}

	var reservedPools []*resource.ReservedPool6
	if err := tx.FillEx(reservedPools,
		"select * from gr_reserved_pool6 where subnet6 in ('"+
			subnetIdsArgs+"')"); err != nil {
		return fmt.Errorf("get reserved pool6s failed: %s", err.Error())
	}

	for ip := range unfoundIps {
		for _, reservedPool := range reservedPools {
			if reservedPool.Contains(ip) {
				ipv6Infos[ip].AddressType = resource.AddressTypeReserve.String()
				delete(unfoundIps, ip)
				break
			}
		}
	}

	if len(unfoundIps) == 0 {
		return nil
	}

	var pools []*resource.Pool6
	if err := tx.FillEx(pools,
		"select * from gr_pool6 where subnet6 in ('"+
			subnetIdsArgs+"')"); err != nil {
		return fmt.Errorf("get pool6s failed: %s", err.Error())
	}

	for ip := range unfoundIps {
		for _, pool := range pools {
			if pool.Contains(ip) {
				ipv6Infos[ip].AddressType = resource.AddressTypeDynamic.String()
				delete(unfoundIps, ip)
				break
			}
		}
	}

	return nil
}

func getClaimedSubnetLease6s(tx restdb.Transaction, subnetIdsArgs, ipsArgs string) (map[string]*resource.SubnetLease6, error) {
	var subnetLeases []*resource.SubnetLease6
	if err := tx.FillEx(&subnetLeases,
		"select * from gr_subnet_lease6 where address in ('"+
			ipsArgs+" and subnet6 in ('"+subnetIdsArgs+"')"); err != nil {
		return nil, fmt.Errorf("get subnet reclaimed lease6s failed: %s", err.Error())
	}

	subnetLeasesMap := make(map[string]*resource.SubnetLease6)
	for _, subnetLease := range subnetLeases {
		subnetLeasesMap[subnetLease.Address] = subnetLease
	}

	return subnetLeasesMap, nil
}

func setSubnetLease6sWithoutReclaimed(ipv6Infos map[string]*pbdhcp.Ipv6Information, subnetLeases map[string]*resource.SubnetLease6) error {
	var reqs []*pbdhcpagent.GetSubnet6LeaseRequest
	for ip, info := range ipv6Infos {
		reqs = append(reqs, &pbdhcpagent.GetSubnet6LeaseRequest{
			Id:      info.Subnet.SubnetId,
			Address: ip,
		})
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6LeasesWithIps(context.TODO(),
		&pbdhcpagent.GetSubnet6LeasesWithIpsRequest{Addresses: reqs})
	if err != nil {
		return fmt.Errorf("get subnet6 leases failed: %s", err.Error())
	}

	for ip, lease6 := range resp.GetLeases() {
		subnetLease6 := SubnetLease6FromPbLease6(lease6)
		if subnetLease, ok := subnetLeases[ip]; ok && subnetLease.Equal(subnetLease6) {
			continue
		}

		ipv6Infos[ip].Lease = pbdhcpLease6FromSubnetLease6(subnetLease6)
	}

	return nil
}

//// V4
func (d *DHCPService) GetListWithAllSubnet4s() ([]*pbdhcp.DhcpSubnet4, error) {
	return getWithSubnet4s()
}

func getWithSubnet4s() ([]*pbdhcp.DhcpSubnet4, error) {
	listCtx := genGetSubnetsContext(resource.TableSubnet4)
	var subnets []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, listCtx.sql, listCtx.params...)
	}); err != nil {
		return nil, fmt.Errorf("list subnet4s from db failed: %s", err.Error())
	}
	if err := setSubnet4sLeasesUsedInfo(subnets, listCtx); err != nil {
		return nil, fmt.Errorf("set subnet4s leases used info failed: %s", err.Error())
	}
	if nodeNames, err := getNodeNames(true); err != nil {
		return nil, fmt.Errorf("get node names failed: %s", err.Error())
	} else {
		setSubnet4sNodeNames(subnets, nodeNames)
	}
	return pbDhcpSubnet4sFromSubnet4List(subnets)
}

func genGetSubnetsContext(table restdb.ResourceType) listSubnetContext {
	listCtx := listSubnetContext{}
	sqls := []string{"select * from gr_" + string(table)}
	if listCtx.hasFilterSubnet == false {
		sqls = append(sqls, "order by subnet_id")
	}
	listCtx.sql = strings.Join(sqls, " ")
	return listCtx
}

type AgentRole string

const (
	AgentRoleSentry4 AgentRole = "sentry4"
	AgentRoleSentry6 AgentRole = "sentry6"
)

func getNodeNames(isv4 bool) (map[string]string, error) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list dhcp agent4s failed: %s", err.Error())
	}

	sentryRole := AgentRoleSentry4
	if isv4 == false {
		sentryRole = AgentRoleSentry6
	}

	nodeNames := make(map[string]string)
	for _, node := range dhcpNodes.GetNodes() {
		if IsAgentService(node.GetServiceTags(), sentryRole) {
			if node.GetVirtualIp() != "" {
				return map[string]string{node.GetIpv4(): node.GetName()}, nil
			} else {
				nodeNames[node.GetIpv4()] = node.GetName()
			}
		}
	}

	return nodeNames, nil
}

func setSubnet4sNodeNames(subnets []*resource.Subnet4, nodeNames map[string]string) {
	for _, subnet := range subnets {
		subnet.NodeNames = getSubnetNodeNames(subnet.Nodes, nodeNames)
	}
}

func getSubnetNodeNames(nodes []string, nodeNames map[string]string) []string {
	var names []string
	for _, node := range nodes {
		if name, ok := nodeNames[node]; ok {
			names = append(names, name)
		}
	}
	return names
}

func IsAgentService(tags []string, role AgentRole) bool {
	for _, tag := range tags {
		if tag == string(role) {
			return true
		}
	}
	return false
}

func (d *DHCPService) GetListWithSubnet4sByPrefixes(prefixes []string) ([]*pbdhcp.DhcpSubnet4, error) {
	return getWithSubnet4sPrefixes(prefixes)
}

func getWithSubnet4sPrefixes(prefixes []string) ([]*pbdhcp.DhcpSubnet4, error) {

	for _, subnet := range prefixes {
		if _, err := gohelperip.ParseCIDRv4(subnet); err != nil {
			return nil, fmt.Errorf("action check subnet could be created input invalid: %s", err.Error())
		}
	}

	var subnets []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets,
			fmt.Sprintf("select * from gr_subnet4 where subnet in ('%s')",
				strings.Join(prefixes, "','")))
	}); err != nil {
		return nil, fmt.Errorf("action list subnet failed: %s", err.Error())
	}

	if err := setSubnet4sLeasesUsedInfo(subnets,
		listSubnetContext{hasFilterSubnet: true}); err != nil {
		return nil, fmt.Errorf("set subnet4s leases used info failed: %s", err.Error())
	}

	return pbDhcpSubnet4sFromSubnet4List(subnets)
}

func setSubnet4sLeasesUsedInfo(subnets []*resource.Subnet4, ctx listSubnetContext) error {
	if ctx.needSetSubnetsLeasesUsedInfo() == false || len(subnets) == 0 {
		return nil
	}

	var resp *pbdhcpagent.GetSubnetsLeasesCountResponse
	var err error
	if ctx.isUseIds() {
		var ids []uint64
		for _, subnet := range subnets {
			if subnet.Capacity != 0 {
				ids = append(ids, subnet.SubnetId)
			}
		}

		if len(ids) != 0 {
			resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets4LeasesCountWithIds(
				context.TODO(), &pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: ids})
		}
	} else {
		resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets4LeasesCount(
			context.TODO(), &pbdhcpagent.GetSubnetsLeasesCountRequest{})
	}

	if err != nil {
		return err
	}

	subnetsLeasesCount := resp.GetSubnetsLeasesCount()
	for _, subnet := range subnets {
		if subnet.Capacity != 0 {
			if leasesCount, ok := subnetsLeasesCount[subnet.SubnetId]; ok {
				subnet.UsedCount = leasesCount
				subnet.UsedRatio = fmt.Sprintf("%.4f",
					float64(leasesCount)/float64(subnet.Capacity))
			}
		}
	}

	return nil
}

type listSubnetContext struct {
	countSql        string
	sql             string
	params          []interface{}
	hasFilterSubnet bool
	hasPagination   bool
	hasExclude      bool
	hasShared       bool
}

func (l listSubnetContext) isUseIds() bool {
	return l.hasPagination || l.hasFilterSubnet
}

func (l listSubnetContext) needSetSubnetsLeasesUsedInfo() bool {
	return l.hasExclude == false && l.hasShared == false
}

func pbDhcpSubnet4sFromSubnet4List(subnetList []*resource.Subnet4) ([]*pbdhcp.DhcpSubnet4, error) {
	tmpList := make([]*pbdhcp.DhcpSubnet4, 0)
	for _, v := range subnetList {
		tmpList = append(tmpList, pbDhcpSubnet4sFromSubnet4(v))
	}
	return tmpList, nil
}

func pbDhcpSubnet4sFromSubnet4(subnet *resource.Subnet4) *pbdhcp.DhcpSubnet4 {
	return &pbdhcp.DhcpSubnet4{
		Id:                  subnet.ID,
		Type:                subnet.Type,
		CreationTimestamp:   EncodeIsoTime(subnet.GetCreationTimestamp()),
		DeletionTimestamp:   EncodeIsoTime(subnet.GetDeletionTimestamp()),
		Links:               EncodeLinks(subnet.GetLinks()),
		Subnet:              subnet.Subnet,
		IpNet:               subnet.Ipnet.String(),
		SubnetId:            subnet.SubnetId,
		ValidLifetime:       subnet.ValidLifetime,
		MaxValidLifetime:    subnet.MaxValidLifetime,
		MinValidLifetime:    subnet.MinValidLifetime,
		SubnetMask:          subnet.SubnetMask,
		DomainServers:       subnet.DomainServers,
		Routers:             subnet.Routers,
		ClientClass:         subnet.ClientClass,
		TftpServer:          subnet.TftpServer,
		BootFile:            subnet.Bootfile,
		RelayAgentAddresses: subnet.RelayAgentAddresses,
		IFaceName:           subnet.IfaceName,
		NextServer:          subnet.NextServer,
		Tags:                subnet.Tags,
		NodeNames:           subnet.NodeNames,
		Nodes:               subnet.Nodes,
		Capacity:            subnet.Capacity,
		UsedRatio:           subnet.UsedRatio,
		UseCount:            subnet.UsedCount,
	}
}

func (d DHCPService) GetListPool4BySubnet4Id(subnet4Id string) ([]*pbdhcp.DhcpPool4, error) {
	return getPool4BySubnet4Id(subnet4Id)
}

func getPool4BySubnet4Id(subnet4Id string) ([]*pbdhcp.DhcpPool4, error) {
	var subnet *resource.Subnet4
	var pools []*resource.Pool4
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if tmpSubnet, err := getSubnet4FromDB(tx, subnet4Id); err != nil {
			return err
		} else {
			subnet = tmpSubnet
		}
		if err := tx.Fill(map[string]interface{}{"subnet4": subnet.GetID(),
			"orderby": "begin_ip"}, &pools); err != nil {
			return err
		}

		return tx.FillEx(&reservations,
			"select * from gr_reservation4 where id in (select distinct r4.id from gr_reservation4 r4, "+
				"gr_pool4 p4 where r4.subnet4 = $1 and r4.subnet4 = p4.subnet4 and "+
				"r4.ip_address >= p4.begin_address and r4.ip_address <= p4.end_address)",
			subnet.GetID())
	}); err != nil {
		return nil, fmt.Errorf("list pools with subnet %s from db failed: %s",
			subnet.GetID(), err.Error())
	}

	poolsLeases, err := loadPool4sLeases(subnet, pools, reservations)
	if err != nil {
		return nil, err
	}
	for _, pool := range pools {
		setPool4LeasesUsedRatio(pool, poolsLeases[pool.GetID()])
	}
	return pdDhcpPool4FromPool4List(pools)
}

func getSubnet4FromDB(tx restdb.Transaction, subnetId string) (*resource.Subnet4, error) {
	var subnets []*resource.Subnet4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: subnetId},
		&subnets); err != nil {
		return nil, fmt.Errorf("get subnet %s from db failed: %s",
			subnetId, err.Error())
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("get subnet %s is nil", subnetId)
	}

	return subnets[0], nil
}

func loadPool4sLeases(subnet *resource.Subnet4, pools []*resource.Pool4, reservations []*resource.Reservation4) (map[string]uint64, error) {
	resp, err := getSubnet4Leases(subnet.SubnetId)
	if err != nil {
		return nil, fmt.Errorf("get subnet %s leases failed: %s", subnet.GetID(), err.Error())
	}

	if len(resp.GetLeases()) == 0 {
		return nil, nil
	}

	reservationMap := reservationMapFromReservation4s(reservations)
	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if _, ok := reservationMap[lease.GetAddress()]; ok {
			continue
		}

		for _, pool := range pools {
			if pool.Capacity != 0 && pool.Contains(lease.GetAddress()) {
				count := leasesCount[pool.GetID()]
				count += 1
				leasesCount[pool.GetID()] = count
			}
		}
	}

	return leasesCount, nil
}

func getSubnet4Leases(subnetId uint64) (*pbdhcpagent.GetLeases4Response, error) {
	return grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Leases(context.TODO(),
		&pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
}

func reservationMapFromReservation4s(reservations []*resource.Reservation4) map[string]string {
	reservationMap := make(map[string]string)
	for _, reservation := range reservations {
		reservationMap[reservation.IpAddress] = reservation.HwAddress
	}

	return reservationMap
}

func setPool4LeasesUsedRatio(pool *resource.Pool4, leasesCount uint64) {
	if leasesCount != 0 && pool.Capacity != 0 {
		pool.UsedCount = leasesCount
		pool.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(pool.Capacity))
	}
}

func pdDhcpPool4FromPool4List(pools []*resource.Pool4) ([]*pbdhcp.DhcpPool4, error) {
	ret := make([]*pbdhcp.DhcpPool4, 0)
	for _, v := range pools {
		ret = append(ret, pdDhcpPool4FromPool4t(v))
	}
	return ret, nil
}

func pdDhcpPool4FromPool4t(pool *resource.Pool4) *pbdhcp.DhcpPool4 {
	return &pbdhcp.DhcpPool4{
		Id:                pool.ID,
		Type:              pool.Type,
		CreationTimestamp: EncodeIsoTime(pool.GetCreationTimestamp()),
		DeletionTimestamp: EncodeIsoTime(pool.GetDeletionTimestamp()),
		Links:             EncodeLinks(pool.GetLinks()),
		Subnet4:           pool.Subnet4,
		BeginAddress:      pool.BeginAddress,
		BeginIp:           pool.BeginIp.String(),
		EndAddress:        pool.EndAddress,
		EndIp:             pool.EndIp.String(),
		Capacity:          pool.Capacity,
		UsedRatio:         pool.UsedRatio,
		UsedCount:         pool.UsedCount,
		Template:          pool.Template,
		Comment:           pool.Comment,
	}
}

func (d *DHCPService) GetListReservedPool4BySubnet4Id(subnet4Id string) ([]*pbdhcp.DhcpReservedPool4, error) {
	return getReservedPool4BySubnet4Id(subnet4Id)
}

func getReservedPool4BySubnet4Id(subnet4Id string) ([]*pbdhcp.DhcpReservedPool4, error) {
	var pools []*resource.ReservedPool4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{"subnet4": subnet4Id, "orderby": "begin_ip"},
			&pools)
	}); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("list reserved pools with subnet %s from db failed: %s",
			subnet4Id, err.Error()))
	}
	return pbDhcpReservedPool4FromReservedPool4(pools)
}

func pbDhcpReservedPool4FromReservedPool4(pools []*resource.ReservedPool4) ([]*pbdhcp.DhcpReservedPool4, error) {
	tmpPools := make([]*pbdhcp.DhcpReservedPool4, 0)
	for _, v := range pools {
		tmpPools = append(tmpPools, pbDhcpReservedPool4FromReservedPool4Info(v))
	}
	return tmpPools, nil
}

func pbDhcpReservedPool4FromReservedPool4Info(pool *resource.ReservedPool4) *pbdhcp.DhcpReservedPool4 {
	return &pbdhcp.DhcpReservedPool4{
		Id:                pool.ID,
		Type:              pool.Type,
		CreationTimestamp: EncodeIsoTime(pool.GetCreationTimestamp()),
		DeletionTimestamp: EncodeIsoTime(pool.GetDeletionTimestamp()),
		Links:             EncodeLinks(pool.GetLinks()),
		Subnet4:           pool.Subnet4,
		BeginAddress:      pool.BeginAddress,
		BeginIp:           pool.BeginIp.String(),
		EndAddress:        pool.EndAddress,
		EndIp:             pool.EndIp.String(),
		Capacity:          pool.Capacity,
		UsedRatio:         pool.UsedRatio,
		UsedCount:         pool.UsedCount,
		Template:          pool.Template,
		Comment:           pool.Comment,
	}
}

func (d *DHCPService) GetListReservation4BySubnet4Id(subnetId string) ([]*pbdhcp.DhcpReservation4, error) {
	return getReservation4BySubnet4Id(subnetId)
}

func getReservation4BySubnet4Id(subnetId string) ([]*pbdhcp.DhcpReservation4, error) {
	var reservations []*resource.Reservation4
	if err := db.GetResources(map[string]interface{}{
		"subnet4": subnetId, "orderby": "ip"}, &reservations); err != nil {
		return nil, fmt.Errorf("list reservations with subnet %s from db failed: %s",
			subnetId, err.Error())
	}

	leasesCount, err := getReservation4sLeasesCount(subnetIDStrToUint64(subnetId), reservations)
	if err != nil {
		return nil, err
	}
	for _, reservation := range reservations {
		setReservation4LeasesUsedRatio(reservation, leasesCount[reservation.IpAddress])
	}
	return pbDchpReservation4FormReservation4(reservations)
}

func getReservation4sLeasesCount(subnetId uint64, reservations []*resource.Reservation4) (map[string]uint64, error) {
	resp, err := getSubnet4Leases(subnetId)
	if err != nil {
		return nil, fmt.Errorf("get subnet %d leases failed: %s", subnetId, err.Error())
	}

	if len(resp.GetLeases()) == 0 {
		return nil, nil
	}

	reservationMap := reservationMapFromReservation4s(reservations)
	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if mac, ok := reservationMap[lease.GetAddress()]; ok &&
			mac == lease.GetHwAddress() {
			leasesCount[lease.GetAddress()] = 1
		}
	}

	return leasesCount, nil
}

func subnetIDStrToUint64(subnetID string) uint64 {
	id, _ := strconv.ParseUint(subnetID, 10, 64)
	return id
}

func setReservation4LeasesUsedRatio(reservation *resource.Reservation4, leasesCount uint64) {
	if leasesCount != 0 {
		reservation.UsedCount = leasesCount
		reservation.UsedRatio = fmt.Sprintf("%.4f",
			float64(leasesCount)/float64(reservation.Capacity))
	}
}

func pbDchpReservation4FormReservation4(reservations []*resource.Reservation4) ([]*pbdhcp.DhcpReservation4, error) {
	tmpList := make([]*pbdhcp.DhcpReservation4, 0)
	for _, v := range reservations {
		tmpList = append(tmpList, pbDhcpReservation4FormReservation4Info(v))
	}
	return tmpList, nil
}

func pbDhcpReservation4FormReservation4Info(old *resource.Reservation4) *pbdhcp.DhcpReservation4 {
	return &pbdhcp.DhcpReservation4{
		Id:                old.ID,
		Type:              old.Type,
		CreationTimestamp: EncodeIsoTime(old.GetCreationTimestamp()),
		DeletionTimestamp: EncodeIsoTime(old.GetDeletionTimestamp()),
		Links:             EncodeLinks(old.GetLinks()),
		Subnet4:           old.Subnet4,
		HwAddress:         old.HwAddress,
		IpAddress:         old.IpAddress,
		Ip:                old.Ip.String(),
		UsedRatio:         old.UsedRatio,
		UsedCount:         old.UsedCount,
		Capacity:          old.Capacity,
		Comment:           old.Comment,
	}
}

var ErrorIpNotBelongToSubnet = fmt.Errorf("ip not belongs to subnet")

func (d DHCPService) GetListLease4BySubnet4Id(subnetId string) ([]*pbdhcp.DhcpSubnetLease4, error) {
	return getLease4BySubnet4Id(subnetId)
}

func getLease4BySubnet4Id(subnetId string) ([]*pbdhcp.DhcpSubnetLease4, error) {
	var subnet4SubnetId uint64
	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, subnetId)
		if err != nil {
			return err
		}
		subnet4SubnetId = subnet4.SubnetId
		reservations, subnetLeases, err = getReservation4sAndSubnetLease4s(
			tx, subnetId)
		return err
	}); err != nil {
		if err == ErrorIpNotBelongToSubnet {
			return nil, nil
		} else {
			return nil, fmt.Errorf("get subnet4 %s from db failed: %s", subnetId, err.Error())
		}
	}
	if subnetLeasesList, err := getSubnetLease4s(subnet4SubnetId, reservations, subnetLeases); err != nil {
		return nil, err
	} else {
		return pbDhcpSubnetLeasesFromSubnetLeases(subnetLeasesList)
	}
}

func getReservation4sAndSubnetLease4s(tx restdb.Transaction, subnetId string) ([]*resource.Reservation4, []*resource.SubnetLease4, error) {
	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := tx.Fill(map[string]interface{}{"subnet4": subnetId},
		&reservations); err != nil {
		return nil, nil, fmt.Errorf("get reservation4s failed: %s", err.Error())
	}

	if err := tx.Fill(map[string]interface{}{"subnet4": subnetId},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease4s failed: %s", err.Error())
	}

	return reservations, subnetLeases, nil
}

func getSubnetLease4s(subnetId uint64, reservations []*resource.Reservation4, subnetLeases []*resource.SubnetLease4) ([]*resource.SubnetLease4, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Leases(context.TODO(),
		&pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
	if err != nil {
		log.Warnf("get subnet4 %d leases failed: %s", subnetId, err.Error())
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
		return leases, nil
	}

	return leases, nil
}

func subnetLease4FromPbLease4AndReservations(lease *pbdhcpagent.DHCPLease4, reservationMap map[string]string) *resource.SubnetLease4 {
	subnetLease4 := SubnetLease4FromPbLease4(lease)
	if _, ok := reservationMap[subnetLease4.Address]; ok {
		subnetLease4.AddressType = resource.AddressTypeReservation
	}
	return subnetLease4
}

func pbDhcpSubnetLeasesFromSubnetLeases(subnetLeases []*resource.SubnetLease4) ([]*pbdhcp.DhcpSubnetLease4, error) {
	tmpList := make([]*pbdhcp.DhcpSubnetLease4, 0)
	for _, v := range subnetLeases {
		tmpList = append(tmpList, pbDhcpSubnetLeasesFromSubnetLeasesInfo(v))
	}
	return tmpList, nil
}

func pbDhcpSubnetLeasesFromSubnetLeasesInfo(old *resource.SubnetLease4) *pbdhcp.DhcpSubnetLease4 {
	return &pbdhcp.DhcpSubnetLease4{
		Id:                    old.ID,
		Type:                  old.Type,
		CreationTimestamp:     EncodeIsoTime(old.GetCreationTimestamp()),
		DeletionTimestamp:     EncodeIsoTime(old.GetDeletionTimestamp()),
		Links:                 EncodeLinks(old.GetLinks()),
		Subnet4:               old.Subnet4,
		Address:               old.Address,
		AddressType:           old.AddressType.String(),
		HwAddress:             old.HwAddress,
		HwAddressOrganization: old.HwAddressOrganization,
		ClientId:              old.ClientId,
		ValidLifetime:         old.ValidLifetime,
		Expire:                old.Expire,
		Hostname:              old.Hostname,
		Fingerprint:           old.Fingerprint,
		VendorId:              old.VendorId,
		OperatingSystem:       old.OperatingSystem,
		ClientType:            old.ClientType,
		LeaseState:            old.LeaseState,
	}
}

func (d *DHCPService) GetListLease4ByIp(subnet4Id, ip string) (*pbdhcp.DhcpSubnetLease4, error) {

	if _, err := gohelperip.ParseIPv4(ip); err != nil {
		return nil, nil
	}
	var subnet4SubnetId uint64
	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet4, err := getSubnet4FromDB(tx, subnet4Id)
		if err != nil {
			return err
		}
		subnet4SubnetId = subnet4.SubnetId
		reservations, subnetLeases, err = getReservation4sAndSubnetLease4sWithIp(
			tx, subnet4, ip)
		return err
	}); err != nil {
		if err == ErrorIpNotBelongToSubnet {
			return nil, nil
		} else {
			return nil, fmt.Errorf(fmt.Sprintf("get subnet4 %s from db failed: %s", subnet4Id, err.Error()))
		}
	}
	if tmpList, err := getSubnetLease4sWithIp(subnet4SubnetId, ip, reservations, subnetLeases); err != nil {
		return nil, nil
	} else {
		if len(tmpList) == 0 {
			log.Debugf("GetListLease4ByIp is nil")
			return nil, nil
		}
		return pbDhcpSubnetLeasesFromSubnetLeasesInfo(tmpList[0]), nil
	}
}

func getReservation4sAndSubnetLease4sWithIp(tx restdb.Transaction, subnet4 *resource.Subnet4, ip string) ([]*resource.Reservation4, []*resource.SubnetLease4, error) {
	if subnet4.Ipnet.Contains(net.ParseIP(ip)) == false {
		return nil, nil, ErrorIpNotBelongToSubnet
	}

	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := tx.Fill(map[string]interface{}{"ip_address": ip, "subnet4": subnet4.GetID()},
		&reservations); err != nil {
		return nil, nil, fmt.Errorf("get reservation4 %s failed: %s", ip, err.Error())
	}

	if err := tx.Fill(map[string]interface{}{"address": ip, "subnet4": subnet4.GetID()},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease4 %s failed: %s", ip, err.Error())
	}

	return reservations, subnetLeases, nil
}

func getSubnetLease4sWithIp(subnetId uint64, ip string, reservations []*resource.Reservation4, subnetLeases []*resource.SubnetLease4) ([]*resource.SubnetLease4, error) {
	lease4, err := GetSubnetLease4WithoutReclaimed(subnetId, ip,
		subnetLeases)
	if err != nil {
		return nil, err
	} else if lease4 == nil {
		return nil, fmt.Errorf("get subnetLease4 ret lease4 is nil")
	}

	for _, reservation := range reservations {
		if reservation.IpAddress == lease4.Address {
			lease4.AddressType = resource.AddressTypeReservation
			break
		}
	}

	return []*resource.SubnetLease4{lease4}, nil
}

//// V6
func (d *DHCPService) GetListSubnet6All() ([]*pbdhcp.DhcpSubnet6, error) {
	return getSubnet6All()
}

func getSubnet6All() ([]*pbdhcp.DhcpSubnet6, error) {
	listCtx := genGetSubnetsContext(resource.TableSubnet6)
	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets, listCtx.sql, listCtx.params...)
	}); err != nil {
		return nil, fmt.Errorf("list subnet6s from db failed: %s", err.Error())
	}

	if err := setSubnet6sLeasesUsedInfo(subnets, listCtx.isUseIds()); err != nil {
		return nil, fmt.Errorf("set subnet6s leases used info failed: %s", err.Error())
	}

	if nodeNames, err := getNodeNames(false); err != nil {
		return nil, fmt.Errorf("get node names failed: %s", err.Error())
	} else {
		setSubnet6sNodeNames(subnets, nodeNames)
	}
	return pbDhcpSubnet6FromSubnet6(subnets)
}

func setSubnet6sLeasesUsedInfo(subnets []*resource.Subnet6, useIds bool) error {
	if len(subnets) == 0 {
		return nil
	}

	var resp *pbdhcpagent.GetSubnetsLeasesCountResponse
	var err error
	if useIds {
		var ids []uint64
		for _, subnet := range subnets {
			if subnet.Capacity != 0 {
				ids = append(ids, subnet.SubnetId)
			}
		}

		if len(ids) != 0 {
			resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets6LeasesCountWithIds(
				context.TODO(), &pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: ids})
		}
	} else {
		resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets6LeasesCount(
			context.TODO(), &pbdhcpagent.GetSubnetsLeasesCountRequest{})
	}

	if err != nil {
		return err
	}

	subnetsLeasesCount := resp.GetSubnetsLeasesCount()
	for _, subnet := range subnets {
		if subnet.Capacity != 0 {
			if leasesCount, ok := subnetsLeasesCount[subnet.SubnetId]; ok {
				subnet.UsedCount = leasesCount
				subnet.UsedRatio = fmt.Sprintf("%.4f",
					float64(leasesCount)/float64(subnet.Capacity))
			}
		}
	}

	return nil
}

func setSubnet6sNodeNames(subnets []*resource.Subnet6, nodeNames map[string]string) {
	for _, subnet := range subnets {
		subnet.NodeNames = getSubnetNodeNames(subnet.Nodes, nodeNames)
	}
}

func pbDhcpSubnet6FromSubnet6(subnets []*resource.Subnet6) ([]*pbdhcp.DhcpSubnet6, error) {
	tmpList := make([]*pbdhcp.DhcpSubnet6, 0)
	for _, v := range subnets {
		tmpList = append(tmpList, pbDhcpSubnet6FromSubnet6Info(v))
	}
	return tmpList, nil
}

func pbDhcpSubnet6FromSubnet6Info(old *resource.Subnet6) *pbdhcp.DhcpSubnet6 {
	return &pbdhcp.DhcpSubnet6{
		Id:                    old.ID,
		Type:                  old.Type,
		CreationTimestamp:     EncodeIsoTime(old.GetCreationTimestamp()),
		DeletionTimestamp:     EncodeIsoTime(old.GetDeletionTimestamp()),
		Links:                 EncodeLinks(old.GetLinks()),
		Subnet:                old.Subnet,
		IpNet:                 old.Ipnet.String(),
		SubnetId:              old.SubnetId,
		ValidLifetime:         old.ValidLifetime,
		MaxValidLifetime:      old.MaxValidLifetime,
		MinValidLifetime:      old.MinValidLifetime,
		PreferredLifetime:     old.PreferredLifetime,
		DomainServers:         old.DomainServers,
		ClientClass:           old.ClientClass,
		IFaceName:             old.IfaceName,
		RelayAgentAddresses:   old.RelayAgentAddresses,
		RelayAgentInterfaceId: old.RelayAgentInterfaceId,
		Tags:                  old.Tags,
		NodeNames:             old.NodeNames,
		Nodes:                 old.Nodes,
		RapidCommit:           old.RapidCommit,
		UseEui64:              old.UseEui64,
		Capacity:              old.Capacity,
		UsedRatio:             old.UsedRatio,
		UsedCount:             old.UsedCount,
	}
}

func (d *DHCPService) GetListSubnet6ByPrefixes(prefixes []string) ([]*pbdhcp.DhcpSubnet6, error) {
	return getSubnet6ByPrefixes(prefixes)
}

func getSubnet6ByPrefixes(prefixes []string) ([]*pbdhcp.DhcpSubnet6, error) {
	for _, subnet := range prefixes {
		if _, err := gohelperip.ParseCIDRv6(subnet); err != nil {
			return nil, fmt.Errorf("action check subnet could be created input subnet %s invalid: %s", subnet, err.Error())
		}
	}

	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.FillEx(&subnets,
			fmt.Sprintf("select * from gr_subnet6 where subnet in ('%s')",
				strings.Join(prefixes, "','")))
	}); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("action list subnet failed: %s", err.Error()))
	}

	if err := setSubnet6sLeasesUsedInfo(subnets, true); err != nil {
		return nil, fmt.Errorf("set subnet6s leases used info failed: %s", err.Error())
	}
	return pbDhcpFromSubnet6s(subnets)
}

func pbDhcpFromSubnet6s(subnets []*resource.Subnet6) ([]*pbdhcp.DhcpSubnet6, error) {
	tmpList := make([]*pbdhcp.DhcpSubnet6, 0)
	for _, v := range subnets {
		tmpList = append(tmpList, pbDhcpSubnet6sFromSubnet6sInfo(v))
	}
	return tmpList, nil
}

func pbDhcpSubnet6sFromSubnet6sInfo(old *resource.Subnet6) *pbdhcp.DhcpSubnet6 {
	return &pbdhcp.DhcpSubnet6{
		Id:                    old.ID,
		Type:                  old.Type,
		CreationTimestamp:     EncodeIsoTime(old.GetCreationTimestamp()),
		DeletionTimestamp:     EncodeIsoTime(old.GetDeletionTimestamp()),
		Links:                 EncodeLinks(old.GetLinks()),
		Subnet:                old.Subnet,
		IpNet:                 old.Ipnet.String(),
		SubnetId:              old.SubnetId,
		ValidLifetime:         old.ValidLifetime,
		MaxValidLifetime:      old.MaxValidLifetime,
		MinValidLifetime:      old.MinValidLifetime,
		PreferredLifetime:     old.PreferredLifetime,
		DomainServers:         old.DomainServers,
		ClientClass:           old.ClientClass,
		IFaceName:             old.IfaceName,
		RelayAgentAddresses:   old.RelayAgentAddresses,
		RelayAgentInterfaceId: old.RelayAgentInterfaceId,
		Tags:                  old.Tags,
		NodeNames:             old.NodeNames,
		Nodes:                 old.Nodes,
		RapidCommit:           old.RapidCommit,
		UseEui64:              old.UseEui64,
		Capacity:              old.Capacity,
		UsedRatio:             old.UsedRatio,
		UsedCount:             old.UsedCount,
	}
}

func (d *DHCPService) GetListPool6BySubnet6Id(subnetId string) ([]*pbdhcp.DhcpPool6, error) {
	return getPool6BySubnet6Id(subnetId)
}

func getPool6BySubnet6Id(subnetId string) ([]*pbdhcp.DhcpPool6, error) {
	var subnet *resource.Subnet6
	var pools []*resource.Pool6
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if tmpSubnet, err := getSubnet6FromDB(tx, subnetId); err != nil {
			return err
		} else {
			subnet = tmpSubnet
		}

		if err := tx.Fill(map[string]interface{}{"subnet6": subnet.GetID(),
			"orderby": "begin_ip"}, &pools); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{"subnet6": subnet.GetID()}, &reservations)
	}); err != nil {
		return nil, fmt.Errorf("list pools with subnet %s from db failed: %s",
			subnet.GetID(), err.Error())
	}

	poolsLeases, err := loadPool6sLeases(subnet, pools, reservations)
	if err != nil {
		return nil, err
	}
	for _, pool := range pools {
		setPool6LeasesUsedRatio(pool, poolsLeases[pool.GetID()])
	}
	return pbDhcpPoolsFromPools(pools)
}

func getSubnet6FromDB(tx restdb.Transaction, subnetId string) (*resource.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: subnetId},
		&subnets); err != nil {
		return nil, fmt.Errorf("get subnet %s from db failed: %s", subnetId, err.Error())
	}

	if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet %s", subnetId)
	}

	return subnets[0], nil
}

func loadPool6sLeases(subnet *resource.Subnet6, pools []*resource.Pool6, reservations []*resource.Reservation6) (map[string]uint64, error) {
	resp, err := getSubnet6Leases(subnet.SubnetId)
	if err != nil {
		return nil, fmt.Errorf("get subnet %s leases failed: %s", subnet.GetID(), err.Error())
	}

	if len(resp.GetLeases()) == 0 {
		return nil, nil
	}

	reservationMap := reservationMapFromReservation6s(reservations)
	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if _, ok := reservationMap[lease.GetAddress()]; ok {
			continue
		}

		for _, pool := range pools {
			if pool.Capacity != 0 && pool.Contains(lease.GetAddress()) {
				count := leasesCount[pool.GetID()]
				count += 1
				leasesCount[pool.GetID()] = count
			}
		}
	}

	return leasesCount, nil
}

func getSubnet6Leases(subnetId uint64) (*pbdhcpagent.GetLeases6Response, error) {
	return grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Leases(context.TODO(),
		&pbdhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
}

func setPool6LeasesUsedRatio(pool *resource.Pool6, leasesCount uint64) {
	if leasesCount != 0 && pool.Capacity != 0 {
		pool.UsedCount = leasesCount
		pool.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(pool.Capacity))
	}
}

func pbDhcpPoolsFromPools(pools []*resource.Pool6) ([]*pbdhcp.DhcpPool6, error) {
	tmpList := make([]*pbdhcp.DhcpPool6, 0)
	for _, v := range pools {
		tmpList = append(tmpList, pbDhcpPoolsFromPoolsInfo(v))
	}
	return tmpList, nil
}

func pbDhcpPoolsFromPoolsInfo(old *resource.Pool6) *pbdhcp.DhcpPool6 {
	return &pbdhcp.DhcpPool6{
		Id:                old.ID,
		Type:              old.Type,
		CreationTimestamp: EncodeIsoTime(old.GetCreationTimestamp()),
		DeletionTimestamp: EncodeIsoTime(old.GetDeletionTimestamp()),
		Links:             EncodeLinks(old.GetLinks()),
		Subnet6:           old.Subnet6,
		BeginAddress:      old.BeginAddress,
		BeginIp:           old.BeginIp.String(),
		EndAddress:        old.EndAddress,
		EndIp:             old.EndIp.String(),
		Capacity:          old.Capacity,
		UsedRatio:         old.UsedRatio,
		UsedCount:         old.UsedCount,
		Template:          old.Template,
		Comment:           old.Comment,
	}
}

func (d *DHCPService) GetListReservedPool6BySubnet6Id(subnetId string) ([]*pbdhcp.DhcpReservedPool6, error) {
	return getReservedPool6BySubnet6Id(subnetId)
}

func getReservedPool6BySubnet6Id(subnetId string) ([]*pbdhcp.DhcpReservedPool6, error) {
	var subnet *resource.Subnet6
	var pools []*resource.ReservedPool6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if subnet6, err := getSubnet6FromDB(tx, subnetId); err != nil {
			return err
		} else {
			subnet = subnet6
		}
		return tx.Fill(map[string]interface{}{
			"subnet6": subnet.GetID(), "orderby": "begin_ip"}, &pools)
	}); err != nil {
		return nil, fmt.Errorf("list pools with subnet %s from db failed: %s",
			subnet.GetID(), err.Error())
	}
	return pbDhcpReservedPool6FromReservedPool6(pools)
}

func pbDhcpReservedPool6FromReservedPool6(pools []*resource.ReservedPool6) ([]*pbdhcp.DhcpReservedPool6, error) {
	tmpList := make([]*pbdhcp.DhcpReservedPool6, 0)
	for _, v := range pools {
		tmpList = append(tmpList, fillPbDhcpReservedPool6Info(v))
	}
	return tmpList, nil
}

func fillPbDhcpReservedPool6Info(old *resource.ReservedPool6) *pbdhcp.DhcpReservedPool6 {
	return &pbdhcp.DhcpReservedPool6{
		Id:                old.ID,
		Type:              old.Type,
		CreationTimestamp: EncodeIsoTime(old.GetCreationTimestamp()),
		DeletionTimestamp: EncodeIsoTime(old.GetDeletionTimestamp()),
		Links:             EncodeLinks(old.GetLinks()),
		Subnet6:           old.Subnet6,
		BeginAddress:      old.BeginAddress,
		BeginIp:           old.BeginIp.String(),
		EndAddress:        old.EndAddress,
		EndIp:             old.EndIp.String(),
		Capacity:          old.Capacity,
		UsedRatio:         old.UsedRatio,
		UsedCount:         old.UsedCount,
		Template:          old.Template,
		Comment:           old.Comment,
	}
}

func (d *DHCPService) GetListReservation6BySubnet6Id(subnetId string) ([]*pbdhcp.DhcpReservation6, error) {
	return getReservation6BySubnet6Id(subnetId)
}

func getReservation6BySubnet6Id(subnetId string) ([]*pbdhcp.DhcpReservation6, error) {
	var reservations []*resource.Reservation6
	if err := db.GetResources(map[string]interface{}{
		"subnet6": subnetId, "orderby": "duid, hw_address"}, &reservations); err != nil {
		return nil, fmt.Errorf("list reservations with subnet %s from db failed: %s",
			subnetId, err.Error())
	}

	leasesCount, err := getReservation6sLeasesCount(subnetIDStrToUint64(subnetId), reservations)
	if err != nil {
		return nil, err
	}
	for _, reservation := range reservations {
		setReservation6LeasesUsedRatio(reservation, leasesCount[reservation.GetID()])
	}
	return pbDhcpReservation6FromReservation6(reservations)
}

func getReservation6sLeasesCount(subnetId uint64, reservations []*resource.Reservation6) (map[string]uint64, error) {
	resp, err := getSubnet6Leases(subnetId)
	if err != nil {
		return nil, fmt.Errorf("get subnet %d leases failed: %s", subnetId, err.Error())
	}

	if len(resp.GetLeases()) == 0 {
		return nil, nil
	}

	reservationMap := make(map[string]*resource.Reservation6)
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress] = reservation
		}
	}

	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if reservation, ok := reservationMap[lease.GetAddress()]; ok {
			count := leasesCount[reservation.GetID()]
			if (len(reservation.Duid) != 0 && reservation.Duid == lease.GetDuid()) ||
				(len(reservation.HwAddress) != 0 &&
					reservation.HwAddress == lease.GetHwAddress()) {
				count += 1
			}
			leasesCount[reservation.GetID()] = count
		}
	}

	return leasesCount, nil
}

func reservationMapFromReservation6s(reservations []*resource.Reservation6) map[string]struct{} {
	reservationMap := make(map[string]struct{})
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress] = struct{}{}
		}
	}

	return reservationMap
}

func setReservation6LeasesUsedRatio(reservation *resource.Reservation6, leasesCount uint64) {
	if leasesCount != 0 {
		reservation.UsedCount = leasesCount
		reservation.UsedRatio = fmt.Sprintf("%.4f",
			float64(leasesCount)/float64(reservation.Capacity))
	}
}

func pbDhcpReservation6FromReservation6(reservations []*resource.Reservation6) ([]*pbdhcp.DhcpReservation6, error) {
	tmpList := make([]*pbdhcp.DhcpReservation6, 0)
	for _, v := range reservations {
		tmpList = append(tmpList, fillPbDhcpReservation6Info(v))
	}
	return tmpList, nil
}

func fillPbDhcpReservation6Info(old *resource.Reservation6) *pbdhcp.DhcpReservation6 {
	return &pbdhcp.DhcpReservation6{
		Id:                old.ID,
		Type:              old.Type,
		CreationTimestamp: EncodeIsoTime(old.GetCreationTimestamp()),
		DeletionTimestamp: EncodeIsoTime(old.GetDeletionTimestamp()),
		Links:             EncodeLinks(old.GetLinks()),
		Subnet6:           old.Subnet6,
		DUid:              old.Duid,
		HwAddress:         old.HwAddress,
		IpAddresses:       old.IpAddresses,
		Ips:               ipListToStrList(old.Ips),
		Prefixes:          old.Prefixes,
		Capacity:          old.Capacity,
		UsedRatio:         old.UsedRatio,
		UsedCount:         old.UsedCount,
		Comment:           old.Comment,
	}
}

func ipListToStrList(ips []net.IP) []string {
	l := make([]string, 0)
	for _, v := range ips {
		l = append(l, v.String())
	}
	return l
}

func (d *DHCPService) GetListLease6BySubnet6Id(subnet6Id string) ([]*pbdhcp.DhcpSubnetLease6, error) {
	return getLease6BySubnet6Id(subnet6Id)
}

func getLease6BySubnet6Id(subnetId string) ([]*pbdhcp.DhcpSubnetLease6, error) {
	var subnet6SubnetId uint64
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnetId)
		if err != nil {
			return err
		}
		subnet6SubnetId = subnet6.SubnetId
		reservations, subnetLeases, err = getReservation6sAndSubnetLease6s(tx, subnetId)

		return err
	}); err != nil {
		if err == ErrorIpNotBelongToSubnet {
			return nil, nil
		} else {
			return nil, fmt.Errorf("get subnet6 %s from db failed: %s", subnetId, err.Error())
		}
	}
	if subnetLeasesList, err := getSubnetLease6s(subnet6SubnetId, reservations, subnetLeases); err != nil {
		return nil, nil
	} else {
		return dpDhcpSubnetLease6FromSubnetLease6(subnetLeasesList)
	}
}

func getSubnetLease6s(subnetId uint64, reservations []*resource.Reservation6, subnetLeases []*resource.SubnetLease6) ([]*resource.SubnetLease6, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Leases(context.TODO(),
		&pbdhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
	if err != nil {
		return nil, fmt.Errorf("get subnet6 %d leases failed: %s", subnetId, err.Error())
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
		return nil, fmt.Errorf("delete reclaim leases failed: %s", err.Error())
	}
	return leases, nil
}

func subnetLease6FromPbLease6AndReservations(lease *pbdhcpagent.DHCPLease6, reservationMap map[string]struct{}) *resource.SubnetLease6 {
	subnetLease6 := SubnetLease6FromPbLease6(lease)
	if _, ok := reservationMap[subnetLease6.Address]; ok {
		subnetLease6.AddressType = resource.AddressTypeReservation
	}
	return subnetLease6
}

func dpDhcpSubnetLease6FromSubnetLease6(subnetLeases []*resource.SubnetLease6) ([]*pbdhcp.DhcpSubnetLease6, error) {
	tmpList := make([]*pbdhcp.DhcpSubnetLease6, 0)
	for _, v := range subnetLeases {
		tmpList = append(tmpList, fillDpDhcpSubnetLease6Info(v))
	}
	return tmpList, nil
}

func fillDpDhcpSubnetLease6Info(old *resource.SubnetLease6) *pbdhcp.DhcpSubnetLease6 {
	return &pbdhcp.DhcpSubnetLease6{
		Id:                    old.ID,
		Type:                  old.Type,
		CreationTimestamp:     EncodeIsoTime(old.GetCreationTimestamp()),
		DeletionTimestamp:     EncodeIsoTime(old.GetDeletionTimestamp()),
		Links:                 EncodeLinks(old.GetLinks()),
		Subnet6:               old.Subnet6,
		Address:               old.Address,
		AddressType:           old.AddressType.String(),
		PrefixLen:             old.PrefixLen,
		DUid:                  old.Duid,
		IAid:                  old.Iaid,
		PreferredLifetime:     old.PreferredLifetime,
		ValidLifetime:         old.ValidLifetime,
		Expire:                old.Expire,
		HwAddress:             old.HwAddress,
		HwAddressType:         old.HwAddressType,
		HwAddressSource:       old.HwAddressSource,
		HwAddressOrganization: old.HwAddressOrganization,
		LeaseType:             old.LeaseType,
		Hostname:              old.Hostname,
		Fingerprint:           old.Fingerprint,
		VendorId:              old.VendorId,
		OperatingSystem:       old.OperatingSystem,
		ClientType:            old.ClientType,
		LeaseState:            old.LeaseState,
	}
}

func (d *DHCPService) GetListLease6ByIp(subnetId, ip6 string) (*pbdhcp.DhcpSubnetLease6, error) {
	var subnet6SubnetId uint64
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnetId)
		if err != nil {
			return err
		}
		subnet6SubnetId = subnet6.SubnetId
		reservations, subnetLeases, err = getReservation6sAndSubnetLease6sWithIp(
			tx, subnet6, ip6)

		return err
	}); err != nil {
		if err == ErrorIpNotBelongToSubnet {
			return nil, nil
		} else {
			return nil, fmt.Errorf("get subnet6 %s from db failed: %s", subnetId, err.Error())
		}
	}
	if tmpList, err := getSubnetLease6sWithIp(subnet6SubnetId, ip6, reservations, subnetLeases); err != nil {
		return nil, nil
	} else {
		if len(tmpList) == 0 {
			log.Debugf("GetListLease4ByIp is nil")
			return nil, nil
		}
		return fillDpDhcpSubnetLease6Info(tmpList[0]), nil
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

	if err := tx.Fill(map[string]interface{}{"address": ip, "subnet6": subnet6.GetID()},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease6 %s failed: %s", ip, err.Error())
	}

	return reservations, subnetLeases, nil
}

func getReservation6sAndSubnetLease6s(tx restdb.Transaction, subnetId string) ([]*resource.Reservation6, []*resource.SubnetLease6, error) {
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetId},
		&reservations); err != nil {
		return nil, nil, fmt.Errorf("get reservation6s failed: %s", err.Error())
	}

	if err := tx.Fill(map[string]interface{}{"subnet6": subnetId},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease6s failed: %s", err.Error())
	}

	return reservations, subnetLeases, nil
}

func getSubnetLease6sWithIp(subnetId uint64, ip string, reservations []*resource.Reservation6, subnetLeases []*resource.SubnetLease6) ([]*resource.SubnetLease6, error) {
	lease6, err := GetSubnetLease6WithoutReclaimed(subnetId, ip,
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

func EncodeLinks(old map[restresource.ResourceLinkType]restresource.ResourceLink) map[string]string {
	links := make(map[string]string)
	for keyStr, valStr := range old {
		links[string(keyStr)] = string(valStr)
	}
	return links
}

func EncodeIsoTime(old time.Time) string {
	return old.Format(time.RFC3339)
}
