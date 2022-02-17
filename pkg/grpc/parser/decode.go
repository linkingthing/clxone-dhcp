package parser

import (
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"net"
	"time"

	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
)

func decodeTimeUnix(t int64) string {
	return time.Unix(t, 0).Format(time.RFC3339)
}

func decodeTimeStr(timeStr string) time.Time {
	t, _ := time.Parse(time.RFC3339, timeStr)
	return t
}

func decodeLinks(old map[string]string) map[restresource.ResourceLinkType]restresource.ResourceLink {
	links := make(map[restresource.ResourceLinkType]restresource.ResourceLink)
	for keyStr, valStr := range old {
		links[restresource.ResourceLinkType(keyStr)] = restresource.ResourceLink(valStr)
	}
	return links
}

func decodeIps(ips []string) []net.IP {
	rets := make([]net.IP, len(ips))
	for _, v := range ips {
		rets = append(rets, decodeIp(v))
	}
	return rets
}

func decodeIp(ip string) net.IP {
	return net.ParseIP(ip)
}

func decodeIpNet(ipNet string) net.IPNet {
	_, retNet, err := net.ParseCIDR(ipNet)
	if err != nil {
		return net.IPNet{}
	}
	return *retNet
}

func DecodePbSubnet4(old *pbdhcp.DhcpSubnet4) *resource.Subnet4 {
	ret := &resource.Subnet4{
		Subnet:              old.GetSubnet(),
		Ipnet:               decodeIpNet(old.GetIpNet()),
		SubnetId:            old.GetSubnetId(),
		ValidLifetime:       old.GetValidLifetime(),
		MaxValidLifetime:    old.GetMaxValidLifetime(),
		MinValidLifetime:    old.GetMinValidLifetime(),
		SubnetMask:          old.GetSubnetMask(),
		DomainServers:       old.GetDomainServers(),
		Routers:             old.GetRouters(),
		ClientClass:         old.GetClientClass(),
		TftpServer:          old.GetTftpServer(),
		Bootfile:            old.GetBootFile(),
		RelayAgentAddresses: old.GetRelayAgentAddresses(),
		IfaceName:           old.GetIFaceName(),
		NextServer:          old.GetIpNet(),
		Tags:                old.GetTags(),
		NodeNames:           old.GetNodeNames(),
		Nodes:               old.GetNodes(),
		Capacity:            old.GetCapacity(),
		UsedRatio:           old.GetUsedRatio(),
		UsedCount:           old.GetUseCount(),
	}
	ret.SetID(old.GetId())
	ret.SetLinks(decodeLinks(old.GetLinks()))
	ret.SetCreationTimestamp(decodeTimeStr(old.GetCreationTimestamp()))
	ret.SetDeletionTimestamp(decodeTimeStr(old.GetDeletionTimestamp()))
	return ret
}

func DecodePbCReservation4(old *pbdhcp.CreateReservation) *resource.Reservation4 {
	ret := &resource.Reservation4{
		HwAddress: old.GetHwAddress(),
		IpAddress: old.GetIpAddress(),
	}
	return ret
}

func DecodePbCReservedPool4(old *pbdhcp.CreateReserved) *resource.ReservedPool4 {
	ret := &resource.ReservedPool4{
		BeginAddress: old.GetBeginAddress(),
		EndAddress:   old.GetEndAddress(),
	}
	return ret
}

func DecodePbSubnet6(old *pbdhcp.DhcpSubnet6) *resource.Subnet6 {
	ret := &resource.Subnet6{
		Subnet:                old.GetSubnet(),
		Ipnet:                 decodeIpNet(old.GetIpNet()),
		SubnetId:              old.GetSubnetId(),
		ValidLifetime:         old.GetValidLifetime(),
		MaxValidLifetime:      old.GetMaxValidLifetime(),
		MinValidLifetime:      old.GetMinValidLifetime(),
		PreferredLifetime:     old.GetPreferredLifetime(),
		DomainServers:         old.GetDomainServers(),
		ClientClass:           old.GetClientClass(),
		IfaceName:             old.GetIFaceName(),
		RelayAgentAddresses:   old.GetRelayAgentAddresses(),
		RelayAgentInterfaceId: old.GetRelayAgentInterfaceId(),
		Tags:                  old.GetTags(),
		NodeNames:             old.GetNodeNames(),
		Nodes:                 old.GetNodes(),
		RapidCommit:           old.GetRapidCommit(),
		UseEui64:              old.GetUseEui64(),
		Capacity:              old.GetCapacity(),
		UsedRatio:             old.GetUsedRatio(),
		UsedCount:             old.GetUsedCount(),
	}
	ret.SetID(old.GetId())
	ret.SetLinks(decodeLinks(old.GetLinks()))
	ret.SetCreationTimestamp(decodeTimeStr(old.GetCreationTimestamp()))
	ret.SetDeletionTimestamp(decodeTimeStr(old.GetDeletionTimestamp()))
	return ret
}

func DecodePbCReservation6(old *pbdhcp.CreateReservation) *resource.Reservation6 {
	ret := &resource.Reservation6{
		HwAddress:   old.GetHwAddress(),
		IpAddresses: []string{old.GetIpAddress()},
	}
	return ret
}

func DecodePbCReservedPool6(old *pbdhcp.CreateReserved) *resource.ReservedPool6 {
	ret := &resource.ReservedPool6{
		BeginAddress: old.GetBeginAddress(),
		EndAddress:   old.GetEndAddress(),
	}
	return ret
}
func DecodeSubnetLease4FromPbLease4(lease *pbdhcpagent.DHCPLease4) *resource.SubnetLease4 {
	lease4 := &resource.SubnetLease4{
		Address:               lease.GetAddress(),
		AddressType:           resource.AddressTypeDynamic,
		HwAddress:             lease.GetHwAddress(),
		HwAddressOrganization: lease.GetHwAddressOrganization(),
		ClientId:              lease.GetClientId(),
		ValidLifetime:         lease.GetValidLifetime(),
		Expire:                decodeTimeUnix(lease.GetExpire()),
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

func DecodeSubnetLease6FromPbLease6(lease *pbdhcpagent.DHCPLease6) *resource.SubnetLease6 {
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
		Expire:                decodeTimeUnix(lease.GetExpire()),
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
