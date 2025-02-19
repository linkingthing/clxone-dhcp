package resource

import (
	"net"
	"time"

	"github.com/linkingthing/cement/uuid"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"

	gohelperip "github.com/cuityhj/gohelper/ip"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableReservedPool6 = restdb.ResourceDBType(&ReservedPool6{})

var ReservedPool6Columns = []string{restdb.IDField, restdb.CreateTimeField, SqlColumnSubnet6,
	SqlColumnBeginAddress, SqlColumnBeginIp, SqlColumnEndAddress, SqlColumnEndIp,
	SqlColumnCapacity, SqlColumnComment}

type ReservedPool6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string `json:"-" db:"ownby"`
	BeginAddress              string `json:"beginAddress" rest:"description=immutable"`
	BeginIp                   net.IP `json:"-"`
	EndAddress                string `json:"endAddress" rest:"description=immutable"`
	EndIp                     net.IP `json:"-"`
	Capacity                  string `json:"capacity" rest:"description=readonly"`
	Template                  string `json:"template" db:"-"`
	Comment                   string `json:"comment"`
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
	return gohelperip.IP(another.EndIp).Cmp(gohelperip.IP(p.BeginIp)) != -1 &&
		gohelperip.IP(another.BeginIp).Cmp(gohelperip.IP(p.EndIp)) != 1
}

func (p *ReservedPool6) ContainsIpstr(ipstr string) bool {
	if ip, err := gohelperip.ParseIPv6(ipstr); err != nil {
		return false
	} else {
		return p.ContainsIp(ip)
	}
}

func (p *ReservedPool6) ContainsIp(ip net.IP) bool {
	return ip != nil && gohelperip.IP(ip).Cmp(gohelperip.IP(p.BeginIp)) != -1 &&
		gohelperip.IP(ip).Cmp(gohelperip.IP(p.EndIp)) != 1
}

func (p *ReservedPool6) Equals(another *ReservedPool6) bool {
	return p.Subnet6 == another.Subnet6 &&
		p.BeginAddress == another.BeginAddress &&
		p.EndAddress == another.EndAddress
}

func (p *ReservedPool6) String() string {
	if p.BeginAddress != "" {
		return p.BeginAddress + PoolDelimiter + p.EndAddress
	} else {
		return ""
	}
}

func (p *ReservedPool6) Validate() error {
	if err := util.ValidateStrings(util.RegexpTypeComma, p.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, p.Comment)
	}

	if p.Template != "" {
		return nil
	}

	return p.ValidateAddress()
}

func (p *ReservedPool6) ParseAddressWithTemplate(tx restdb.Transaction, subnet *Subnet6) error {
	if p.Template == "" {
		return nil
	}

	beginIp, endIp, capacity, err := parsePool6FromTemplate(tx, p.Template, subnet)
	if err != nil {
		return err
	}

	p.setAddrAndCapacity(beginIp, endIp, capacity)
	return nil
}

func (p *ReservedPool6) setAddrAndCapacity(beginIp, endIp net.IP, capacity string) {
	p.BeginAddress = beginIp.String()
	p.EndAddress = endIp.String()
	p.BeginIp = beginIp
	p.EndIp = endIp
	p.Capacity = capacity
}

func (p *ReservedPool6) ValidateAddress() error {
	beginIp, endIp, capacity, err := validPool6(p.BeginAddress, p.EndAddress)
	if err != nil {
		return err
	}

	p.setAddrAndCapacity(beginIp, endIp, capacity)
	return nil
}

func (p *ReservedPool6) GenCopyValues() []interface{} {
	if p.GetID() == "" {
		p.ID, _ = uuid.Gen()
	}
	return []interface{}{
		p.GetID(),
		time.Now(),
		p.Subnet6,
		p.BeginAddress,
		p.BeginIp,
		p.EndAddress,
		p.EndIp,
		p.Capacity,
		p.Comment,
	}
}
