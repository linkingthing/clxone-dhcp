package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/grpc/parser"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
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
	err = tx.Fill(map[string]interface{}{resource.SqlColumnAddress: ip, resource.SqlColumnSubnet4: subnetId}, &subnetLeases)
	return addressType, subnetLeases, err
}

func GetIPv4AddressType(tx restdb.Transaction, subnetId, ip string) (resource.AddressType, error) {
	addressType := resource.AddressTypeExclusion
	if exists, err := tx.Exists(resource.TableReservation4,
		map[string]interface{}{resource.SqlColumnIpAddress: ip, resource.SqlColumnSubnet4: subnetId}); err != nil {
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
	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnAddress: ip,
		resource.SqlColumnSubnet6: subnetId},
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

////
func (d *DHCPService) GetAllSubnet4s() ([]*pbdhcp.DhcpSubnet4, error) {
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

func (d *DHCPService) GetWithSubnet4sByPrefixes(prefixes []string) ([]*pbdhcp.DhcpSubnet4, error) {
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

func (d DHCPService) GetWithPool4BySubnet4Id(subnet4Id string) ([]*pbdhcp.DhcpPool4, error) {
	var subnet *resource.Subnet4
	var pools []*resource.Pool4
	var reservations []*resource.Reservation4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if tmpSubnet, err := getSubnet4FromDB(tx, subnet4Id); err != nil {
			return err
		} else {
			subnet = tmpSubnet
		}
		if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnet.GetID(),
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

func (d *DHCPService) GetReservedPool4BySubnet4Id(subnet4Id string) ([]*pbdhcp.DhcpReservedPool4, error) {
	var pools []*resource.ReservedPool4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnet4Id, "orderby": "begin_ip"},
			&pools)
	}); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("list reserved pools with subnet %s from db failed: %s",
			subnet4Id, err.Error()))
	}
	return pbDhcpReservedPool4FromReservedPool4(pools)
}

func (d *DHCPService) GetReservation4BySubnet4Id(subnetId string) ([]*pbdhcp.DhcpReservation4, error) {
	var reservations []*resource.Reservation4
	if err := db.GetResources(map[string]interface{}{
		resource.SqlColumnSubnet4: subnetId,
		util.SqlOrderBy:           resource.SqlColumnSubnet4}, &reservations); err != nil {
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

func (d DHCPService) GetWithLease4BySubnet4Id(subnetId string) ([]*pbdhcp.DhcpSubnetLease4, error) {
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

func (d *DHCPService) GetWithLease4ByIp(subnet4Id, ip string) (*pbdhcp.DhcpSubnetLease4, error) {
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
		return parser.EncodeDhcpSubnetLeases4(tmpList[0]), nil
	}
}

func (d *DHCPService) GetAllSubnet6() ([]*pbdhcp.DhcpSubnet6, error) {
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

func (d *DHCPService) GetWithSubnet6ByPrefixes(prefixes []string) ([]*pbdhcp.DhcpSubnet6, error) {
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

func (d *DHCPService) GetWithPool6BySubnet6Id(subnetId string) ([]*pbdhcp.DhcpPool6, error) {
	var subnet *resource.Subnet6
	var pools []*resource.Pool6
	var reservations []*resource.Reservation6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if tmpSubnet, err := getSubnet6FromDB(tx, subnetId); err != nil {
			return err
		} else {
			subnet = tmpSubnet
		}
		if err := tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet6: subnet.GetID(),
			util.SqlOrderBy:           resource.SqlColumnBeginIp}, &pools); err != nil {
			return err
		}
		return tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnet.GetID()}, &reservations)
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

func (d *DHCPService) GetWithReservedPool6BySubnet6Id(subnetId string) ([]*pbdhcp.DhcpReservedPool6, error) {
	var subnet *resource.Subnet6
	var pools []*resource.ReservedPool6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if subnet6, err := getSubnet6FromDB(tx, subnetId); err != nil {
			return err
		} else {
			subnet = subnet6
		}
		return tx.Fill(map[string]interface{}{
			resource.SqlColumnSubnet6: subnet.GetID(),
			util.SqlOrderBy:           "begin_ip"}, &pools)
	}); err != nil {
		return nil, fmt.Errorf("list pools with subnet %s from db failed: %s",
			subnet.GetID(), err.Error())
	}
	return pbDhcpReservedPool6FromReservedPool6(pools)
}

func (d *DHCPService) GetWithReservation6BySubnet6Id(subnetId string) ([]*pbdhcp.DhcpReservation6, error) {
	var reservations []*resource.Reservation6
	if err := db.GetResources(map[string]interface{}{
		resource.SqlColumnSubnet6: subnetId,
		util.SqlOrderBy:           "duid, hw_address"}, &reservations); err != nil {
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

func (d *DHCPService) GetWithLease6BySubnet6Id(subnet6Id string) ([]*pbdhcp.DhcpSubnetLease6, error) {
	var subnet6SubnetId uint64
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnet6Id)
		if err != nil {
			return err
		}
		subnet6SubnetId = subnet6.SubnetId
		reservations, subnetLeases, err = getReservation6sAndSubnetLease6s(tx, subnet6Id)
		return err
	}); err != nil {
		if err == ErrorIpNotBelongToSubnet {
			return nil, nil
		} else {
			return nil, fmt.Errorf("get subnet6 %s from db failed: %s", subnet6Id, err.Error())
		}
	}
	if subnetLeasesList, err := getSubnetLease6s(subnet6SubnetId, reservations, subnetLeases); err != nil {
		return nil, nil
	} else {
		return dpDhcpSubnetLease6FromSubnetLease6(subnetLeasesList)
	}
}

func (d *DHCPService) GetWithLease6ByIp(subnetId, ip6 string) (*pbdhcp.DhcpSubnetLease6, error) {
	var subnet6SubnetId uint64
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		subnet6, err := getSubnet6FromDB(tx, subnetId)
		if err != nil {
			return err
		}
		subnet6SubnetId = subnet6.SubnetId
		reservations, subnetLeases, err = getReservation6sAndSubnetLease6sWithIp(tx, subnet6, ip6)
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
		return parser.EncodeDhcpSubnetLease6(tmpList[0]), nil
	}
}
