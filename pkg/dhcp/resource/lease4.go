package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

type AddressType string

const (
	AddressTypeDynamic     AddressType = "dynamic"
	AddressTypeReservation AddressType = "reservation"
)

var TableSubnetLease4 = restdb.ResourceDBType(&SubnetLease4{})

type SubnetLease4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet4                   string      `json:"-" db:"ownby"`
	Address                   string      `json:"address"`
	AddressType               AddressType `json:"addressType"`
	HwAddress                 string      `json:"hwAddress"`
	ClientId                  string      `json:"clientId"`
	ValidLifetime             uint32      `json:"validLifetime"`
	Expire                    string      `json:"expire"`
	Hostname                  string      `json:"hostname"`
	Fingerprint               string      `json:"fingerprint"`
	VendorId                  string      `json:"vendorId"`
	OperatingSystem           string      `json:"operatingSystem"`
	ClientType                string      `json:"clientType"`
	LeaseState                string      `json:"leaseState"`
}

func (l SubnetLease4) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet4{}}
}

func (l *SubnetLease4) Equal(another *SubnetLease4) bool {
	return l.Address == another.Address &&
		l.Expire == another.Expire &&
		l.HwAddress == another.HwAddress &&
		l.ClientId == another.ClientId
}
