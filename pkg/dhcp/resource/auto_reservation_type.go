package resource

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
