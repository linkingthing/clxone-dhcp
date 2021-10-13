package resource

import (
	"fmt"
	"net"
	"strconv"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TablePdPool = restdb.ResourceDBType(&PdPool{})

type PdPool struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string `json:"-" db:"ownby"`
	Prefix                    string `json:"prefix" rest:"required=true"`
	PrefixLen                 uint32 `json:"prefixLen" rest:"required=true"`
	DelegatedLen              uint32 `json:"delegatedLen" rest:"required=true"`
	Capacity                  uint64 `json:"capacity" rest:"description=readonly"`
}

func (p PdPool) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}

func (p *PdPool) String() string {
	return p.Prefix + "/" + strconv.Itoa(int(p.PrefixLen))
}

type PdPools []*PdPool

func (p PdPools) Len() int {
	return len(p)
}

func (p PdPools) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p PdPools) Less(i, j int) bool {
	return util.OneIpLessThanAnother(p[i].Prefix, p[j].Prefix)
}

func (pdpool *PdPool) Validate() error {
	prefix, capacity, err := validPdPool(pdpool.Prefix, pdpool.PrefixLen, pdpool.DelegatedLen)
	if err != nil {
		return err
	}

	pdpool.Prefix = prefix
	pdpool.Capacity = capacity
	return nil
}

func (pdpool *PdPool) CheckConflictWithAnother(another *PdPool) bool {
	return pdpool.Contains(another.String()) || another.Contains(pdpool.String())
}

func (pdpool *PdPool) Contains(prefix string) bool {
	ip, ipnet, err := util.ParseCIDR(prefix, false)
	if err != nil {
		return false
	}

	if ones, _ := ipnet.Mask.Size(); uint32(ones) <= pdpool.PrefixLen ||
		uint32(ones) > pdpool.DelegatedLen {
		return false
	}

	return ipToIPNet(net.ParseIP(pdpool.Prefix), pdpool.PrefixLen).Contains(ip)
}

func ipToIPNet(ip net.IP, prefixLen uint32) *net.IPNet {
	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(int(prefixLen), 128),
	}
}

func validPdPool(prefix string, prefixLen, delegatedLen uint32) (string, uint64, error) {
	prefixIp, isv4, err := util.ParseIP(prefix)
	if err != nil || isv4 {
		return "", 0, fmt.Errorf("pdpool prefix %s is invalid", prefix)
	}

	if prefixLen >= 64 {
		return "", 0, fmt.Errorf("pdpool prefix len %d should not bigger than 64)", prefixLen)
	}

	if delegatedLen < prefixLen || delegatedLen > 64 {
		return "", 0, fmt.Errorf("pdpool delegated len %d not in (%d, 64]",
			delegatedLen, prefixLen)
	}

	return prefixIp.String(), (1 << (delegatedLen - prefixLen)) - 1, nil
}

func (pdpool *PdPool) GetRange() (string, string) {
	return getPdPoolRange(pdpool.Prefix, pdpool.PrefixLen, pdpool.DelegatedLen)
}

func getPdPoolRange(prefix string, prefixLen, delegatedLen uint32) (string, string) {
	prefixTo16 := net.ParseIP(prefix).To16()
	prefixBytes := make([]byte, len(prefixTo16))
	copy(prefixBytes, prefixTo16)
	beginIndex := (prefixLen - 1) / 8
	endIndex := (delegatedLen - 1) / 8
	for i := endIndex; i > beginIndex; i-- {
		if prefixBytes[i] == 0 {
			prefixBytes[i] += 255
		}
	}

	return prefix, net.IP(prefixBytes).String()
}
