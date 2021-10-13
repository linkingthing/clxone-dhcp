package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type Lease6 struct {
	restresource.ResourceBase `json:",inline"`
	Address                   string               `json:"address"`
	PrefixLen                 uint32               `json:"prefixLen"`
	Duid                      string               `json:"duid"`
	Iaid                      uint32               `json:"iaid"`
	PreferredLifetime         uint32               `json:"preferredLifetime"`
	ValidLifetime             uint32               `json:"validLifetime"`
	Expire                    restresource.ISOTime `json:"expire"`
	HwAddress                 string               `json:"hwAddress"`
	HwAddressType             uint32               `json:"hwAddressType"`
	HwAddressSource           uint32               `json:"hwAddressSource"`
	LeaseType                 string               `json:"leaseType"`
	Hostname                  string               `json:"hostname"`
	Fingerprint               string               `json:"fingerprint"`
	VendorId                  string               `json:"vendorId"`
	OperatingSystem           string               `json:"operatingSystem"`
	ClientType                string               `json:"clientType"`
	State                     uint32               `json:"state"`
}

func (l Lease6) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}
