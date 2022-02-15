package resource

import (
	"fmt"
	"net"
	"strconv"

	gohelperip "github.com/cuityhj/gohelper/ip"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TablePdPool = restdb.ResourceDBType(&PdPool{})

type PdPool struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string    `json:"-" db:"ownby"`
	Prefix                    string    `json:"prefix" rest:"required=true"`
	PrefixLen                 uint32    `json:"prefixLen" rest:"required=true"`
	PrefixIpnet               net.IPNet `json:"-"`
	DelegatedLen              uint32    `json:"delegatedLen" rest:"required=true"`
	Capacity                  uint64    `json:"capacity" rest:"description=readonly"`
	Comment                   string    `json:"comment"`
}

const (
	SqlColumnSubnet6     = "subnet6"
	SqlColumnSubnet4     = "subnet4"
	SqlColumnBeginIp     = "begin_ip"
	SqlColumnPrefixIpNet = "prefix_ipnet"
	SqlColumnBeginOffset = "begin_offset"
)

func (p PdPool) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}

func (p *PdPool) String() string {
	return p.PrefixIpnet.String() + "-" + strconv.Itoa(int(p.DelegatedLen))
}

func (pdpool *PdPool) Validate() error {
	prefix, capacity, err := validPdPool(pdpool.Prefix, pdpool.PrefixLen, pdpool.DelegatedLen)
	if err != nil {
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
		return pdpool.PrefixIpnet.Contains(ipnet.IP)
	}
}

func validPdPool(prefix string, prefixLen, delegatedLen uint32) (net.IP, uint64, error) {
	prefixIp, err := gohelperip.ParseIPv6(prefix)
	if err != nil {
		return nil, 0, fmt.Errorf("pdpool prefix %s is invalid: %s", prefix, err.Error())
	}

	if prefixLen <= 0 || prefixLen >= 64 {
		return nil, 0, fmt.Errorf("pdpool prefix len %d not in (0, 64)", prefixLen)
	}

	if delegatedLen < prefixLen || delegatedLen > 64 {
		return nil, 0, fmt.Errorf("pdpool delegated len %d not in [%d, 64]",
			delegatedLen, prefixLen)
	}

	return prefixIp, (1 << (delegatedLen - prefixLen)) - 1, nil
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
