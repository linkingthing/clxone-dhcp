package parser

import (
	"net"
	"time"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
	restresource "github.com/linkingthing/gorest/resource"
)

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

func EncodeDhcpSubnet4(subnet *resource.Subnet4) *pbdhcp.DhcpSubnet4 {
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

func EncodeDhcpPool4(pool *resource.Pool4) *pbdhcp.DhcpPool4 {
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

func EncodeDhcpReservedPool4(pool *resource.ReservedPool4) *pbdhcp.DhcpReservedPool4 {
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

func EncodeDhcpReservation4(old *resource.Reservation4) *pbdhcp.DhcpReservation4 {
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

func EncodeDhcpSubnetLeases4(old *resource.SubnetLease4) *pbdhcp.DhcpSubnetLease4 {
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

func EncodeDhcpSubnet6(old *resource.Subnet6) *pbdhcp.DhcpSubnet6 {
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

func EncodeDhcpPool6(old *resource.Pool6) *pbdhcp.DhcpPool6 {
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

func EncodeDhcpReservedPool6(old *resource.ReservedPool6) *pbdhcp.DhcpReservedPool6 {
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

func EncodeDhcpReservation6(old *resource.Reservation6) *pbdhcp.DhcpReservation6 {
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

func EncodeDhcpSubnetLease6(old *resource.SubnetLease6) *pbdhcp.DhcpSubnetLease6 {
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
