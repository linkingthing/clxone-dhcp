package resource

import (
	"net"
	"strconv"

	gohelperip "github.com/cuityhj/gohelper/ip"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableReservedPdPool = restdb.ResourceDBType(&ReservedPdPool{})

type ReservedPdPool struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string    `json:"-" db:"ownby"`
	Prefix                    string    `json:"prefix" rest:"required=true"`
	PrefixLen                 uint32    `json:"prefixLen" rest:"required=true"`
	PrefixIpnet               net.IPNet `json:"-"`
	DelegatedLen              uint32    `json:"delegatedLen" rest:"required=true"`
	Capacity                  string    `json:"capacity" rest:"description=readonly"`
	Comment                   string    `json:"comment"`
}

func (pdpool ReservedPdPool) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}

func (pdpool *ReservedPdPool) String() string {
	return pdpool.Prefix + PoolDelimiter + strconv.Itoa(int(pdpool.PrefixLen)) + PoolDelimiter + strconv.Itoa(int(pdpool.DelegatedLen))
}

func (pdpool *ReservedPdPool) Validate() error {
	prefix, capacity, err := validPdPool(pdpool.Prefix, pdpool.PrefixLen, pdpool.DelegatedLen)
	if err != nil {
		return err
	}

	if err := util.ValidateStrings(util.RegexpTypeComma, pdpool.Comment); err != nil {
		return err
	}

	pdpool.Prefix = prefix.String()
	pdpool.PrefixIpnet = ipToIPNet(prefix, pdpool.PrefixLen)
	pdpool.Capacity = capacity
	return nil
}

func (pdpool *ReservedPdPool) CheckConflictWithAnother(another *ReservedPdPool) bool {
	return pdpool.PrefixIpnet.Contains(another.PrefixIpnet.IP) ||
		another.PrefixIpnet.Contains(pdpool.PrefixIpnet.IP)
}

func (pdpool *ReservedPdPool) IntersectIpnetString(prefix string) bool {
	if ipnet, err := gohelperip.ParseCIDRv6(prefix); err != nil {
		return false
	} else {
		return pdpool.PrefixIpnet.Contains(ipnet.IP) ||
			ipnet.Contains(pdpool.PrefixIpnet.IP)
	}
}

func (pdpool *ReservedPdPool) IntersectIpnet(ipnet net.IPNet) bool {
	return ipnet.IP != nil && (pdpool.PrefixIpnet.Contains(ipnet.IP) ||
		ipnet.Contains(pdpool.PrefixIpnet.IP))
}

func (pdpool *ReservedPdPool) GetRange() (string, string) {
	return pdpool.Prefix, getPdPoolEndPrefix(pdpool.PrefixIpnet, pdpool.DelegatedLen)
}
