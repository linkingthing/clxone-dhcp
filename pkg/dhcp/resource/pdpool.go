package resource

import (
	"math/big"
	"net"
	"strconv"

	gohelperip "github.com/cuityhj/gohelper/ip"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TablePdPool = restdb.ResourceDBType(&PdPool{})

type PdPool struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string    `json:"-" db:"ownby"`
	Prefix                    string    `json:"prefix" rest:"required=true"`
	PrefixLen                 uint32    `json:"prefixLen" rest:"required=true"`
	PrefixIpnet               net.IPNet `json:"-"`
	DelegatedLen              uint32    `json:"delegatedLen" rest:"required=true"`
	Capacity                  string    `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string    `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64    `json:"usedCount" rest:"description=readonly" db:"-"`
	Comment                   string    `json:"comment"`
}

func (pdpool PdPool) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}

func (pdpool *PdPool) String() string {
	return pdpool.Prefix + PoolDelimiter + strconv.Itoa(int(pdpool.PrefixLen)) + PoolDelimiter + strconv.Itoa(int(pdpool.DelegatedLen))
}

func (pdpool *PdPool) Validate() error {
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

func ipToIPNet(ip net.IP, prefixLen uint32) net.IPNet {
	return net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(int(prefixLen), 128),
	}
}

func (pdpool *PdPool) CheckConflictWithAnother(another *PdPool) bool {
	return pdpool.PrefixIpnet.Contains(another.PrefixIpnet.IP) ||
		another.PrefixIpnet.Contains(pdpool.PrefixIpnet.IP)
}

func (pdpool *PdPool) Contains(prefix string) bool {
	if ipnet, err := gohelperip.ParseCIDRv6(prefix); err != nil {
		return false
	} else {
		prefixLen, _ := ipnet.Mask.Size()
		return pdpool.DelegatedLen == uint32(prefixLen) &&
			pdpool.PrefixIpnet.Contains(ipnet.IP)
	}
}

func (pdpool *PdPool) IntersectPrefix(prefix string) bool {
	if ipnet, err := gohelperip.ParseCIDRv6(prefix); err != nil {
		return false
	} else {
		return pdpool.PrefixIpnet.Contains(ipnet.IP) ||
			ipnet.Contains(pdpool.PrefixIpnet.IP)
	}
}

func (pdpool *PdPool) IntersectIpnet(ipnet net.IPNet) bool {
	return pdpool.PrefixIpnet.Contains(ipnet.IP) ||
		ipnet.Contains(pdpool.PrefixIpnet.IP)
}

func validPdPool(prefix string, prefixLen, delegatedLen uint32) (net.IP, string, error) {
	prefixIp, err := gohelperip.ParseIPv6(prefix)
	if err != nil {
		return nil, "", errorno.ErrInvalidAddress(prefix)
	}

	if prefixLen <= 0 || prefixLen > 64 {
		return nil, "", errorno.ErrNotInScope(errorno.ErrNamePrefix, 0, 64)
	}

	if delegatedLen < prefixLen || delegatedLen > 64 {
		return nil, "", errorno.ErrNotInScope(errorno.ErrNameDelegatedLen,
			prefixLen, 64)
	}

	return prefixIp, new(big.Int).Lsh(big.NewInt(1), uint(delegatedLen-prefixLen)).String(), nil
}

func (pdpool *PdPool) GetRange() (string, string) {
	return pdpool.Prefix, getPdPoolEndPrefix(pdpool.PrefixIpnet, pdpool.DelegatedLen)
}

func getPdPoolEndPrefix(prefixIpnet net.IPNet, delegatedLen uint32) string {
	prefixTo16 := prefixIpnet.IP.To16()
	prefixBytes := make([]byte, len(prefixTo16))
	copy(prefixBytes, prefixTo16)
	prefixLen, _ := prefixIpnet.Mask.Size()
	beginIndex := uint32((prefixLen - 1) / 8)
	endIndex := (delegatedLen - 1) / 8
	for i := endIndex; i > beginIndex; i-- {
		if prefixBytes[i] == 0 {
			prefixBytes[i] += 255
		}
	}

	return net.IP(prefixBytes).String()
}

func (pdpool *PdPool) AddCapacityWithBigInt(capacityForAdd *big.Int) string {
	pdpool.Capacity = AddCapacityWithBigInt(pdpool.Capacity, capacityForAdd)
	return pdpool.Capacity
}

func (pdpool *PdPool) SubCapacityWithBigInt(capacityForSub *big.Int) string {
	pdpool.Capacity = SubCapacityWithBigInt(pdpool.Capacity, capacityForSub)
	return pdpool.Capacity
}
