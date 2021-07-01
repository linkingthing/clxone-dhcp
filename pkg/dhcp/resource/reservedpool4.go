package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableReservedPool4 = restdb.ResourceDBType(&ReservedPool4{})

type ReservedPool4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet4                   string `json:"-" db:"ownby"`
	BeginAddress              string `json:"beginAddress" rest:"description=immutable" db:"uk"`
	EndAddress                string `json:"endAddress" rest:"description=immutable" db:"uk"`
	Capacity                  uint64 `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64 `json:"usedCount" rest:"description=readonly" db:"-"`
	Template                  string `json:"template" db:"-"`
}

func (p ReservedPool4) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet4{}}
}

func (p ReservedPool4) GetActions() []restresource.Action {
	return []restresource.Action{
		restresource.Action{
			Name:   ActionNameValidTemplate,
			Input:  &TemplateInfo{},
			Output: &TemplatePool{},
		},
	}
}

func (p *ReservedPool4) CheckConflictWithAnother(another *ReservedPool4) bool {
	if util.OneIpLessThanAnother(another.EndAddress, p.BeginAddress) ||
		util.OneIpLessThanAnother(p.EndAddress, another.BeginAddress) {
		return false
	}

	return true
}

func (p *ReservedPool4) Contains(ip string) bool {
	return p.CheckConflictWithAnother(&ReservedPool4{BeginAddress: ip, EndAddress: ip})
}

func (p *ReservedPool4) Equals(another *ReservedPool4) bool {
	return p.Subnet4 == another.Subnet4 &&
		p.BeginAddress == another.BeginAddress &&
		p.EndAddress == another.EndAddress
}

func (p *ReservedPool4) String() string {
	if p.BeginAddress != "" {
		return p.BeginAddress + "-" + p.EndAddress
	} else {
		return ""
	}
}

type ReservedPool4s []*ReservedPool4

func (p ReservedPool4s) Len() int {
	return len(p)
}

func (p ReservedPool4s) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p ReservedPool4s) Less(i, j int) bool {
	return util.OneIpLessThanAnother(p[i].BeginAddress, p[j].BeginAddress)
}

func (p *ReservedPool4) Validate() error {
	if p.Template != "" {
		return nil
	}

	return p.ValidateAddress()
}

func (p *ReservedPool4) ParseAddressWithTemplate(tx restdb.Transaction, subnet *Subnet4) error {
	if p.Template == "" {
		return nil
	}

	pool, capacity, err := parsePool4FromTemplate(tx, p.Template, subnet.Ipnet.IP)
	if err != nil {
		return err
	}

	p.BeginAddress = pool.BeginAddress
	p.EndAddress = pool.EndAddress
	p.Capacity = capacity
	return nil
}

func (p *ReservedPool4) ValidateAddress() error {
	beginAddr, endAddr, capacity, err := validPool4(p.BeginAddress, p.EndAddress)
	if err != nil {
		return err
	}

	p.BeginAddress = beginAddr
	p.EndAddress = endAddr
	p.Capacity = capacity
	return nil
}
