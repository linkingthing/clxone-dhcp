package service

import (
	"context"
	"strings"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/grpc/parser"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
	}

	if len(subnets) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV4, ip)
	}

	return subnet4sToPbDHCPSubnet4s(map[string]*resource.Subnet4{ip: subnets[0]}), nil
}

func subnet4sToPbDHCPSubnet4s(subnets map[string]*resource.Subnet4) map[string]*pbdhcp.Subnet4 {
	leasesCount, err := getSubnet4sLeasesCount(subnets)
	if err != nil {
		log.Warnf("get subnet4s leases count failed: %s", err.Error())
	}

	pbsubnets := make(map[string]*pbdhcp.Subnet4)
	for ip, subnet := range subnets {
		pbsubnets[ip] = parser.Subnet4ToPbDHCPSubnet4(subnet, leasesCount[subnet.SubnetId])
	}

	return pbsubnets
}

func getSubnet4sLeasesCount(subnets map[string]*resource.Subnet4) (map[uint64]uint64, error) {
	var subnetIds []uint64
	for _, subnet := range subnets {
		if subnet.Capacity != 0 {
			subnetIds = append(subnetIds, subnet.SubnetId)
		}
	}

	if len(subnetIds) == 0 {
		return nil, nil
	} else {
		var err error
		var resp *pbdhcpagent.GetSubnetsLeasesCountResponse
		err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
			resp, err = client.GetSubnets4LeasesCountWithIds(ctx,
				&pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: subnetIds})
			return err
		})

		return resp.GetSubnetsLeasesCount(), err
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	}

	if len(subnets) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV6, ip)
	}

	return subnet6sToPbDHCPSubnet6s(map[string]*resource.Subnet6{ip: subnets[0]}), nil
}

func subnet6sToPbDHCPSubnet6s(subnets map[string]*resource.Subnet6) map[string]*pbdhcp.Subnet6 {
	leasesCount, err := getSubnet6sLeasesCount(subnets)
	if err != nil {
		log.Warnf("get subnet6 leases count failed: %s", err.Error())
	}

	pbsubnets := make(map[string]*pbdhcp.Subnet6)
	for ip, subnet := range subnets {
		pbsubnets[ip] = parser.Subnet6ToPbDHCPSubnet6(subnet, leasesCount[subnet.SubnetId])
	}

	return pbsubnets
}

func getSubnet6sLeasesCount(subnets map[string]*resource.Subnet6) (map[uint64]uint64, error) {
	var subnetIds []uint64
	for _, subnet := range subnets {
		if !resource.IsCapacityZero(subnet.Capacity) {
			subnetIds = append(subnetIds, subnet.SubnetId)
		}
	}

	if len(subnetIds) == 0 {
		return nil, nil
	} else {
		var err error
		var resp *pbdhcpagent.GetSubnetsLeasesCountResponse
		err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
			resp, err = client.GetSubnets6LeasesCountWithIds(ctx,
				&pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: subnetIds})
			return err
		})

		return resp.GetSubnetsLeasesCount(), err
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
	}

	if len(subnets) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV4, "")
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

	if len(closestSubnets) == 0 && len(ips) > 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV4, ips[0])
	}

	return subnet4sToPbDHCPSubnet4s(closestSubnets), nil
}

func (d *DHCPService) GetSubnets6WithIps(ips []string) (map[string]*pbdhcp.Subnet6, error) {
	return getSubnets6WithIps(ips)
}

func getSubnets6WithIps(ips []string) (map[string]*pbdhcp.Subnet6, error) {
	if err := gohelperip.CheckIPv6sValid(ips...); err != nil {
		return nil, err
	}

	var subnets []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &subnets)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	}

	if len(subnets) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV6, "")
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

	if len(closestSubnets) == 0 && len(ips) > 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkV6, ips[0])
	}

	return subnet6sToPbDHCPSubnet6s(closestSubnets), nil
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
		addressType, err = getIPv4AddressType(tx, subnet.Id, ip)
		if err != nil {
			return
		}

		return tx.Fill(map[string]interface{}{resource.SqlColumnAddress: ip,
			resource.SqlColumnSubnet4: subnet.Id}, &subnetLeases)
	}); err != nil {
		log.Warnf("get address type and reclaimed leases with subnet4 %s failed: %s",
			subnet.Subnet, pg.Error(err).Error())
		return ipv4Info, nil
	}

	lease4, err := service.GetSubnetLease4WithoutReclaimed(subnet.SubnetId, ip, subnetLeases)
	if err != nil {
		log.Warnf("get subnet lease with ip %s and subnet4 %s failed: %s",
			ip, subnet.Subnet, err.Error())
	}

	ipv4Info[ip].AddressType = addressType.String()
	ipv4Info[ip].Lease = parser.SubnetLease4ToPbDHCPLease4(lease4)
	return ipv4Info, nil
}

func getIPv4AddressType(tx restdb.Transaction, subnetId, ip string) (resource.AddressType, error) {
	addressType := resource.AddressTypeExclusion
	if exists, err := tx.Exists(resource.TableReservation4,
		map[string]interface{}{resource.SqlColumnIpAddress: ip, resource.SqlColumnSubnet4: subnetId}); err != nil {
		return addressType, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	} else if exists {
		return resource.AddressTypeReservation, nil
	}

	if count, err := tx.CountEx(resource.TableReservedPool4,
		"select count(*) from gr_reserved_pool4 where subnet4 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetId, ip, ip); err != nil {
		return addressType, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
	} else if count != 0 {
		return resource.AddressTypeReserve, nil
	}

	if count, err := tx.CountEx(resource.TablePool4,
		"select count(*) from gr_pool4 where subnet4 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetId, ip, ip); err != nil {
		return addressType, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
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
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		addressType, err = getIPv6AddressType(tx, subnet.Id, ip)
		if err != nil {
			return
		}

		if err = tx.Fill(map[string]interface{}{resource.SqlColumnAddress: ip,
			resource.SqlColumnSubnet6: subnet.Id}, &subnetLeases); err != nil {
			return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkLease), pg.Error(err).Error())
		}
		return nil
	}); err != nil {
		log.Warnf("get address type and reclaimed leases with subnet6 %s failed: %s",
			subnet.Subnet, pg.Error(err).Error())
		return ipv6Info, nil
	}

	lease6, err := service.GetSubnetLease6WithoutReclaimed(subnet.SubnetId, ip, subnetLeases)
	if err != nil {
		log.Warnf("get subnet lease with ip %s and subnet6 %s failed: %s",
			ip, subnet.Subnet, err.Error())
	}

	ipv6Info[ip].AddressType = addressType.String()
	ipv6Info[ip].Lease = parser.SubnetLease6ToPbDHCPLease6(lease6)
	return ipv6Info, nil
}

func getIPv6AddressType(tx restdb.Transaction, subnetId, ip string) (resource.AddressType, error) {
	addressType := resource.AddressTypeExclusion
	if count, err := tx.CountEx(resource.TableReservation6,
		"select count(*) from gr_reservation6 where subnet6 = $1 and $2::text = any(ip_addresses)",
		subnetId, ip); err != nil {
		return addressType, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	} else if count != 0 {
		return resource.AddressTypeReservation, nil
	}

	if count, err := tx.CountEx(resource.TableReservedPool6,
		"select count(*) from gr_reserved_pool6 where subnet6 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetId, ip, ip); err != nil {
		return addressType, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
	} else if count != 0 {
		return resource.AddressTypeReserve, nil
	}

	if count, err := tx.CountEx(resource.TablePool6,
		"select count(*) from gr_pool6 where subnet6 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetId, ip, ip); err != nil {
		return addressType, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
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

	subnetIds := make([]string, 0, len(subnets))
	unfoundIps := make(map[string]struct{})
	ipv4Infos := make(map[string]*pbdhcp.Ipv4Information)
	for ip, subnet := range subnets {
		subnetIds = append(subnetIds, subnet.Id)
		unfoundIps[ip] = struct{}{}
		ipv4Infos[ip] = &pbdhcp.Ipv4Information{
			Address:     ip,
			AddressType: resource.AddressTypeExclusion.String(),
			Subnet:      subnet,
		}
	}

	var subnetLeases map[string]*resource.SubnetLease4
	subnetIdsArgs := strings.Join(subnetIds, "','")
	ipsArgs := strings.Join(ips, "','")
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		if err = setIpv4InfosAddressType(tx, subnetIdsArgs, ipsArgs, unfoundIps, ipv4Infos); err != nil {
			return
		}

		subnetLeases, err = getReclaimedSubnetLease4s(tx, subnetIdsArgs, ipsArgs)
		return
	}); err != nil {
		log.Warnf("get address type and reclaimed lease4s failed: %s", pg.Error(err).Error())
		return ipv4Infos, nil
	}

	if err := setSubnetLease4sWithoutReclaimed(ipv4Infos, subnetLeases); err != nil {
		log.Warnf("get subnet4 leases failed: %s", err.Error())
	}

	return ipv4Infos, nil
}

func setIpv4InfosAddressType(tx restdb.Transaction, subnetIdsArgs, ipsArgs string, unfoundIps map[string]struct{}, ipv4Infos map[string]*pbdhcp.Ipv4Information) error {
	var reservations []*resource.Reservation4
	if err := tx.FillEx(&reservations, "select * from gr_reservation4 where ip_address in ('"+
		ipsArgs+"') and subnet4 in ('"+subnetIdsArgs+"')"); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
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
	if err := tx.FillEx(reservedPools, "select * from gr_reserved_pool4 where subnet4 in ('"+
		subnetIdsArgs+"')"); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
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
	if err := tx.FillEx(pools, "select * from gr_pool4 where subnet4 in ('"+
		subnetIdsArgs+"')"); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
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

func getReclaimedSubnetLease4s(tx restdb.Transaction, subnetIdsArgs, ipsArgs string) (map[string]*resource.SubnetLease4, error) {
	var subnetLeases []*resource.SubnetLease4
	if err := tx.FillEx(&subnetLeases, "select * from gr_subnet_lease4 where address in ('"+
		ipsArgs+"') and subnet4 in ('"+subnetIdsArgs+"')"); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkLease), pg.Error(err).Error())
	}

	subnetLeasesMap := make(map[string]*resource.SubnetLease4)
	for _, subnetLease := range subnetLeases {
		subnetLeasesMap[subnetLease.Address] = subnetLease
	}

	return subnetLeasesMap, nil
}

func setSubnetLease4sWithoutReclaimed(ipv4Infos map[string]*pbdhcp.Ipv4Information, reclaimedSubnetLeases map[string]*resource.SubnetLease4) error {
	reqs := make([]*pbdhcpagent.GetSubnet4LeaseRequest, 0, len(ipv4Infos))
	for ip, info := range ipv4Infos {
		reqs = append(reqs, &pbdhcpagent.GetSubnet4LeaseRequest{
			Id:      info.Subnet.SubnetId,
			Address: ip,
		})
	}

	var err error
	var resp *pbdhcpagent.GetLeases4Response
	if err = transport.CallDhcpAgentGrpc4(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet4LeasesWithIps(ctx,
			&pbdhcpagent.GetSubnet4LeasesWithIpsRequest{Addresses: reqs})
		return err
	}); err != nil {
		return errorno.ErrNetworkError(errorno.ErrNameNetworkLease, err.Error())
	}

	for _, lease4 := range resp.GetLeases() {
		if subnetLease, ok := reclaimedSubnetLeases[lease4.Address]; ok &&
			subnetLease.Address == lease4.GetAddress() &&
			subnetLease.Expire == service.TimeFromUinx(lease4.GetExpire()) &&
			subnetLease.HwAddress == lease4.GetHwAddress() &&
			subnetLease.ClientId == lease4.GetClientId() {
			continue
		}

		ipv4Infos[lease4.Address].Lease = pbDHCPLease4FromPbDHCPAgentDHCPLease4(lease4)
	}

	return nil
}

func pbDHCPLease4FromPbDHCPAgentDHCPLease4(lease *pbdhcpagent.DHCPLease4) *pbdhcp.Lease4 {
	return &pbdhcp.Lease4{
		Address:               lease.GetAddress(),
		HwAddress:             lease.GetHwAddress(),
		HwAddressOrganization: lease.GetHwAddressOrganization(),
		ClientId:              lease.GetClientId(),
		ValidLifetime:         lease.GetValidLifetime(),
		Expire:                service.TimeFromUinx(lease.GetExpire()),
		Hostname:              lease.GetHostname(),
		VendorId:              lease.GetVendorId(),
		OperatingSystem:       lease.GetOperatingSystem(),
		ClientType:            lease.GetClientType(),
		LeaseState:            lease.GetLeaseState().String(),
	}
}

func (d *DHCPService) GetSubnets6AndLeases6WithIps(ips []string) (map[string]*pbdhcp.Ipv6Information, error) {
	subnets, err := getSubnets6WithIps(ips)
	if err != nil {
		return nil, err
	}

	subnetIds := make([]string, 0, len(subnets))
	unfoundIps := make(map[string]struct{})
	ipv6Infos := make(map[string]*pbdhcp.Ipv6Information)
	for ip, subnet := range subnets {
		subnetIds = append(subnetIds, subnet.Id)
		unfoundIps[ip] = struct{}{}
		ipv6Infos[ip] = &pbdhcp.Ipv6Information{
			Address:     ip,
			AddressType: resource.AddressTypeExclusion.String(),
			Subnet:      subnet,
		}
	}

	subnetIdsArgs := strings.Join(subnetIds, "','")
	var subnetLeases map[string]*resource.SubnetLease6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) (err error) {
		if err = setIpv6InfosAddressType(tx, subnetIdsArgs, unfoundIps, ipv6Infos); err != nil {
			return
		}

		subnetLeases, err = getReclaimedSubnetLease6s(tx, subnetIdsArgs, strings.Join(ips, "','"))
		return
	}); err != nil {
		log.Warnf("get address type and reclaimed lease6s failed: %s", pg.Error(err).Error())
		return ipv6Infos, nil
	}

	if err := setSubnetLease6sWithoutReclaimed(ipv6Infos, subnetLeases); err != nil {
		log.Warnf("get subnet6 leases failed: %s", err.Error())
	}

	return ipv6Infos, nil
}

func setIpv6InfosAddressType(tx restdb.Transaction, subnetIdsArgs string, unfoundIps map[string]struct{}, ipv6Infos map[string]*pbdhcp.Ipv6Information) error {
	var reservations []*resource.Reservation6
	if err := tx.FillEx(reservations, "select * from gr_reservation6 where subnet6 in ('"+
		subnetIdsArgs+"')"); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservation), pg.Error(err).Error())
	}

	for _, reservation := range reservations {
		for _, ipaddress := range reservation.IpAddresses {
			if ipv6Info, ok := ipv6Infos[ipaddress]; ok {
				ipv6Info.AddressType = resource.AddressTypeReservation.String()
				delete(unfoundIps, ipaddress)
			}
		}
	}

	if len(unfoundIps) == 0 {
		return nil
	}

	var reservedPools []*resource.ReservedPool6
	if err := tx.FillEx(reservedPools, "select * from gr_reserved_pool6 where subnet6 in ('"+
		subnetIdsArgs+"')"); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpReservedPool), pg.Error(err).Error())
	}

	for ip := range unfoundIps {
		for _, reservedPool := range reservedPools {
			if reservedPool.ContainsIpString(ip) {
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
	if err := tx.FillEx(pools, "select * from gr_pool6 where subnet6 in ('"+
		subnetIdsArgs+"')"); err != nil {
		return errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameDhcpPool), pg.Error(err).Error())
	}

	for ip := range unfoundIps {
		for _, pool := range pools {
			if pool.ContainsIpString(ip) {
				ipv6Infos[ip].AddressType = resource.AddressTypeDynamic.String()
				delete(unfoundIps, ip)
				break
			}
		}
	}

	return nil
}

func getReclaimedSubnetLease6s(tx restdb.Transaction, subnetIdsArgs, ipsArgs string) (map[string]*resource.SubnetLease6, error) {
	var subnetLeases []*resource.SubnetLease6
	if err := tx.FillEx(&subnetLeases, "select * from gr_subnet_lease6 where address in ('"+
		ipsArgs+"') and subnet6 in ('"+subnetIdsArgs+"')"); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkLease), pg.Error(err).Error())
	}

	subnetLeasesMap := make(map[string]*resource.SubnetLease6)
	for _, subnetLease := range subnetLeases {
		subnetLeasesMap[subnetLease.Address] = subnetLease
	}

	return subnetLeasesMap, nil
}

func setSubnetLease6sWithoutReclaimed(ipv6Infos map[string]*pbdhcp.Ipv6Information, reclaimedSubnetLeases map[string]*resource.SubnetLease6) error {
	reqs := make([]*pbdhcpagent.GetSubnet6LeaseRequest, 0, len(ipv6Infos))
	for ip, info := range ipv6Infos {
		reqs = append(reqs, &pbdhcpagent.GetSubnet6LeaseRequest{
			Id:      info.Subnet.SubnetId,
			Address: ip,
		})
	}

	var err error
	var resp *pbdhcpagent.GetLeases6Response
	if err = transport.CallDhcpAgentGrpc6(func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error {
		resp, err = client.GetSubnet6LeasesWithIps(ctx,
			&pbdhcpagent.GetSubnet6LeasesWithIpsRequest{Addresses: reqs})
		return err
	}); err != nil {
		return errorno.ErrNetworkError(errorno.ErrNameNetworkLease, err.Error())
	}

	for _, lease := range resp.GetLeases() {
		if subnetLease, ok := reclaimedSubnetLeases[lease.Address]; ok &&
			subnetLease.Address == lease.GetAddress() &&
			subnetLease.Expire == service.TimeFromUinx(lease.GetExpire()) &&
			subnetLease.Duid == lease.GetDuid() &&
			subnetLease.HwAddress == lease.GetHwAddress() &&
			subnetLease.LeaseType == lease.GetLeaseType() &&
			subnetLease.Iaid == lease.GetIaid() {
			continue
		}

		ipv6Infos[lease.Address].Lease = pbDHCPLease6FromPbDHCPAgentDHCPLease6(lease)
	}

	return nil
}

func pbDHCPLease6FromPbDHCPAgentDHCPLease6(lease *pbdhcpagent.DHCPLease6) *pbdhcp.Lease6 {
	return &pbdhcp.Lease6{
		Address:               lease.GetAddress(),
		PrefixLen:             lease.GetPrefixLen(),
		Duid:                  lease.GetDuid(),
		Iaid:                  lease.GetIaid(),
		HwAddress:             lease.GetHwAddress(),
		HwAddressType:         lease.GetHwAddressType(),
		HwAddressSource:       lease.GetHwAddressSource().String(),
		HwAddressOrganization: lease.GetHwAddressOrganization(),
		ValidLifetime:         lease.GetValidLifetime(),
		PreferredLifetime:     lease.GetPreferredLifetime(),
		Expire:                service.TimeFromUinx(lease.GetExpire()),
		LeaseType:             lease.GetLeaseType(),
		Hostname:              lease.GetHostname(),
		VendorId:              lease.GetVendorId(),
		OperatingSystem:       lease.GetOperatingSystem(),
		ClientType:            lease.GetClientType(),
		LeaseState:            lease.GetLeaseState().String(),
	}
}

func (d *DHCPService) GetAllSubnet4s() ([]*pbdhcp.Subnet4, error) {
	var subnets []*resource.Subnet4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &subnets)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
	}

	if err := service.SetSubnet4UsedInfo(subnets, true); err != nil {
		log.Warnf("set subnet4s leases used info failed: %s", err.Error())
	}

	return parser.Subnet4sToPbDHCPSubnet4s(subnets), nil
}

func (d *DHCPService) GetSubnet4sByPrefixes(prefixes []string) ([]*pbdhcp.Subnet4, error) {
	if subnets, err := service.ListSubnet4sByPrefixes(prefixes); err != nil {
		return nil, err
	} else {
		return parser.Subnet4sToPbDHCPSubnet4s(subnets), nil
	}
}

func (d *DHCPService) GetPool4sBySubnet(prefix string) ([]*pbdhcp.Pool4, error) {
	if pools, err := service.GetPool4sByPrefix(prefix); err != nil {
		return nil, err
	} else {
		return parser.Pool4sToPbDHCPPool4s(pools), nil
	}
}

func (d *DHCPService) GetReservedPool4sBySubnet(prefix string) ([]*pbdhcp.ReservedPool4, error) {
	if pools, err := service.GetReservedPool4sByPrefix(prefix); err != nil {
		return nil, err
	} else {
		return parser.ReservedPool4sToPbDHCPReservedPool4s(pools), nil
	}
}

func (d *DHCPService) GetReservation4sBySubnet(prefix string) ([]*pbdhcp.Reservation4, error) {
	if pools, err := service.GetReservationPool4sByPrefix(prefix); err != nil {
		return nil, err
	} else {
		return parser.Reservation4sToPbDHCPReservation4s(pools), nil
	}
}

func (d DHCPService) GetLease4ByIp(ip string) (*pbdhcp.Lease4, error) {
	subnet, err := service.GetSubnet4ByIP(ip)
	if err != nil {
		return nil, err
	}

	lease4s, err := service.ListSubnetLease4(subnet, ip)
	if err != nil {
		return nil, err
	}

	if len(lease4s) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkLease, ip)
	} else {
		return parser.SubnetLease4ToPbDHCPLease4(lease4s[0]), nil
	}
}

func (d DHCPService) GetLease4ByPrefix(prefix string) ([]*pbdhcp.Lease4, error) {
	subnet, err := service.GetSubnet4ByPrefix(prefix)
	if err != nil {
		return nil, err
	}

	if lease4s, err := service.ListSubnetLease4(subnet, ""); err != nil {
		return nil, err
	} else {
		return parser.SubnetLeases4sToPbDHCPLease4s(lease4s), nil
	}
}

func (d DHCPService) GetLease4SWithMacs(hwAddresses []string) ([]*pbdhcp.Lease4, error) {
	lease4s, err := service.GetSubnets4LeasesWithMacs(hwAddresses)
	if err != nil {
		return nil, err
	}
	return parser.SubnetLeases4sToPbDHCPLease4s(lease4s), nil
}

func (d DHCPService) GetLease6SWithMacs(hwAddresses []string) ([]*pbdhcp.Lease6, error) {
	lease6s, err := service.GetSubnets6LeasesWithMacs(hwAddresses)
	if err != nil {
		return nil, err
	}
	return parser.SubnetLease6sToPbDHCPLease6s(lease6s), nil
}

func (d *DHCPService) GetAllSubnet6s() ([]*pbdhcp.Subnet6, error) {
	var subnet6s []*resource.Subnet6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &subnet6s)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
	}

	if err := service.SetSubnet6sLeasesUsedInfo(subnet6s, true); err != nil {
		log.Warnf("set subnet6s leases used info failed: %s", err.Error())
	}

	return parser.Subnet6sToPbDHCPSubnet6s(subnet6s), nil
}

func (d *DHCPService) GetSubnet6sByPrefixes(prefixes []string) ([]*pbdhcp.Subnet6, error) {
	if subnet6s, err := service.ListSubnet6sByPrefixes(prefixes); err != nil {
		return nil, err
	} else {
		return parser.Subnet6sToPbDHCPSubnet6s(subnet6s), nil
	}
}

func (d *DHCPService) GetPool6sBySubnet(prefix string) ([]*pbdhcp.Pool6, error) {
	if pools, err := service.GetPool6sByPrefix(prefix); err != nil {
		return nil, err
	} else {
		return parser.Pool6sToPbDHCPPool6s(pools), nil
	}
}

func (d *DHCPService) GetReservedPool6sBySubnet(prefix string) ([]*pbdhcp.ReservedPool6, error) {
	if pools, err := service.GetReservedPool6sByPrefix(prefix); err != nil {
		return nil, err
	} else {
		return parser.ReservedPool6sToPbDHCPReservedPool6s(pools), nil
	}
}

func (d *DHCPService) GetReservation6sBySubnet(prefix string) ([]*pbdhcp.Reservation6, error) {
	if pools, err := service.GetReservationPool6sByPrefix(prefix); err != nil {
		return nil, err
	} else {
		return parser.Reservation6sToPbDHCPReservation6s(pools), nil
	}
}

func (d *DHCPService) GetPdPool6sBySubnet(prefix string) ([]*pbdhcp.PdPool6, error) {
	if pools, err := service.GetPdPool6sByPrefix(prefix); err != nil {
		return nil, err
	} else {
		return parser.PdPool6sToPbDHCPPdPools(pools), nil
	}
}

func (d *DHCPService) GetLease6ByIp(ip string) (*pbdhcp.Lease6, error) {
	subnet, err := service.GetSubnet6ByIP(ip)
	if err != nil {
		return nil, err
	}

	lease6s, err := service.ListSubnetLease6(subnet, ip)
	if err != nil {
		return nil, err
	}

	if len(lease6s) == 0 {
		return nil, errorno.ErrNotFound(errorno.ErrNameNetworkLease, ip)
	} else {
		return parser.SubnetLease6ToPbDHCPLease6(lease6s[0]), nil
	}
}

func (d *DHCPService) GetLease6sBySubnet(prefix string) ([]*pbdhcp.Lease6, error) {
	subnet, err := service.GetSubnet6ByPrefix(prefix)
	if err != nil {
		return nil, err
	}

	if lease6s, err := service.ListSubnetLease6(subnet, ""); err != nil {
		return nil, err
	} else {
		return parser.SubnetLease6sToPbDHCPLease6s(lease6s), nil
	}
}

func (d *DHCPService) CreateReservation4s(prefix string, pbPools []*pbdhcp.Reservation4) error {
	return service.BatchCreateReservation4s(prefix, parser.Reservation4sFromPbDHCPReservation4s(pbPools))
}

func (d *DHCPService) CreateReservedPool4s(prefix string, pools []*pbdhcp.ReservedPool4) error {
	return service.BatchCreateReservedPool4s(prefix, parser.ReservedPool4sFromPbDHCPReservedPool4s(pools))
}

func (d *DHCPService) CreateReservation6s(prefix string, pools []*pbdhcp.Reservation6) error {
	return service.BatchCreateReservation6s(prefix, parser.Reservation6sFromPbDHCPReservation6s(pools))
}

func (d *DHCPService) CreateReservedPool6s(prefix string, pools []*pbdhcp.ReservedPool6) error {
	return service.BatchCreateReservedPool6s(prefix, parser.ReservedPool6sFromPbDHCPReservedPool6s(pools))
}
