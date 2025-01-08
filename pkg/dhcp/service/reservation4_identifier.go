package service

import (
	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type Reservation4Identifier struct {
	Macs      map[string]struct{}
	Hostnames map[string]struct{}
	Ips       map[uint32]struct{}
}

func Reservation4IdentifierFromReservations(reservations []*resource.Reservation4) *Reservation4Identifier {
	reservation4Identifier := &Reservation4Identifier{
		Macs:      make(map[string]struct{}, len(reservations)),
		Hostnames: make(map[string]struct{}, len(reservations)),
		Ips:       make(map[uint32]struct{}, len(reservations)),
	}

	for _, reservation := range reservations {
		if reservation.HwAddress != "" {
			reservation4Identifier.Macs[reservation.HwAddress] = struct{}{}
		}

		if reservation.Hostname != "" {
			reservation4Identifier.Hostnames[reservation.Hostname] = struct{}{}
		}

		reservation4Identifier.Ips[gohelperip.IPv4ToUint32(reservation.Ip)] = struct{}{}
	}

	return reservation4Identifier
}

func (r *Reservation4Identifier) Add(reservation *resource.Reservation4) error {
	if reservation.HwAddress != "" {
		if _, ok := r.Macs[reservation.HwAddress]; ok {
			return errorno.ErrUsedReservation(reservation.HwAddress)
		} else {
			r.Macs[reservation.HwAddress] = struct{}{}
		}
	}

	if reservation.Hostname != "" {
		if _, ok := r.Hostnames[reservation.Hostname]; ok {
			return errorno.ErrUsedReservation(reservation.Hostname)
		} else {
			r.Hostnames[reservation.Hostname] = struct{}{}
		}
	}

	ipUint32 := gohelperip.IPv4ToUint32(reservation.Ip)
	if _, ok := r.Ips[ipUint32]; ok {
		return errorno.ErrUsedReservation(reservation.IpAddress)
	} else {
		r.Ips[ipUint32] = struct{}{}
	}

	return nil
}
