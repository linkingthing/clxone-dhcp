package resource

import (
	"fmt"
	"net"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableReservation4 = restdb.ResourceDBType(&Reservation4{})

type Reservation4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet4                   string `json:"-" db:"ownby"`
	HwAddress                 string `json:"hwAddress" rest:"required=true"`
	IpAddress                 string `json:"ipAddress" rest:"required=true"`
	UsedRatio                 string `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64 `json:"usedCount" rest:"description=readonly" db:"-"`
	Capacity                  uint64 `json:"capacity" rest:"description=readonly"`
}

func (r Reservation4) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet4{}}
}

func (r *Reservation4) String() string {
	return r.HwAddress + "-" + r.IpAddress
}

type Reservation4s []*Reservation4

func (r Reservation4s) Len() int {
	return len(r)
}

func (r Reservation4s) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Reservation4s) Less(i, j int) bool {
	return util.OneIpLessThanAnother(r[i].IpAddress, r[j].IpAddress)
}

func (r *Reservation4) Validate() error {
	if _, err := net.ParseMAC(r.HwAddress); err != nil {
		return fmt.Errorf("hwaddress %s is invalid", r.HwAddress)
	}

	if _, isv4, err := util.ParseIP(r.IpAddress); err != nil {
		return err
	} else if isv4 == false {
		return fmt.Errorf("ip %s is not ipv4", r.IpAddress)
	}

	return nil
}
