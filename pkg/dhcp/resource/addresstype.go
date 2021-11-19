package resource

type AddressType string

const (
	AddressTypeDynamic     AddressType = "dynamic"
	AddressTypeReservation AddressType = "reservation"
	AddressTypeReserve     AddressType = "reserve"
	AddressTypeExclusion   AddressType = "exclusion"
	AddressTypeDelegation  AddressType = "delegation"
)
