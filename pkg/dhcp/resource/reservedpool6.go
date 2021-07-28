package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableReservedPool6 = restdb.ResourceDBType(&ReservedPool6{})

type ReservedPool6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string `json:"-" db:"ownby"`
	BeginAddress              string `json:"beginAddress" rest:"description=immutable" db:"uk"`
	EndAddress                string `json:"endAddress" rest:"description=immutable" db:"uk"`
	Capacity                  uint64 `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64 `json:"usedCount" rest:"description=readonly" db:"-"`
	Template                  string `json:"template" db:"-"`
}

func (p ReservedPool6) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}

func (p ReservedPool6) GetActions() []restresource.Action {
	return []restresource.Action{
		restresource.Action{
			Name:   ActionNameValidTemplate,
			Input:  &TemplateInfo{},
			Output: &TemplatePool{},
		},
	}
}

func (p *ReservedPool6) CheckConflictWithAnother(another *ReservedPool6) bool {
	if util.OneIpLessThanAnother(another.EndAddress, p.BeginAddress) ||
		util.OneIpLessThanAnother(p.EndAddress, another.BeginAddress) {
		return false
	}

	return true
}

func (p *ReservedPool6) Contains(ip string) bool {
	return p.CheckConflictWithAnother(&ReservedPool6{BeginAddress: ip, EndAddress: ip})
}

func (p *ReservedPool6) Equals(another *ReservedPool6) bool {
	return p.Subnet6 == another.Subnet6 &&
		p.BeginAddress == another.BeginAddress &&
		p.EndAddress == another.EndAddress
}

func (p *ReservedPool6) String() string {
	if p.BeginAddress != "" {
		return p.BeginAddress + "-" + p.EndAddress
	} else {
		return ""
	}
}

type ReservedPool6s []*ReservedPool6

func (p ReservedPool6s) Len() int {
	return len(p)
}

func (p ReservedPool6s) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p ReservedPool6s) Less(i, j int) bool {
	return util.OneIpLessThanAnother(p[i].BeginAddress, p[j].BeginAddress)
}

func (p *ReservedPool6) Validate() error {
	if p.Template != "" {
		return nil
	}

	return p.ValidateAddress()
}

func (p *ReservedPool6) ParseAddressWithTemplate(tx restdb.Transaction, subnet *Subnet6) error {
	if p.Template == "" {
		return nil
	}

	pool, capacity, err := parsePool6FromTemplate(tx, p.Template, subnet)
	if err != nil {
		return err
	}

	p.BeginAddress = pool.BeginAddress
	p.EndAddress = pool.EndAddress
	p.Capacity = capacity
	return nil
}

func (p *ReservedPool6) ValidateAddress() error {
	beginAddr, endAddr, capacity, err := validPool6(p.BeginAddress, p.EndAddress)
	if err != nil {
		return err
	}

	p.BeginAddress = beginAddr
	p.EndAddress = endAddr
	p.Capacity = capacity
	return nil
}
