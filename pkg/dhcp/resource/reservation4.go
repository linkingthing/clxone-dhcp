package resource

import (
	"fmt"
	"net"

	gohelperip "github.com/cuityhj/gohelper/ip"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TableReservation4 = restdb.ResourceDBType(&Reservation4{})

type Reservation4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet4                   string `json:"-" db:"ownby"`
	HwAddress                 string `json:"hwAddress" rest:"required=true"`
	IpAddress                 string `json:"ipAddress" rest:"required=true"`
	Ip                        net.IP `json:"-"`
	UsedRatio                 string `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64 `json:"usedCount" rest:"description=readonly" db:"-"`
	Capacity                  uint64 `json:"capacity" rest:"description=readonly"`
	Comment                   string `json:"comment"`
}

const (
	SqlColumnsIp = "ip"
)

func (r Reservation4) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet4{}}
}

func (r *Reservation4) String() string {
	return r.HwAddress + "-" + r.IpAddress
}

func (r *Reservation4) Validate() error {
	if _, err := net.ParseMAC(r.HwAddress); err != nil {
		return fmt.Errorf("hwaddress %s is invalid", r.HwAddress)
	}

	if ipv4, err := gohelperip.ParseIPv4(r.IpAddress); err != nil {
		return err
	} else {
		r.Ip = ipv4
	}

	if err := checkCommentValid(r.Comment); err != nil {
		return err
	}

	r.Capacity = 1
	return nil
}
