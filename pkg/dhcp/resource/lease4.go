package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type Lease4 struct {
	restresource.ResourceBase `json:",inline"`
	Address                   string               `json:"address"`
	HwAddress                 string               `json:"hwAddress"`
	ClientId                  string               `json:"clientId"`
	ValidLifetime             uint32               `json:"validLifetime"`
	Expire                    restresource.ISOTime `json:"expire"`
	Hostname                  string               `json:"hostname"`
	Fingerprint               string               `json:"fingerprint"`
	VendorId                  string               `json:"vendorId"`
	OperatingSystem           string               `json:"operatingSystem"`
	ClientType                string               `json:"clientType"`
	State                     uint32               `json:"state"`
}

func (l Lease4) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet4{}}
}
