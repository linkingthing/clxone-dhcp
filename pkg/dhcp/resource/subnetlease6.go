package resource

import (
	restresource "github.com/linkingthing/gorest/resource"
)

type SubnetLease6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string      `json:"-" db:"ownby"`
	Address                   string      `json:"address"`
	AddressType               AddressType `json:"addressType"`
	PrefixLen                 uint32      `json:"prefixLen"`
	Duid                      string      `json:"duid"`
	Iaid                      uint32      `json:"iaid"`
	PreferredLifetime         uint32      `json:"preferredLifetime"`
	ValidLifetime             uint32      `json:"validLifetime"`
	Expire                    string      `json:"expire"`
	HwAddress                 string      `json:"hwAddress"`
	HwAddressType             string      `json:"hwAddressType"`
	HwAddressSource           string      `json:"hwAddressSource"`
	HwAddressOrganization     string      `json:"hwAddressOrganization"`
	LeaseType                 string      `json:"leaseType"`
	Hostname                  string      `json:"hostname"`
	Fingerprint               string      `json:"fingerprint"`
	VendorId                  string      `json:"vendorId"`
	OperatingSystem           string      `json:"operatingSystem"`
	ClientType                string      `json:"clientType"`
	LeaseState                string      `json:"leaseState"`
	RequestSourceAddr         string      `json:"requestSourceAddr"`
}

func (l SubnetLease6) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}

func (l *SubnetLease6) Equal(another *SubnetLease6) bool {
	return l.Address == another.Address &&
		l.Expire == another.Expire &&
		l.Duid == another.Duid &&
		l.HwAddress == another.HwAddress &&
		l.LeaseType == another.LeaseType &&
		l.Iaid == another.Iaid
}
