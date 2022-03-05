package resource

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	dhcp6 "github.com/insomniacslk/dhcp/dhcpv6"

	gohelperip "github.com/cuityhj/gohelper/ip"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TableReservation6 = restdb.ResourceDBType(&Reservation6{})

type Reservation6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string   `json:"-" db:"ownby"`
	Duid                      string   `json:"duid"`
	HwAddress                 string   `json:"hwAddress"`
	IpAddresses               []string `json:"ipAddresses"`
	Ips                       []net.IP `json:"-"`
	Prefixes                  []string `json:"prefixes"`
	Capacity                  uint64   `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string   `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64   `json:"usedCount" rest:"description=readonly" db:"-"`
	Comment                   string   `json:"comment"`
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

func (r *Reservation6) AddrString() string {
	if len(r.IpAddresses) != 0 {
		return "ips-" + strings.Join(r.IpAddresses, "_")
	} else {
		return "prefixes-" + strings.Join(r.IpAddresses, "_")
	}
}

func (r *Reservation6) CheckConflictWithAnother(another *Reservation6) bool {
	if r.Duid == another.Duid && r.HwAddress == another.HwAddress {
		return true
	}

	for _, ipAddress := range r.IpAddresses {
		for _, ipAddress_ := range another.IpAddresses {
			if ipAddress_ == ipAddress {
				return true
			}
		}
	}

	for _, prefix := range r.Prefixes {
		for _, prefix_ := range another.Prefixes {
			if prefix_ == prefix {
				return true
			}
		}
	}

	return false
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
	} else {
		if err := parseDUID(r.Duid); err != nil {
			return fmt.Errorf("duid %s is invalid", r.Duid)
		}
	}

	for _, ip := range r.IpAddresses {
		if ipv6, err := gohelperip.ParseIPv6(ip); err != nil {
			return err
		} else {
			r.Ips = append(r.Ips, ipv6)
		}
	}

	for _, prefix := range r.Prefixes {
		if ipnet, err := gohelperip.ParseCIDRv6(prefix); err != nil {
			return err
		} else if ones, _ := ipnet.Mask.Size(); ones >= 64 {
			return fmt.Errorf("prefix %s mask size %d should less than 64", prefix, ones)
		}
	}

	r.Capacity = uint64(len(r.IpAddresses) + len(r.Prefixes))
	return nil
}

func parseDUID(duid string) error {
	if len(duid) == 0 {
		return fmt.Errorf("duid is required")
	}

	duidhexstr := strings.Replace(duid, ":", "", -1)
	duidbytes, err := hex.DecodeString(duidhexstr)
	if err != nil {
		return fmt.Errorf("decode duid with hex failed: %s", err.Error())
	}

	_, err = dhcp6.DuidFromBytes(duidbytes)
	return err
}
