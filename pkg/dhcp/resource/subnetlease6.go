package resource

import (
	"strings"

	restresource "github.com/linkingthing/gorest/resource"
)

type SubnetLease6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string      `json:"-" db:"ownby"`
	Address                   string      `json:"address" db:"uk"`
	AddressType               AddressType `json:"addressType"`
	Duid                      string      `json:"duid"`
	HwAddress                 string      `json:"hwAddress"`
	HwAddressType             string      `json:"hwAddressType"`
	HwAddressSource           string      `json:"hwAddressSource"`
	HwAddressOrganization     string      `json:"hwAddressOrganization"`
	FqdnFwd                   bool        `json:"fqdnFwd"`
	FqdnRev                   bool        `json:"fqdnRev"`
	Hostname                  string      `json:"hostname"`
	Iaid                      uint32      `json:"iaid"`
	LeaseState                string      `json:"leaseState"`
	LeaseType                 string      `json:"leaseType"`
	PrefixLen                 uint32      `json:"prefixLen"`
	RequestType               string      `json:"requestType"`
	RequestTime               string      `json:"requestTime"`
	ValidLifetime             uint32      `json:"validLifetime"`
	PreferredLifetime         uint32      `json:"preferredLifetime"`
	ExpirationTime            string      `json:"expirationTime"`
	Fingerprint               string      `json:"fingerprint"`
	VendorId                  string      `json:"vendorId"`
	OperatingSystem           string      `json:"operatingSystem"`
	ClientType                string      `json:"clientType"`
	RequestSourceAddr         string      `json:"requestSourceAddr"`
	AddressCodes              []string    `json:"addressCodes"`
	AddressCodeBegins         []uint32    `json:"addressCodeBegins"`
	AddressCodeEnds           []uint32    `json:"addressCodeEnds"`
	Subnet                    string      `json:"subnet"`
	BelongEui64Subnet         bool        `json:"-" db:"-"`
}

func (l SubnetLease6) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}

func (l *SubnetLease6) Equal(another *SubnetLease6) bool {
	return l.Address == another.Address &&
		l.ExpirationTime == another.ExpirationTime &&
		l.Duid == another.Duid &&
		strings.EqualFold(l.HwAddress, another.HwAddress) &&
		l.LeaseType == another.LeaseType &&
		l.Iaid == another.Iaid &&
		l.Hostname == another.Hostname
}

func (s SubnetLease6) GetActions() []restresource.Action {
	return []restresource.Action{
		{
			Name:  ActionBatchDelete,
			Input: &BatchDeleteLeasesInput{},
		},
		{
			Name:   ActionListToReservation,
			Input:  &ConvToReservationInput{},
			Output: &ConvToReservationOutput{},
		},
		{
			Name:  ActionDynamicToReservation,
			Input: &ConvToReservationInput{},
		},
	}
}
