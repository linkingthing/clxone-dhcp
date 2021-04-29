package resource

import (
	"fmt"
	"net"

	agentutil "github.com/linkingthing/ddi-agent/pkg/dhcp/util"
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableReservation = restdb.ResourceDBType(&Reservation{})

type Reservation struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string         `json:"-" db:"ownby"`
	HwAddress                 string         `json:"hwAddress" rest:"required=true" db:"uk"`
	IpAddress                 string         `json:"ipAddress" rest:"required=true"`
	DomainServers             []string       `json:"domainServers"`
	Routers                   []string       `json:"routers"`
	Capacity                  uint64         `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string         `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64         `json:"usedCount" rest:"description=readonly" db:"-"`
	Version                   util.IPVersion `json:"version" rest:"description=readonly"`
}

func (r Reservation) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet{}}
}

func (r *Reservation) String() string {
	return r.HwAddress + "-" + r.IpAddress
}

type Reservations []*Reservation

func (r Reservations) Len() int {
	return len(r)
}

func (r Reservations) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Reservations) Less(i, j int) bool {
	return agentutil.OneIpLessThanAnother(r[i].IpAddress, r[j].IpAddress)
}

func (r *Reservation) Validate() error {
	if version, err := checkMacAndIp(r.HwAddress, r.IpAddress); err != nil {
		return fmt.Errorf("reservation is invalid %s", err.Error())
	} else {
		r.Version = version
	}

	return checkCommonOptions(r.Version, "", r.DomainServers, r.Routers)
}

func checkMacAndIp(mac, ip string) (util.IPVersion, error) {
	if _, err := net.ParseMAC(mac); err != nil {
		return util.IPVersion(0), fmt.Errorf("hwaddress %s is invalid", mac)
	}

	version := util.IPVersion4
	if _, isv4, err := util.ParseIP(ip); err != nil {
		return util.IPVersion(0), err
	} else if isv4 == false {
		version = util.IPVersion6
	}

	return version, nil
}
