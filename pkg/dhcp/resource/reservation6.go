package resource

import (
	"encoding/hex"
	"fmt"
	"math/big"
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
	Capacity                  string   `json:"capacity" rest:"description=readonly"`
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
		return "prefixes-" + strings.Join(r.Prefixes, "_")
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
			if isPrefixesIntersect(prefix, prefix_) {
				return true
			}
		}
	}

	return false
}

func isPrefixesIntersect(onePrefix, anotherPrefix string) bool {
	one, err := gohelperip.ParseCIDRv6(onePrefix)
	if err != nil {
		return false
	}

	another, err := gohelperip.ParseCIDRv6(anotherPrefix)
	if err != nil {
		return false
	}

	return one.Contains(another.IP) || another.Contains(one.IP)
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

	capacity := big.NewInt(0)
	for _, ip := range r.IpAddresses {
		if ipv6, err := gohelperip.ParseIPv6(ip); err != nil {
			return err
		} else {
			r.Ips = append(r.Ips, ipv6)
			capacity = new(big.Int).Add(capacity, big.NewInt(1))
		}
	}

	for _, prefix := range r.Prefixes {
		if ipnet, err := gohelperip.ParseCIDRv6(prefix); err != nil {
			return err
		} else if ones, _ := ipnet.Mask.Size(); ones > 64 {
			return fmt.Errorf("prefix %s mask size %d must not bigger than 64", prefix, ones)
		} else {
			capacity = new(big.Int).Add(capacity, big.NewInt(1))
		}
	}

	if err := checkCommentValid(r.Comment); err != nil {
		return err
	}

	r.Capacity = capacity.String()
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
