package resource

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"net"

	gohelperip "github.com/cuityhj/gohelper/ip"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableReservedPool4 = restdb.ResourceDBType(&ReservedPool4{})

type ReservedPool4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet4                   string `json:"-" db:"ownby"`
	BeginAddress              string `json:"beginAddress" rest:"description=immutable"`
	BeginIp                   net.IP `json:"-"`
	EndAddress                string `json:"endAddress" rest:"description=immutable"`
	EndIp                     net.IP `json:"-"`
	Capacity                  uint64 `json:"capacity" rest:"description=readonly"`
	Template                  string `json:"template" db:"-"`
	Comment                   string `json:"comment"`
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
	return gohelperip.IP(another.EndIp).Cmp(gohelperip.IP(p.BeginIp)) != -1 &&
		gohelperip.IP(another.BeginIp).Cmp(gohelperip.IP(p.EndIp)) != 1
}

func (p *ReservedPool4) Contains(ip string) bool {
	if ip_, err := gohelperip.ParseIPv4(ip); err != nil {
		return false
	} else {
		return p.CheckConflictWithAnother(&ReservedPool4{BeginIp: ip_, EndIp: ip_})
	}
}

func (p *ReservedPool4) Equals(another *ReservedPool4) bool {
	return p.Subnet4 == another.Subnet4 &&
		p.BeginAddress == another.BeginAddress &&
		p.EndAddress == another.EndAddress
}

func (p *ReservedPool4) String() string {
	if p.BeginAddress != "" {
		return p.BeginAddress + PoolDelimiter + p.EndAddress
	} else {
		return ""
	}
}

func (p *ReservedPool4) Validate() error {
	if err := util.ValidateStrings(util.RegexpTypeComma, p.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, p.Comment)
	}

	if p.Template != "" {
		return nil
	}

	return p.ValidateAddress()
}

func (p *ReservedPool4) ParseAddressWithTemplate(tx restdb.Transaction, subnet *Subnet4) error {
	if p.Template == "" {
		return nil
	}

	beginIp, endIp, capacity, err := parsePool4FromTemplate(tx, p.Template, subnet)
	if err != nil {
		return err
	}

	p.setAddrAndCapacity(beginIp, endIp, capacity)
	return nil
}

func (p *ReservedPool4) setAddrAndCapacity(beginIp, endIp net.IP, capacity uint64) {
	p.BeginAddress = beginIp.String()
	p.EndAddress = endIp.String()
	p.BeginIp = beginIp
	p.EndIp = endIp
	p.Capacity = capacity
}

func (p *ReservedPool4) ValidateAddress() error {
	beginIp, endIp, capacity, err := validPool4(p.BeginAddress, p.EndAddress)
	if err != nil {
		return err
	}

	p.setAddrAndCapacity(beginIp, endIp, capacity)
	return nil
}
