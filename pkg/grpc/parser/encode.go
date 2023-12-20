package parser

import (
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
)

func Subnet4sToPbDHCPSubnet4s(subnets []*resource.Subnet4) []*pbdhcp.Subnet4 {
	pbSubnets := make([]*pbdhcp.Subnet4, len(subnets))
	for i, subnet := range subnets {
		pbSubnets[i] = Subnet4ToPbDHCPSubnet4(subnet, subnet.UsedCount)
	}

	return pbSubnets
}

func Subnet4ToPbDHCPSubnet4(subnet *resource.Subnet4, leasesCount uint64) *pbdhcp.Subnet4 {
	return &pbdhcp.Subnet4{
		Id:            subnet.GetID(),
		Subnet:        subnet.Subnet,
		SubnetId:      subnet.SubnetId,
		Capacity:      subnet.Capacity,
		DomainServers: subnet.DomainServers,
		Routers:       subnet.Routers,
		UsedCount:     leasesCount,
	}
}

func Pool4sToPbDHCPPool4s(pools []*resource.Pool4) []*pbdhcp.Pool4 {
	pbPools := make([]*pbdhcp.Pool4, len(pools))
	for i, pool := range pools {
		pbPools[i] = &pbdhcp.Pool4{
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
			Capacity:     pool.Capacity,
			UsedCount:    pool.UsedCount,
			Comment:      pool.Comment,
		}
	}

	return pbPools
}

func ReservedPool4sToPbDHCPReservedPool4s(pools []*resource.ReservedPool4) []*pbdhcp.ReservedPool4 {
	pbPools := make([]*pbdhcp.ReservedPool4, len(pools))
	for i, pool := range pools {
		pbPools[i] = &pbdhcp.ReservedPool4{
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
			Capacity:     pool.Capacity,
			Comment:      pool.Comment,
		}
	}

	return pbPools
}

func Reservation4sToPbDHCPReservation4s(reservations []*resource.Reservation4) []*pbdhcp.Reservation4 {
	pbReservations := make([]*pbdhcp.Reservation4, len(reservations))
	for i, reservation := range reservations {
		pbReservations[i] = &pbdhcp.Reservation4{
			HwAddress: reservation.HwAddress,
			IpAddress: reservation.IpAddress,
			Capacity:  reservation.Capacity,
			UsedCount: reservation.UsedCount,
			Comment:   reservation.Comment,
		}
	}

	return pbReservations
}

func SubnetLeases4sToPbDHCPLease4s(leases []*resource.SubnetLease4) []*pbdhcp.Lease4 {
	pbLease4s := make([]*pbdhcp.Lease4, len(leases))
	for i, lease := range leases {
		pbLease4s[i] = SubnetLease4ToPbDHCPLease4(lease)
	}

	return pbLease4s
}

func SubnetLease4ToPbDHCPLease4(lease *resource.SubnetLease4) *pbdhcp.Lease4 {
	if lease == nil {
		return nil
	}

	return &pbdhcp.Lease4{
		Address:               lease.Address,
		HwAddress:             lease.HwAddress,
		HwAddressOrganization: lease.HwAddressOrganization,
		ClientId:              lease.ClientId,
		ValidLifetime:         lease.ValidLifetime,
		Expire:                lease.ExpirationTime,
		Hostname:              lease.Hostname,
		VendorId:              lease.VendorId,
		OperatingSystem:       lease.OperatingSystem,
		ClientType:            lease.ClientType,
		LeaseState:            lease.LeaseState,
		AddressType:           lease.AddressType.String(),
	}
}

func Subnet6sToPbDHCPSubnet6s(subnets []*resource.Subnet6) []*pbdhcp.Subnet6 {
	pbSubnets := make([]*pbdhcp.Subnet6, len(subnets))
	for i, subnet := range subnets {
		pbSubnets[i] = Subnet6ToPbDHCPSubnet6(subnet, subnet.UsedCount)
	}

	return pbSubnets
}

func Subnet6ToPbDHCPSubnet6(subnet *resource.Subnet6, leasesCount uint64) *pbdhcp.Subnet6 {
	return &pbdhcp.Subnet6{
		Id:            subnet.GetID(),
		SubnetId:      subnet.SubnetId,
		Subnet:        subnet.Subnet,
		Capacity:      subnet.Capacity,
		DomainServers: subnet.DomainServers,
		UseEui64:      subnet.UseEui64,
		UsedCount:     leasesCount,
		AddressCode:   subnet.AddressCode,
	}
}

func Pool6sToPbDHCPPool6s(pools []*resource.Pool6) []*pbdhcp.Pool6 {
	pbPools := make([]*pbdhcp.Pool6, len(pools))
	for i, pool := range pools {
		pbPools[i] = &pbdhcp.Pool6{
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
			Capacity:     pool.Capacity,
			UsedCount:    pool.UsedCount,
			Comment:      pool.Comment,
		}
	}

	return pbPools
}

func ReservedPool6sToPbDHCPReservedPool6s(pools []*resource.ReservedPool6) []*pbdhcp.ReservedPool6 {
	pbPools := make([]*pbdhcp.ReservedPool6, len(pools))
	for i, pool := range pools {
		pbPools[i] = &pbdhcp.ReservedPool6{
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
			Capacity:     pool.Capacity,
			Comment:      pool.Comment,
		}
	}

	return pbPools
}

func Reservation6sToPbDHCPReservation6s(reservations []*resource.Reservation6) []*pbdhcp.Reservation6 {
	pbReservations := make([]*pbdhcp.Reservation6, len(reservations))
	for i, reservation := range reservations {
		pbReservations[i] = &pbdhcp.Reservation6{
			HwAddress:   reservation.HwAddress,
			IpAddresses: reservation.IpAddresses,
			Capacity:    reservation.Capacity,
			UsedCount:   reservation.UsedCount,
			Comment:     reservation.Comment,
		}
	}

	return pbReservations
}

func PdPool6sToPbDHCPPdPools(pdpools []*resource.PdPool) []*pbdhcp.PdPool6 {
	pbPdPools := make([]*pbdhcp.PdPool6, len(pdpools))
	for i, pdpool := range pdpools {
		pbPdPools[i] = &pbdhcp.PdPool6{
			Prefix:       pdpool.Prefix,
			PrefixLen:    pdpool.PrefixLen,
			PrefixIpnet:  pdpool.PrefixIpnet.String(),
			DelegatedLen: pdpool.DelegatedLen,
			Capacity:     pdpool.Capacity,
			Comment:      pdpool.Comment,
		}
	}

	return pbPdPools
}

func SubnetLease6sToPbDHCPLease6s(leases []*resource.SubnetLease6) []*pbdhcp.Lease6 {
	pbLeases := make([]*pbdhcp.Lease6, len(leases))
	for i, lease := range leases {
		pbLeases[i] = SubnetLease6ToPbDHCPLease6(lease)
	}
	return pbLeases
}

func SubnetLease6ToPbDHCPLease6(lease *resource.SubnetLease6) *pbdhcp.Lease6 {
	if lease == nil {
		return nil
	}

	return &pbdhcp.Lease6{
		Address:               lease.Address,
		PrefixLen:             lease.PrefixLen,
		Duid:                  lease.Duid,
		Iaid:                  lease.Iaid,
		PreferredLifetime:     lease.PreferredLifetime,
		ValidLifetime:         lease.ValidLifetime,
		Expire:                lease.ExpirationTime,
		HwAddress:             lease.HwAddress,
		HwAddressType:         lease.HwAddressType,
		HwAddressSource:       lease.HwAddressSource,
		HwAddressOrganization: lease.HwAddressOrganization,
		LeaseType:             lease.LeaseType,
		Hostname:              lease.Hostname,
		VendorId:              lease.VendorId,
		OperatingSystem:       lease.OperatingSystem,
		ClientType:            lease.ClientType,
		LeaseState:            lease.LeaseState,
		AddressType:           lease.AddressType.String(),
		AddressCodes:          lease.AddressCodes,
		AddressCodeBegins:     lease.AddressCodeBegins,
		AddressCodeEnds:       lease.AddressCodeEnds,
	}
}
