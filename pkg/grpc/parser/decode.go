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

func DecodePbReservation4(old *pbdhcp.DhcpReservation4) *resource.Reservation4 {
	ret := &resource.Reservation4{
		Subnet4:   old.GetSubnet4(),
		HwAddress: old.GetHwAddress(),
		IpAddress: old.GetIpAddress(),
		Ip:        decodeIp(old.GetIp()),
		UsedRatio: old.GetUsedRatio(),
		UsedCount: old.GetUsedCount(),
		Capacity:  old.GetCapacity(),
		Comment:   old.GetComment(),
	}
	ret.SetID(old.GetId())
	ret.SetLinks(decodeLinks(old.GetLinks()))
	ret.SetCreationTimestamp(decodeTimeStr(old.GetCreationTimestamp()))
	ret.SetDeletionTimestamp(decodeTimeStr(old.GetDeletionTimestamp()))
	return ret
}

func DecodePbReservedPool4(old *pbdhcp.DhcpReservedPool4) *resource.ReservedPool4 {
	ret := &resource.ReservedPool4{
		Subnet4:      old.GetSubnet4(),
		BeginAddress: old.GetBeginAddress(),
		BeginIp:      decodeIp(old.GetBeginIp()),
		EndAddress:   old.GetEndAddress(),
		EndIp:        decodeIp(old.GetEndIp()),
		Capacity:     old.GetCapacity(),
		UsedRatio:    old.GetUsedRatio(),
		UsedCount:    old.GetUsedCount(),
		Template:     old.GetTemplate(),
		Comment:      old.GetComment(),
	}
	ret.SetID(old.GetId())
	ret.SetLinks(decodeLinks(old.GetLinks()))
	ret.SetCreationTimestamp(decodeTimeStr(old.GetCreationTimestamp()))
	ret.SetDeletionTimestamp(decodeTimeStr(old.GetDeletionTimestamp()))
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

func DecodePbReservation6(old *pbdhcp.DhcpReservation6) *resource.Reservation6 {
	ret := &resource.Reservation6{
		Subnet6:     old.GetSubnet6(),
		Duid:        old.GetDUid(),
		HwAddress:   old.GetHwAddress(),
		IpAddresses: old.GetIpAddresses(),
		Ips:         decodeIps(old.GetIps()),
		Prefixes:    old.GetPrefixes(),
		Capacity:    old.GetCapacity(),
		UsedRatio:   old.GetUsedRatio(),
		UsedCount:   old.GetUsedCount(),
		Comment:     old.GetComment(),
	}
	ret.SetID(old.GetId())
	ret.SetLinks(decodeLinks(old.GetLinks()))
	ret.SetCreationTimestamp(decodeTimeStr(old.GetCreationTimestamp()))
	ret.SetDeletionTimestamp(decodeTimeStr(old.GetDeletionTimestamp()))
	return ret
}

func DecodePbReservedPool6(old *pbdhcp.DhcpReservedPool6) *resource.ReservedPool6 {
	ret := &resource.ReservedPool6{
		Subnet6:      old.GetSubnet6(),
		BeginAddress: old.GetBeginAddress(),
		BeginIp:      decodeIp(old.GetBeginIp()),
		EndAddress:   old.GetEndAddress(),
		EndIp:        decodeIp(old.GetEndIp()),
		Capacity:     old.GetCapacity(),
		UsedRatio:    old.GetUsedRatio(),
		UsedCount:    old.GetUsedCount(),
		Template:     old.GetTemplate(),
		Comment:      old.GetComment(),
	}
	ret.SetID(old.GetId())
	ret.SetLinks(decodeLinks(old.GetLinks()))
	ret.SetCreationTimestamp(decodeTimeStr(old.GetCreationTimestamp()))
	ret.SetDeletionTimestamp(decodeTimeStr(old.GetDeletionTimestamp()))
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
