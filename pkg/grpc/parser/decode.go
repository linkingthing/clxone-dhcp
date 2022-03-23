package parser

import (
	"github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
	"time"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

func decodeTimeUnix(t int64) string {
	return time.Unix(t, 0).Format(time.RFC3339)
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

func DecodePbToReservation4s(pbPools []*dhcp.Reservation4) []*resource.Reservation4 {
	pools := make([]*resource.Reservation4, len(pbPools))
	for i, pbPool := range pbPools {
		pools[i] = DecodeOnePbToReservation4(pbPool)
	}
	return pools
}

func DecodeOnePbToReservation4(pbPool *dhcp.Reservation4) *resource.Reservation4 {
	return &resource.Reservation4{
		HwAddress: pbPool.GetHwAddress(),
		IpAddress: pbPool.GetIpAddress(),
		Comment:   pbPool.GetComment(),
	}
}

func DecodePbToReservation6s(pbPools []*dhcp.Reservation6) []*resource.Reservation6 {
	pools := make([]*resource.Reservation6, len(pbPools))
	for i, pbPool := range pbPools {
		pools[i] = DecodeOnePbToReservation6(pbPool)
	}
	return pools
}

func DecodeOnePbToReservation6(pbPool *dhcp.Reservation6) *resource.Reservation6 {
	return &resource.Reservation6{
		HwAddress:   pbPool.GetHwAddress(),
		Duid:        pbPool.GetDuid(),
		IpAddresses: pbPool.GetIpAddresses(),
		Comment:     pbPool.GetComment(),
	}
}

func DecodePbToReservedPool4s(pbPools []*dhcp.ReservedPool4) []*resource.ReservedPool4 {
	pools := make([]*resource.ReservedPool4, len(pbPools))
	for i, pbPool := range pbPools {
		pools[i] = DecodeOnePbToReservedPool4(pbPool)
	}
	return pools
}

func DecodeOnePbToReservedPool4(pbPool *dhcp.ReservedPool4) *resource.ReservedPool4 {
	return &resource.ReservedPool4{
		BeginAddress: pbPool.GetBeginAddress(),
		EndAddress:   pbPool.GetEndAddress(),
		Comment:      pbPool.GetComment(),
	}
}

func DecodePbToReservedPool6s(pbPools []*dhcp.ReservedPool6) []*resource.ReservedPool6 {
	pools := make([]*resource.ReservedPool6, len(pbPools))
	for i, pbPool := range pbPools {
		pools[i] = DecodeOnePbToReservedPool6(pbPool)
	}
	return pools
}

func DecodeOnePbToReservedPool6(pbPool *dhcp.ReservedPool6) *resource.ReservedPool6 {
	return &resource.ReservedPool6{
		BeginAddress: pbPool.GetBeginAddress(),
		EndAddress:   pbPool.GetEndAddress(),
		Comment:      pbPool.GetComment(),
	}
}
