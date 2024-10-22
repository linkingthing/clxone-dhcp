package resource

import (
	"fmt"
	"strings"
	"time"

	"github.com/linkingthing/cement/uuid"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var SubnetLease4Columns = []string{restdb.IDField, restdb.CreateTimeField, SqlColumnSubnet4,
	SqlColumnAddress, SqlColumnAddressType, SqlColumnHwAddress, SqlColumnHwAddressOrganization, SqlColumnClientId,
	SqlColumnFqdnFwd, SqlColumnFqdnRev, SqlColumnHostname, SqlColumnLeaseState, SqlColumnRequestType, SqlColumnRequestTime,
	SqlColumnValidLifetime, SqlColumnExpirationTime, SqlColumnFingerprint, SqlColumnVendorId, SqlColumnOperatingSystem,
	SqlColumnClientType, SqlColumnSubnet,
}

type ReservationType string

const (
	ReservationTypeMac      ReservationType = "mac"
	ReservationTypeHostname ReservationType = "hostname"
	ReservationTypeDuid     ReservationType = "duid"
)

var TableSubnetLease4 = restdb.ResourceDBType(&SubnetLease4{})

type SubnetLease4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet4                   string      `json:"-" db:"ownby"`
	Address                   string      `json:"address" db:"uk"`
	AddressType               AddressType `json:"addressType"`
	HwAddress                 string      `json:"hwAddress"`
	HwAddressOrganization     string      `json:"hwAddressOrganization"`
	ClientId                  string      `json:"clientId"`
	FqdnFwd                   bool        `json:"fqdnFwd"`
	FqdnRev                   bool        `json:"fqdnRev"`
	Hostname                  string      `json:"hostname"`
	LeaseState                string      `json:"leaseState"`
	RequestType               string      `json:"requestType"`
	RequestTime               string      `json:"requestTime"`
	ValidLifetime             uint32      `json:"validLifetime"`
	ExpirationTime            string      `json:"expirationTime"`
	Fingerprint               string      `json:"fingerprint"`
	VendorId                  string      `json:"vendorId"`
	OperatingSystem           string      `json:"operatingSystem"`
	ClientType                string      `json:"clientType"`
	Subnet                    string      `json:"subnet"`
}

func (l *SubnetLease4) GenCopyValues() []interface{} {
	if l.GetID() == "" {
		l.ID, _ = uuid.Gen()
	}
	return []interface{}{
		l.GetID(),
		time.Now(),
		l.Subnet4,
		l.Address,
		l.AddressType,
		l.HwAddress,
		l.HwAddressOrganization,
		l.ClientId,
		l.FqdnFwd,
		l.FqdnRev,
		l.Hostname,
		l.LeaseState,
		l.RequestType,
		l.RequestTime,
		l.ValidLifetime,
		l.ExpirationTime,
		l.Fingerprint,
		l.VendorId,
		l.OperatingSystem,
		l.ClientType,
		l.Subnet,
	}
}

func (l SubnetLease4) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet4{}}
}

func (l *SubnetLease4) Equal(another *SubnetLease4) bool {
	return l.Address == another.Address &&
		l.ExpirationTime == another.ExpirationTime &&
		strings.EqualFold(l.HwAddress, another.HwAddress) &&
		l.ClientId == another.ClientId &&
		l.Hostname == another.Hostname
}

func (l *SubnetLease4) GetUniqueKey() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", l.Address, l.ExpirationTime, strings.ToUpper(l.HwAddress), l.ClientId, l.Hostname)
}

func (s SubnetLease4) GetActions() []restresource.Action {
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

type BatchDeleteLeasesInput struct {
	Addresses []string `json:"addresses"`
}

type ConvToReservationInput struct {
	Addresses       []string                `json:"addresses"`
	ReservationType ReservationType         `json:"reservationType"`
	BothV4V6        bool                    `json:"bothV4V6"`
	Data            []ConvToReservationItem `json:"data"`
}

type ConvToReservationItem struct {
	Address    string   `json:"address"`
	DualStacks []string `json:"dualStacks"`
	HwAddress  string   `json:"hwAddress"`
	Hostname   string   `json:"hostname"`
	Duid       string   `json:"duid"`
}
