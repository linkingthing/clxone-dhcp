package resource

import (
	"fmt"
	"net"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableReservation6 = restdb.ResourceDBType(&Reservation6{})

type Reservation6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string   `json:"-" db:"ownby"`
	Duid                      string   `json:"duid"`
	HwAddress                 string   `json:"hwAddress"`
	IpAddresses               []string `json:"ipAddresses"`
	Prefixes                  []string `json:"prefixes"`
	Capacity                  uint64   `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string   `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64   `json:"usedCount" rest:"description=readonly" db:"-"`
}

func (r Reservation6) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}

func (r *Reservation6) String() string {
	if len(r.Duid) != 0 {
		return "duid-" + r.Duid
	} else {
		return "mac-" + r.HwAddress
	}
}

type Reservation6s []*Reservation6

func (r Reservation6s) Len() int {
	return len(r)
}

func (r Reservation6s) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Reservation6s) Less(i, j int) bool {
	if r[i].Duid == r[j].Duid {
		return r[i].HwAddress < r[j].HwAddress
	} else {
		return r[i].Duid < r[j].Duid
	}
}

func (r *Reservation6) Validate() error {
	if len(r.Duid) != 0 && len(r.HwAddress) != 0 {
		return fmt.Errorf("duid and mac can not coexist")
	}

	if len(r.Duid) == 0 && len(r.HwAddress) == 0 {
		return fmt.Errorf("duid and mac has one at least")
	}

	if len(r.IpAddresses) != 0 && len(r.Prefixes) != 0 {
		return fmt.Errorf("ips and prefixes can not coexist")
	}

	if len(r.IpAddresses) == 0 && len(r.Prefixes) == 0 {
		return fmt.Errorf("ips and prefixes has one at least")
	}

	if len(r.HwAddress) != 0 {
		if _, err := net.ParseMAC(r.HwAddress); err != nil {
			return fmt.Errorf("hwaddress %s is invalid", r.HwAddress)
		}
	}

	for _, ip := range r.IpAddresses {
		if _, isv4, err := util.ParseIP(ip); err != nil {
			return err
		} else if isv4 {
			return fmt.Errorf("ip %s is not ipv6", ip)
		}
	}

	for _, prefix := range r.Prefixes {
		if ip, ipnet, err := net.ParseCIDR(prefix); err != nil {
			return err
		} else if ip.To4() != nil {
			return fmt.Errorf("prefix %s is not ipv6", prefix)
		} else {
			ones, _ := ipnet.Mask.Size()
			if ones >= 64 {
				return fmt.Errorf("prefix %s mask size %d should less than 64", prefix, ones)
			}
		}
	}

	r.Capacity = uint64(len(r.IpAddresses) + len(r.Prefixes))
	return nil
}
