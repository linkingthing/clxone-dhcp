package resource

import (
	"fmt"
	"net"
	"strconv"

	agentutil "github.com/linkingthing/ddi-agent/pkg/dhcp/util"
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TablePdPool = restdb.ResourceDBType(&PdPool{})

type PdPool struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string   `json:"-" db:"ownby"`
	Prefix                    string   `json:"prefix" rest:"required=true"`
	PrefixLen                 uint32   `json:"prefixLen" rest:"required=true"`
	DelegatedLen              uint32   `json:"delegatedLen" rest:"required=true"`
	DomainServers             []string `json:"domainServers"`
	Capacity                  uint64   `json:"capacity" rest:"description=readonly"`
}

func (p PdPool) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet{}}
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
	return agentutil.OneIpLessThanAnother(p[i].Prefix, p[j].Prefix)
}

func (pdpool *PdPool) Validate() error {
	prefix, isv4, err := util.ParseIP(pdpool.Prefix)
	if err != nil || isv4 {
		return fmt.Errorf("pdpool prefix %s is invalid", pdpool.Prefix)
	}

	if pdpool.PrefixLen >= 128 {
		return fmt.Errorf("pdpool prefix len %d should not bigger than 128)", pdpool.PrefixLen)
	}

	if pdpool.DelegatedLen < pdpool.PrefixLen || pdpool.DelegatedLen > 128 {
		return fmt.Errorf("pdpool delegated len %d not in (%d, 128]", pdpool.DelegatedLen, pdpool.PrefixLen)
	}

	if err := util.CheckIPsValidWithVersion(false, pdpool.DomainServers...); err != nil {
		return fmt.Errorf("domain servers %v invalid: %s", pdpool.DomainServers, err.Error())
	}

	pdpool.Prefix = prefix.String()
	pdpool.Capacity = (1 << (pdpool.DelegatedLen - pdpool.PrefixLen)) - 1
	return nil
}

func (pdpool *PdPool) GetRange() (string, string) {
	prefixTo16 := net.ParseIP(pdpool.Prefix).To16()
	prefixBytes := make([]byte, len(prefixTo16))
	copy(prefixBytes, prefixTo16)
	beginIndex := (pdpool.PrefixLen - 1) / 8
	endIndex := (pdpool.DelegatedLen - 1) / 8
	for i := endIndex; i > beginIndex; i-- {
		if prefixBytes[i] == 0 {
			prefixBytes[i] += 255
		}
	}

	return pdpool.Prefix, net.IP(prefixBytes).String()
}
