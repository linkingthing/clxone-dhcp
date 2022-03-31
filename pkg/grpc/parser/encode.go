package parser

import (
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcppb "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
)

func EncodeSubnet4sToPb(subnet4s []*resource.Subnet4) []*dhcppb.Subnet4 {
	pbSubnets := make([]*dhcppb.Subnet4, len(subnet4s))
	for i, subnet := range subnet4s {
		pbSubnets[i] = EncodeOneSubnet4ToPb(subnet)
	}
	return pbSubnets
}

func EncodeOneSubnet4ToPb(subnet *resource.Subnet4) *dhcppb.Subnet4 {
	return &dhcppb.Subnet4{
		Subnet:        subnet.Subnet,
		Capacity:      subnet.Capacity,
		UsedCount:     subnet.UsedCount,
		DomainServers: subnet.DomainServers,
		Routers:       subnet.Routers,
	}
}

func EncodePool4sToPb(pools []*resource.Pool4) []*dhcppb.Pool4 {
	pbPools := make([]*dhcppb.Pool4, len(pools))
	for i, pool4 := range pools {
		pbPools[i] = EncodeOnePool4ToPb(pool4)
	}
	return pbPools
}

func EncodeOnePool4ToPb(pool *resource.Pool4) *dhcppb.Pool4 {
	return &dhcppb.Pool4{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
		Capacity:     pool.Capacity,
		UsedCount:    pool.UsedCount,
		Comment:      pool.Comment,
	}
}

func EncodeReservedPool4sToPb(pools []*resource.ReservedPool4) []*dhcppb.ReservedPool4 {
	pbPools := make([]*dhcppb.ReservedPool4, len(pools))
	for i, reservedPool4 := range pools {
		pbPools[i] = EncodeOneReservedPool4ToPb(reservedPool4)
	}
	return pbPools
}

func EncodeOneReservedPool4ToPb(pool *resource.ReservedPool4) *dhcppb.ReservedPool4 {
	return &dhcppb.ReservedPool4{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
		Capacity:     pool.Capacity,
		Comment:      pool.Comment,
	}
}

func EncodeReservation4sToPb(pools []*resource.Reservation4) []*dhcppb.Reservation4 {
	pbPools := make([]*dhcppb.Reservation4, len(pools))
	for i, reservation4 := range pools {
		pbPools[i] = EncodeOneReservation4ToPb(reservation4)
	}
	return pbPools
}

func EncodeOneReservation4ToPb(pool *resource.Reservation4) *dhcppb.Reservation4 {
	return &dhcppb.Reservation4{
		HwAddress: pool.HwAddress,
		IpAddress: pool.IpAddress,
		Capacity:  pool.Capacity,
		UsedCount: pool.UsedCount,
		Comment:   pool.Comment,
	}
}

func EncodeSubnetLeases4sToPb(lease4s []*resource.SubnetLease4) []*dhcppb.Lease4 {
	pbLease4s := make([]*dhcppb.Lease4, len(lease4s))
	for i, lease4 := range lease4s {
		pbLease4s[i] = EncodeOneSubnetLeases4ToPb(lease4)
	}
	return pbLease4s
}

func EncodeOneSubnetLeases4ToPb(lease4 *resource.SubnetLease4) *dhcppb.Lease4 {
	return &dhcppb.Lease4{
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
		AddressType:           string(lease4.AddressType),
	}
}

func EncodeSubnet6sToPb(subnet6s []*resource.Subnet6) []*dhcppb.Subnet6 {
	pbSubnets := make([]*dhcppb.Subnet6, len(subnet6s))
	for i, subnet := range subnet6s {
		pbSubnets[i] = EncodeOneSubnet6ToPb(subnet)
	}
	return pbSubnets
}

func EncodeOneSubnet6ToPb(subnet *resource.Subnet6) *dhcppb.Subnet6 {
	return &dhcppb.Subnet6{
		Subnet:        subnet.Subnet,
		Capacity:      subnet.Capacity,
		UsedCount:     subnet.UsedCount,
		DomainServers: subnet.DomainServers,
		UseEui64:      subnet.UseEui64,
	}
}

func EncodePool6sToPb(pools []*resource.Pool6) []*dhcppb.Pool6 {
	pbPools := make([]*dhcppb.Pool6, len(pools))
	for i, pool6 := range pools {
		pbPools[i] = EncodeOnePool6ToPb(pool6)
	}
	return pbPools
}

func EncodeOnePool6ToPb(pool *resource.Pool6) *dhcppb.Pool6 {
	return &dhcppb.Pool6{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
		Capacity:     pool.Capacity,
		UsedCount:    pool.UsedCount,
		Comment:      pool.Comment,
	}
}

func EncodeReservedPool6sToPb(pools []*resource.ReservedPool6) []*dhcppb.ReservedPool6 {
	pbPools := make([]*dhcppb.ReservedPool6, len(pools))
	for i, pool6 := range pools {
		pbPools[i] = EncodeOneReservedPool6ToPb(pool6)
	}
	return pbPools
}

func EncodeOneReservedPool6ToPb(pool *resource.ReservedPool6) *dhcppb.ReservedPool6 {
	return &dhcppb.ReservedPool6{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
		Capacity:     pool.Capacity,
		Comment:      pool.Comment,
	}
}

func EncodeReservation6sToPb(pools []*resource.Reservation6) []*dhcppb.Reservation6 {
	pbPools := make([]*dhcppb.Reservation6, len(pools))
	for i, reservation6 := range pools {
		pbPools[i] = EncodeOneReservation6ToPb(reservation6)
	}
	return pbPools
}

func EncodeOneReservation6ToPb(pool *resource.Reservation6) *dhcppb.Reservation6 {
	return &dhcppb.Reservation6{
		HwAddress:   pool.HwAddress,
		IpAddresses: pool.IpAddresses,
		Capacity:    pool.Capacity,
		UsedCount:   pool.UsedCount,
		Comment:     pool.Comment,
	}
}

func EncodePdPool6sToPb(pools []*resource.PdPool) []*dhcppb.PdPool6 {
	pbPools := make([]*dhcppb.PdPool6, len(pools))
	for i, pool := range pools {
		pbPools[i] = EncodeOnePdPoolToPb(pool)
	}
	return pbPools
}

func EncodeOnePdPoolToPb(pool *resource.PdPool) *dhcppb.PdPool6 {
	return &dhcppb.PdPool6{
		Prefix:       pool.Prefix,
		PrefixLen:    pool.PrefixLen,
		PrefixIpnet:  pool.PrefixIpnet.String(),
		DelegatedLen: pool.DelegatedLen,
		Capacity:     pool.Capacity,
		Comment:      pool.Comment,
	}
}

func EncodeSubnetLease6sToPb(lease6s []*resource.SubnetLease6) []*dhcppb.Lease6 {
	pbLease6s := make([]*dhcppb.Lease6, len(lease6s))
	for i, lease6 := range lease6s {
		pbLease6s[i] = EncodeOneSubnetLease6ToPb(lease6)
	}
	return pbLease6s
}

func EncodeOneSubnetLease6ToPb(lease6 *resource.SubnetLease6) *dhcppb.Lease6 {
	return &dhcppb.Lease6{
		Address:               lease6.Address,
		PrefixLen:             lease6.PrefixLen,
		Duid:                  lease6.Duid,
		Iaid:                  lease6.Iaid,
		PreferredLifetime:     lease6.PreferredLifetime,
		ValidLifetime:         lease6.ValidLifetime,
		Expire:                lease6.Expire,
		HwAddress:             lease6.HwAddress,
		HwAddressType:         lease6.HwAddressType,
		HwAddressSource:       lease6.HwAddressSource,
		HwAddressOrganization: lease6.HwAddressOrganization,
		LeaseType:             lease6.LeaseType,
		Hostname:              lease6.Hostname,
		VendorId:              lease6.VendorId,
		OperatingSystem:       lease6.OperatingSystem,
		ClientType:            lease6.ClientType,
		LeaseState:            lease6.LeaseState,
		AddressType:           string(lease6.AddressType),
	}
}

func EncodeDhcpSubnet4FromSubnet4(subnet *resource.Subnet4, leasesCount uint64) *dhcppb.Subnet4 {
	return &dhcppb.Subnet4{
		Subnet:        subnet.Subnet,
		SubnetId:      subnet.SubnetId,
		Capacity:      subnet.Capacity,
		UsedCount:     leasesCount,
		DomainServers: subnet.DomainServers,
		Routers:       subnet.Routers,
		Id:            subnet.GetID(),
	}
}

func EncodeDhcpSubnet6FromSubnet6(subnet *resource.Subnet6, leasesCount uint64) *dhcppb.Subnet6 {
	return &dhcppb.Subnet6{
		Subnet:        subnet.Subnet,
		SubnetId:      subnet.SubnetId,
		Capacity:      subnet.Capacity,
		UsedCount:     leasesCount,
		DomainServers: subnet.DomainServers,
		UseEui64:      subnet.UseEui64,
		Id:            subnet.GetID(),
	}
}

func EncodeDhcpLease4FromSubnetLease4(lease4 *resource.SubnetLease4) *dhcppb.Lease4 {
	if lease4 == nil {
		return nil
	}
	return &dhcppb.Lease4{
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

func EncodeDhcpLease6FromSubnetLease6(lease6 *resource.SubnetLease6) *dhcppb.Lease6 {
	if lease6 == nil {
		return nil
	}

	return &dhcppb.Lease6{
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
