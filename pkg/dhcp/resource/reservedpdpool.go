package resource

import (
	"strconv"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableReservedPdPool = restdb.ResourceDBType(&ReservedPdPool{})

type ReservedPdPool struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string `json:"-" db:"ownby"`
	Prefix                    string `json:"prefix" rest:"required=true"`
	PrefixLen                 uint32 `json:"prefixLen" rest:"required=true"`
	DelegatedLen              uint32 `json:"delegatedLen" rest:"required=true"`
	Capacity                  uint64 `json:"capacity" rest:"description=readonly"`
}

func (p ReservedPdPool) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}

func (p *ReservedPdPool) String() string {
	return p.Prefix + "-" + strconv.Itoa(int(p.PrefixLen)) + "-" + strconv.Itoa(int(p.DelegatedLen))
}

type ReservedPdPools []*ReservedPdPool

func (p ReservedPdPools) Len() int {
	return len(p)
}

func (p ReservedPdPools) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p ReservedPdPools) Less(i, j int) bool {
	return util.OneIpLessThanAnother(p[i].Prefix, p[j].Prefix)
}

func (pdpool *ReservedPdPool) Validate() error {
	prefix, capacity, err := validPdPool(pdpool.Prefix, pdpool.PrefixLen, pdpool.DelegatedLen)
	if err != nil {
		return err
	}

	pdpool.Prefix = prefix
	pdpool.Capacity = capacity
	return nil
}

func (pdpool *ReservedPdPool) CheckConflictWithAnother(another *ReservedPdPool) bool {
	return pdpool.Contains(another.String()) || another.Contains(pdpool.String())
}

func (pdpool *ReservedPdPool) Contains(prefix string) bool {
	pdpool_ := &PdPool{Prefix: pdpool.Prefix, PrefixLen: pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen}
	return pdpool_.Contains(prefix)
}

func (pdpool *ReservedPdPool) GetRange() (string, string) {
	return getPdPoolRange(pdpool.Prefix, pdpool.PrefixLen, pdpool.DelegatedLen)
}
