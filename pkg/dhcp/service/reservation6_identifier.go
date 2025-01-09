package service

import (
	"encoding/binary"
	"net"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type Reservation6Identifier struct {
	Duids     map[string]struct{}
	Macs      map[string]struct{}
	Hostnames map[string]struct{}
	Ips       map[uint64]struct{}
	Prefixes  map[string]struct{}
}

func Reservation6IdentifierFromReservations(reservations []*resource.Reservation6) *Reservation6Identifier {
	reservation6Identifier := &Reservation6Identifier{
		Duids:     make(map[string]struct{}, len(reservations)),
		Macs:      make(map[string]struct{}, len(reservations)),
		Hostnames: make(map[string]struct{}, len(reservations)),
		Ips:       make(map[uint64]struct{}, len(reservations)),
		Prefixes:  make(map[string]struct{}, len(reservations)),
	}

	for _, reservation := range reservations {
		if reservation.Duid != "" {
			reservation6Identifier.Duids[reservation.Duid] = struct{}{}
		}

		if reservation.HwAddress != "" {
			reservation6Identifier.Macs[reservation.HwAddress] = struct{}{}
		}

		if reservation.Hostname != "" {
			reservation6Identifier.Hostnames[reservation.Hostname] = struct{}{}
		}

		for _, ip := range reservation.Ips {
			reservation6Identifier.Ips[ipv6ToUint64(ip)] = struct{}{}
		}

		for _, prefix := range reservation.Prefixes {
			reservation6Identifier.Prefixes[prefix] = struct{}{}
		}
	}

	return reservation6Identifier
}

func ipv6ToUint64(ip net.IP) uint64 {
	return binary.BigEndian.Uint64(ip.To16()[8:])
}

func (r *Reservation6Identifier) Add(reservation *resource.Reservation6) error {
	if reservation.Duid != "" {
		if _, ok := r.Duids[reservation.Duid]; ok {
			return errorno.ErrUsedReservation(reservation.Duid)
		} else {
			r.Duids[reservation.Duid] = struct{}{}
		}
	}

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

	for _, ip := range reservation.Ips {
		ipUint64 := ipv6ToUint64(ip)
		if _, ok := r.Ips[ipUint64]; ok {
			return errorno.ErrUsedReservation(ip.String())
		} else {
			r.Ips[ipUint64] = struct{}{}
		}
	}

	for _, prefix := range reservation.Prefixes {
		if _, ok := r.Prefixes[prefix]; ok {
			return errorno.ErrUsedReservation(prefix)
		} else {
			r.Prefixes[prefix] = struct{}{}
		}
	}

	return nil
}
