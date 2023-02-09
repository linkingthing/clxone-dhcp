package resource

import (
	"encoding/hex"
	"math/big"
	"net"
	"strings"

	gohelperip "github.com/cuityhj/gohelper/ip"
	dhcp6 "github.com/insomniacslk/dhcp/dhcpv6"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	ReservationIdDUID     = "duid"
	ReservationIdMAC      = "mac"
	ReservationIdHostname = "hostname"

	ReservationTypeIps      = "ips"
	ReservationTypePrefixes = "prefixes"
)

var TableReservation6 = restdb.ResourceDBType(&Reservation6{})

type Reservation6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string      `json:"-" db:"ownby"`
	Duid                      string      `json:"duid"`
	HwAddress                 string      `json:"hwAddress"`
	Hostname                  string      `json:"hostname"`
	IpAddresses               []string    `json:"ipAddresses"`
	Ips                       []net.IP    `json:"-"`
	Prefixes                  []string    `json:"prefixes"`
	Ipnets                    []net.IPNet `json:"-"`
	Capacity                  string      `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string      `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64      `json:"usedCount" rest:"description=readonly" db:"-"`
	Comment                   string      `json:"comment"`
}

func (r Reservation6) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}

func (r *Reservation6) String() string {
	if r.Duid != "" {
		return "duid$" + r.Duid
	} else if r.HwAddress != "" {
		return "mac$" + r.HwAddress
	} else {
		return "hostname$" + r.Hostname
	}
}

func (r *Reservation6) AddrString() string {
	if len(r.IpAddresses) != 0 {
		return "ips$" + strings.Join(r.IpAddresses, "_")
	} else {
		return "prefixes$" + strings.Join(r.Prefixes, "_")
	}
}

func (r *Reservation6) CheckConflictWithAnother(another *Reservation6) bool {
	if r.Duid == another.Duid && r.HwAddress == another.HwAddress && r.Hostname == another.Hostname {
		return true
	}

	for _, ip := range r.Ips {
		for _, ip_ := range another.Ips {
			if ip.Equal(ip_) {
				return true
			}
		}
	}

	for _, ipnet := range r.Ipnets {
		for _, ipnet_ := range another.Ipnets {
			if ipnet.Contains(ipnet_.IP) || ipnet_.Contains(ipnet.IP) {
				return true
			}
		}
	}

	return false
}

func (r *Reservation6) Validate() error {
	if (r.Duid != "" && r.HwAddress != "") ||
		(r.Duid != "" && r.Hostname != "") ||
		(r.HwAddress != "" && r.Hostname != "") ||
		(r.Duid == "" && r.HwAddress == "" && r.Hostname == "") {
		return errorno.ErrOnlyOne("DUID", string(errorno.ErrNameMac), string(errorno.ErrNameHostname))
	}

	if (len(r.IpAddresses) != 0 && len(r.Prefixes) != 0) ||
		(len(r.IpAddresses) == 0 && len(r.Prefixes) == 0) {
		return errorno.ErrOnlyOne(string(errorno.ErrNameIp), string(errorno.ErrNamePrefix))
	}

	if r.HwAddress != "" {
		if hw, err := net.ParseMAC(r.HwAddress); err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameMac, r.HwAddress)
		} else {
			r.HwAddress = strings.ToUpper(hw.String())
		}
	} else if r.Duid != "" {
		if err := parseDUID(r.Duid); err != nil {
			return errorno.ErrInvalidParams("DUID", r.Duid)
		}
	} else if r.Hostname != "" {
		if err := util.ValidateStrings(r.Hostname); err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameHostname, r.Hostname)
		}
	}

	uniqueIps := make(map[string]struct{})
	capacity := new(big.Int)
	for _, ip := range r.IpAddresses {
		if ipv6, err := gohelperip.ParseIPv6(ip); err != nil {
			return errorno.ErrInvalidAddress(ip)
		} else if _, ok := uniqueIps[ip]; ok {
			return errorno.ErrDuplicate(errorno.ErrNameIp, ip)
		} else {
			uniqueIps[ip] = struct{}{}
			r.Ips = append(r.Ips, ipv6)
			capacity.Add(capacity, big.NewInt(1))
		}
	}

	for i, prefix := range r.Prefixes {
		if ipnet, err := gohelperip.ParseCIDRv6(prefix); err != nil {
			return errorno.ErrParseCIDR(prefix)
		} else if ones, _ := ipnet.Mask.Size(); ones > 64 {
			return errorno.ErrExceedMaxCount(errorno.ErrNamePrefix, 64)
		} else if prefix_, conflict := isIpnetIntersectPrefixes(r.Prefixes, ipnet, i); conflict {
			return errorno.ErrConflict(errorno.ErrNamePrefix, errorno.ErrNamePrefix, prefix, prefix_)
		} else {
			r.Ipnets = append(r.Ipnets, *ipnet)
			capacity.Add(capacity, big.NewInt(1))
		}
	}

	if err := CheckCommentValid(r.Comment); err != nil {
		return err
	}

	r.Capacity = capacity.String()
	return nil
}

func isIpnetIntersectPrefixes(prefixes []string, ipnet *net.IPNet, index int) (string, bool) {
	for i := index + 1; i < len(prefixes); i++ {
		if isPrefixIntersectWithIpnet(prefixes[i], ipnet) {
			return prefixes[i], true
		}
	}

	return "", false
}

func isPrefixIntersectWithIpnet(prefix string, ipnet *net.IPNet) bool {
	ipnet_, err := gohelperip.ParseCIDRv6(prefix)
	if err != nil {
		return false
	}

	return ipnet.Contains(ipnet_.IP) || ipnet_.Contains(ipnet.IP)
}

func parseDUID(duid string) error {
	if len(duid) == 0 {
		return errorno.ErrEmpty(string(errorno.ErrNameDuid))
	}

	duidhexstr := strings.Replace(duid, ":", "", -1)
	duidbytes, err := hex.DecodeString(duidhexstr)
	if err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameDuid, duid)
	}

	_, err = dhcp6.DuidFromBytes(duidbytes)
	return err
}
