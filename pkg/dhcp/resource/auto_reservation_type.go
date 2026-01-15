package resource

import "fmt"

const (
	AutoReservationTypeNone     = 0
	AutoReservationTypeMac      = 1
	AutoReservationTypeHostname = 2
	AutoReservationTypeDuid     = 3
)

func ValidateAutoReservationType(typ uint32, isV4 bool) bool {
	switch typ {
	case AutoReservationTypeNone, AutoReservationTypeMac, AutoReservationTypeHostname:
		return true
	case AutoReservationTypeDuid:
		return !isV4
	default:
		return false
	}
}

const (
	AutoReservationNameNone     = "不开启"
	AutoReservationNameMAC      = "MAC固定"
	AutoReservationNameHostname = "主机名固定"
	AutoReservationNameDuid     = "DUID固定"
)

func AutoReservationTypeToString(typ uint32) string {
	switch typ {
	case AutoReservationTypeMac:
		return AutoReservationNameMAC
	case AutoReservationTypeHostname:
		return AutoReservationNameHostname
	case AutoReservationTypeDuid:
		return AutoReservationNameDuid
	default:
		return AutoReservationNameNone
	}
}

func AutoReservationTypeFromString(typ string) (uint32, error) {
	switch typ {
	case AutoReservationNameNone:
		return AutoReservationTypeNone, nil
	case AutoReservationNameMAC:
		return AutoReservationTypeMac, nil
	case AutoReservationNameHostname:
		return AutoReservationTypeHostname, nil
	case AutoReservationNameDuid:
		return AutoReservationTypeDuid, nil
	default:
		return AutoReservationTypeNone, fmt.Errorf("unsupported auto reservation type %s", typ)
	}
}
