package parser

import (
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
)

func Reservation4sFromPbDHCPReservation4s(pbReservations []*pbdhcp.Reservation4) []*resource.Reservation4 {
	reservations := make([]*resource.Reservation4, len(pbReservations))
	for i, pbReservation := range pbReservations {
		reservations[i] = &resource.Reservation4{
			HwAddress: pbReservation.GetHwAddress(),
			IpAddress: pbReservation.GetIpAddress(),
			Comment:   pbReservation.GetComment(),
		}
	}

	return reservations
}

func Reservation6sFromPbDHCPReservation6s(pbReservations []*pbdhcp.Reservation6) []*resource.Reservation6 {
	reservations := make([]*resource.Reservation6, len(pbReservations))
	for i, pbReservation := range pbReservations {
		reservations[i] = &resource.Reservation6{
			HwAddress:   pbReservation.GetHwAddress(),
			Duid:        pbReservation.GetDuid(),
			IpAddresses: pbReservation.GetIpAddresses(),
			Comment:     pbReservation.GetComment(),
		}
	}

	return reservations
}

func ReservedPool4sFromPbDHCPReservedPool4s(pbPools []*pbdhcp.ReservedPool4) []*resource.ReservedPool4 {
	pools := make([]*resource.ReservedPool4, len(pbPools))
	for i, pbPool := range pbPools {
		pools[i] = &resource.ReservedPool4{
			BeginAddress: pbPool.GetBeginAddress(),
			EndAddress:   pbPool.GetEndAddress(),
			Comment:      pbPool.GetComment(),
		}
	}

	return pools
}

func ReservedPool6sFromPbDHCPReservedPool6s(pbPools []*pbdhcp.ReservedPool6) []*resource.ReservedPool6 {
	pools := make([]*resource.ReservedPool6, len(pbPools))
	for i, pbPool := range pbPools {
		pools[i] = &resource.ReservedPool6{
			BeginAddress: pbPool.GetBeginAddress(),
			EndAddress:   pbPool.GetEndAddress(),
			Comment:      pbPool.GetComment(),
		}
	}

	return pools
}
