package resource

import (
	"strings"
	"time"

	"github.com/linkingthing/cement/uuid"
	restdb "github.com/linkingthing/gorest/db"

	restresource "github.com/linkingthing/gorest/resource"
)

var TableSubnetLease6 = restdb.ResourceDBType(&SubnetLease6{})

var SubnetLease6Columns = []string{restdb.IDField, restdb.CreateTimeField, SqlColumnSubnet6,
	SqlColumnAddress, SqlColumnAddressType, SqlColumnDuid, SqlColumnHwAddress, SqlColumnHwAddressType,
	SqlColumnHwAddressSource, SqlColumnHwAddressOrganization, SqlColumnFqdnFwd, SqlColumnFqdnRev, SqlColumnHostname,
	SqlColumnIaid, SqlColumnLeaseState, SqlColumnLeaseType, SqlColumnPrefixLen, SqlColumnRequestType, SqlColumnRequestTime,
	SqlColumnValidLifetime, SqlColumnPreferredLifetime, SqlColumnExpirationTime, SqlColumnFingerprint, SqlColumnVendorId,
	SqlColumnOperatingSystem, SqlColumnClientType, SqlColumnRequestSourceAddr, SqlColumnAddressCodes, SqlColumnAddressCodeBegins,
	SqlColumnAddressCodeEnds, SqlColumnSubnet,
}

type SubnetLease6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string `json:"-" db:"ownby"`
	Address                   string `json:"address" db:"uk"`
	Duid                      string `json:"duid"`
	HwAddress                 string `json:"hwAddress"`
	HwAddressType             string `json:"hwAddressType"`
	HwAddressSource           string `json:"hwAddressSource"`
	HwAddressOrganization     string `json:"hwAddressOrganization"`
	FqdnFwd                   bool   `json:"fqdnFwd"`
	FqdnRev                   bool   `json:"fqdnRev"`
	Hostname                  string `json:"hostname"`
	Iaid                      uint32 `json:"iaid"`
	LeaseState                string `json:"leaseState"`
	LeaseType                 string `json:"leaseType"`
	PrefixLen                 uint32 `json:"prefixLen"`
	RequestType               string `json:"requestType"`
	RequestTime               string `json:"requestTime"`
	ValidLifetime             uint32 `json:"validLifetime"`
	PreferredLifetime         uint32 `json:"preferredLifetime"`
	ExpirationTime            string `json:"expirationTime"`
	Fingerprint               string `json:"fingerprint"`
	VendorId                  string `json:"vendorId"`
	OperatingSystem           string `json:"operatingSystem"`
	ClientType                string `json:"clientType"`
	RequestSourceAddr         string `json:"requestSourceAddr"`
	AddressCode               string `json:"addressCode"`
	AddressCodeBegin          uint32 `json:"addressCodeBegin"`
	AddressCodeEnd            uint32 `json:"addressCodeEnd"`
	Subnet                    string `json:"subnet"`
	AllocateMode              string `json:"allocateMode"`
}

func (l *SubnetLease6) GenCopyValues() []interface{} {
	if l.GetID() == "" {
		l.ID, _ = uuid.Gen()
	}

	return []interface{}{
		l.GetID(),
		time.Now(),
		l.Subnet6,
		l.Address,
		l.Duid,
		l.HwAddress,
		l.HwAddressType,
		l.HwAddressSource,
		l.HwAddressOrganization,
		l.FqdnFwd,
		l.FqdnRev,
		l.Hostname,
		l.Iaid,
		l.LeaseState,
		l.LeaseType,
		l.PrefixLen,
		l.RequestType,
		l.RequestTime,
		l.ValidLifetime,
		l.PreferredLifetime,
		l.ExpirationTime,
		l.Fingerprint,
		l.VendorId,
		l.OperatingSystem,
		l.ClientType,
		l.RequestSourceAddr,
		l.AddressCode,
		l.AddressCodeBegin,
		l.AddressCodeEnd,
		l.Subnet,
		l.AllocateMode,
	}
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
			Output: &ConvToReservationInput{},
		},
		{
			Name:  ActionDynamicToReservation,
			Input: &ConvToReservationInput{},
		},
	}
}
