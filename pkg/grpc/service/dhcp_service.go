package service

import (
	"context"
	"fmt"
	"strings"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/log"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/grpc/parser"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
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
		pbsubnets[ip] = parser.EncodeDhcpSubnet4FromSubnet4(subnet, leasesCount[subnet.SubnetId])
	}
	return pbsubnets
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
		pbsubnets[ip] = parser.EncodeDhcpSubnet6FromSubnet6(subnet, leasesCount[subnet.SubnetId])
	}

	return pbsubnets
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

	lease4, err := service.GetSubnetLease4WithoutReclaimed(subnet.SubnetId, ip, subnetLeases)
	if err != nil {
		log.Warnf("get subnet lease with ip %s and subnet4 %s failed: %s",
			ip, subnet.Subnet, err.Error())
	}

	ipv4Info[ip].AddressType = addressType.String()
	ipv4Info[ip].Lease = parser.EncodeDhcpLease4FromSubnetLease4(lease4)
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

	lease6, err := service.GetSubnetLease6WithoutReclaimed(subnet.SubnetId, ip, subnetLeases)
	if err != nil {
		log.Warnf("get subnet lease with ip %s and subnet6 %s failed: %s",
			ip, subnet.Subnet, err.Error())
	}

	ipv6Info[ip].AddressType = addressType.String()
	ipv6Info[ip].Lease = parser.EncodeDhcpLease6FromSubnetLease6(lease6)
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
		subnetLease4 := parser.DecodeSubnetLease4FromPbLease4(lease4)
		if subnetLease, ok := subnetLeases[ip]; ok && subnetLease.Equal(subnetLease4) {
			continue
		}

		ipv4Infos[ip].Lease = parser.EncodeDhcpLease4FromSubnetLease4(subnetLease4)
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
		subnetLease6 := parser.DecodeSubnetLease6FromPbLease6(lease6)
		if subnetLease, ok := subnetLeases[ip]; ok && subnetLease.Equal(subnetLease6) {
			continue
		}

		ipv6Infos[ip].Lease = parser.EncodeDhcpLease6FromSubnetLease6(subnetLease6)
	}

	return nil
}

func (d *DHCPService) GetAllSubnet4s() ([]*pbdhcp.DhcpSubnet4, error) {
	listCtx := service.GenGrpcGetSubnetsContext(resource.TableSubnet4)
	subnets, _, err := service.GetSubnet4List(listCtx)
	if err != nil {
		return nil, err
	}
	return pbDhcpSubnet4sFromSubnet4List(subnets)
}

func (d *DHCPService) GetWithSubnet4sByPrefixes(prefixes []string) ([]*pbdhcp.DhcpSubnet4, error) {
	for _, subnet := range prefixes {
		if _, err := gohelperip.ParseCIDRv4(subnet); err != nil {
			return nil, fmt.Errorf("action check subnet could be created input invalid: %s", err.Error())
		}
	}
	subnets, err := service.GetListWithSubnet4s(prefixes)
	if err != nil {
		return nil, err
	}
	return pbDhcpSubnet4sFromSubnet4List(subnets.Subnet4s)
}

func pbDhcpSubnet4sFromSubnet4List(subnetList []*resource.Subnet4) ([]*pbdhcp.DhcpSubnet4, error) {
	tmpList := make([]*pbdhcp.DhcpSubnet4, 0)
	for _, v := range subnetList {
		tmpList = append(tmpList, parser.EncodeDhcpSubnet4(v))
	}
	return tmpList, nil
}

func (d *DHCPService) GetPool4List(subnet *resource.Subnet4) ([]*pbdhcp.DhcpPool4, error) {
	pools, err := service.GetPool4List(subnet)
	if err != nil {
		return nil, err
	}
	return pdDhcpPool4FromPool4List(pools)
}

func pdDhcpPool4FromPool4List(pools []*resource.Pool4) ([]*pbdhcp.DhcpPool4, error) {
	ret := make([]*pbdhcp.DhcpPool4, 0)
	for _, v := range pools {
		ret = append(ret, parser.EncodeDhcpPool4(v))
	}
	return ret, nil
}

func (d *DHCPService) GetReservedPool4List(subnet4Id string) ([]*pbdhcp.DhcpReservedPool4, error) {
	pools, err := service.GetReservedPool4List(subnet4Id)
	if err != nil {
		return nil, err
	}
	return pbDhcpReservedPool4FromReservedPool4(pools)
}

func pbDhcpReservedPool4FromReservedPool4(pools []*resource.ReservedPool4) ([]*pbdhcp.DhcpReservedPool4, error) {
	tmpPools := make([]*pbdhcp.DhcpReservedPool4, 0)
	for _, v := range pools {
		tmpPools = append(tmpPools, parser.EncodeDhcpReservedPool4(v))
	}
	return tmpPools, nil
}

func (d *DHCPService) GetReservation4List(subnetId string) ([]*pbdhcp.DhcpReservation4, error) {
	reservations, err := service.GetReservation4List(subnetId)
	if err != nil {
		return nil, err
	}
	return pbDchpReservation4FormReservation4(reservations)
}

func pbDchpReservation4FormReservation4(reservations []*resource.Reservation4) ([]*pbdhcp.DhcpReservation4, error) {
	tmpList := make([]*pbdhcp.DhcpReservation4, 0)
	for _, v := range reservations {
		tmpList = append(tmpList, parser.EncodeDhcpReservation4(v))
	}
	return tmpList, nil
}

func (d DHCPService) GetSubnetLease4List(subnetId string) ([]*pbdhcp.DhcpSubnetLease4, error) {
	subnetLeasesList, err := service.GetSubnetLease4List(subnetId)
	if err != nil {
		return nil, err
	}
	return pbDhcpSubnetLeasesFromSubnetLeases(subnetLeasesList)
}

func pbDhcpSubnetLeasesFromSubnetLeases(subnetLeases []*resource.SubnetLease4) ([]*pbdhcp.DhcpSubnetLease4, error) {
	tmpList := make([]*pbdhcp.DhcpSubnetLease4, 0)
	for _, v := range subnetLeases {
		tmpList = append(tmpList, parser.EncodeDhcpSubnetLeases4(v))
	}
	return tmpList, nil
}

func (d *DHCPService) GetSubnetLease4ByIp(subnetId, ip string) (*pbdhcp.DhcpSubnetLease4, error) {
	subnetLeasesList, err := service.GetSubnetLease4ListWithIp(subnetId, ip)
	if err != nil {
		return nil, err
	}
	if len(subnetLeasesList) == 0 {
		return nil, nil
	}
	return parser.EncodeDhcpSubnetLeases4(subnetLeasesList[0]), nil
}

func (d *DHCPService) GetAllSubnet6() ([]*pbdhcp.DhcpSubnet6, error) {
	listCtx := service.GenGrpcGetSubnetsContext(resource.TableSubnet6)
	subnets, _, err := service.GetSubnet6List(listCtx)
	if err != nil {
		return nil, err
	}
	return pbDhcpSubnet6FromSubnet6(subnets)
}

func pbDhcpSubnet6FromSubnet6(subnets []*resource.Subnet6) ([]*pbdhcp.DhcpSubnet6, error) {
	tmpList := make([]*pbdhcp.DhcpSubnet6, 0)
	for _, v := range subnets {
		tmpList = append(tmpList, parser.EncodeDhcpSubnet6(v))
	}
	return tmpList, nil
}

func (d *DHCPService) GetWithSubnet6ByPrefixes(prefixes []string) ([]*pbdhcp.DhcpSubnet6, error) {
	for _, subnet := range prefixes {
		if _, err := gohelperip.ParseCIDRv6(subnet); err != nil {
			return nil, fmt.Errorf("action check subnet could be created input subnet %s invalid: %s", subnet, err.Error())
		}
	}
	ret, err := service.GetListWithSubnet6s(prefixes)
	if err != nil {
		return nil, err
	}
	return pbDhcpFromSubnet6s(ret.Subnet6s)
}

func pbDhcpFromSubnet6s(subnets []*resource.Subnet6) ([]*pbdhcp.DhcpSubnet6, error) {
	tmpList := make([]*pbdhcp.DhcpSubnet6, 0)
	for _, v := range subnets {
		tmpList = append(tmpList, parser.EncodeDhcpSubnet6(v))
	}
	return tmpList, nil
}

func (d *DHCPService) GetPool6List(subnet *resource.Subnet6) ([]*pbdhcp.DhcpPool6, error) {
	pools, err := service.GetPool6List(subnet)
	if err != nil {
		return nil, err
	}
	return pbDhcpPoolsFromPools(pools)
}

func pbDhcpPoolsFromPools(pools []*resource.Pool6) ([]*pbdhcp.DhcpPool6, error) {
	tmpList := make([]*pbdhcp.DhcpPool6, 0)
	for _, v := range pools {
		tmpList = append(tmpList, parser.EncodeDhcpPool6(v))
	}
	return tmpList, nil
}

func (d *DHCPService) GetReservedPool6List(subnet *resource.Subnet6) ([]*pbdhcp.DhcpReservedPool6, error) {
	pools, err := service.GetReservedPool6List(subnet)
	if err != nil {
		return nil, err
	}
	return pbDhcpReservedPool6FromReservedPool6(pools)
}

func pbDhcpReservedPool6FromReservedPool6(pools []*resource.ReservedPool6) ([]*pbdhcp.DhcpReservedPool6, error) {
	tmpList := make([]*pbdhcp.DhcpReservedPool6, 0)
	for _, v := range pools {
		tmpList = append(tmpList, parser.EncodeDhcpReservedPool6(v))
	}
	return tmpList, nil
}

func (d *DHCPService) GetReservation6List(subnetId string) ([]*pbdhcp.DhcpReservation6, error) {
	reservations, err := service.GetReservation6List(subnetId)
	if err != nil {
		return nil, err
	}
	return pbDhcpReservation6FromReservation6(reservations)
}

func pbDhcpReservation6FromReservation6(reservations []*resource.Reservation6) ([]*pbdhcp.DhcpReservation6, error) {
	tmpList := make([]*pbdhcp.DhcpReservation6, 0)
	for _, v := range reservations {
		tmpList = append(tmpList, parser.EncodeDhcpReservation6(v))
	}
	return tmpList, nil
}

func (d *DHCPService) GetSLease6ListBySubnetId(subnetId string) ([]*pbdhcp.DhcpSubnetLease6, error) {
	subnetLeasesList, err := service.GetSubnetLease6List(subnetId)
	if err != nil {
		return nil, err
	}
	return dpDhcpSubnetLease6FromSubnetLease6(subnetLeasesList)

}

func dpDhcpSubnetLease6FromSubnetLease6(subnetLeases []*resource.SubnetLease6) ([]*pbdhcp.DhcpSubnetLease6, error) {
	tmpList := make([]*pbdhcp.DhcpSubnetLease6, 0)
	for _, v := range subnetLeases {
		tmpList = append(tmpList, parser.EncodeDhcpSubnetLease6(v))
	}
	return tmpList, nil
}

func (d *DHCPService) GetSubnetLease6ByIp(subnetId, ip string) (*pbdhcp.DhcpSubnetLease6, error) {
	subnetLeasesList, err := service.GetSubnetLease6ListByIp(subnetId, ip)
	if err != nil {
		return nil, err
	}
	if len(subnetLeasesList) == 0 {
		return nil, nil
	}
	return parser.EncodeDhcpSubnetLease6(subnetLeasesList[0]), nil
}

func (d *DHCPService) CreateReservation4s(subnetId string, reservations []*resource.Reservation4) (bool, error) {
	subnet := &resource.Subnet4{}
	subnet.SetID(subnetId)
	_, err := service.BatchCreateReservation4s(subnet, reservations)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *DHCPService) CreateReservedPool4(subnetId string, pool []*resource.ReservedPool4) (bool, error) {
	subnet := &resource.Subnet4{}
	subnet.SetID(subnetId)
	_, err := service.BatchCreateReservedPool4(subnet, pool)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *DHCPService) CreateReservation6s(subnetId string, reservations []*resource.Reservation6) (bool, error) {
	subnet := &resource.Subnet6{}
	subnet.SetID(subnetId)
	_, err := service.BatchCreateReservation6s(subnet, reservations)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *DHCPService) CreateReservedPool6(subnetId string, pools []*resource.ReservedPool6) (bool, error) {
	subnet := &resource.Subnet6{}
	subnet.SetID(subnetId)
	_, err := service.BatchCreateReservedPool6(subnet, pools)
	if err != nil {
		return false, err
	}
	return true, nil
}
