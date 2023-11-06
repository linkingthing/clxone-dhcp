package resource

import (
	"net"

	gohelperip "github.com/cuityhj/gohelper/ip"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TablePool4 = restdb.ResourceDBType(&Pool4{})

type Pool4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet4                   string `json:"-" db:"ownby"`
	BeginAddress              string `json:"beginAddress" rest:"description=immutable"`
	BeginIp                   net.IP `json:"-"`
	EndAddress                string `json:"endAddress" rest:"description=immutable"`
	EndIp                     net.IP `json:"-"`
	Capacity                  uint64 `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64 `json:"usedCount" rest:"description=readonly" db:"-"`
	Template                  string `json:"template" db:"-"`
	Comment                   string `json:"comment"`
}

func (p Pool4) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet4{}}
}

const ActionNameValidTemplate = "valid_template"

type TemplateInfo struct {
	Template string `json:"template"`
}

type TemplatePool struct {
	BeginAddress string `json:"beginAddress"`
	EndAddress   string `json:"endAddress"`
}

func (p Pool4) GetActions() []restresource.Action {
	return []restresource.Action{
		restresource.Action{
			Name:   ActionNameValidTemplate,
			Input:  &TemplateInfo{},
			Output: &TemplatePool{},
		},
	}
}

func (p *Pool4) CheckConflictWithAnother(another *Pool4) bool {
	return gohelperip.IP(another.EndIp).Cmp(gohelperip.IP(p.BeginIp)) != -1 &&
		gohelperip.IP(another.BeginIp).Cmp(gohelperip.IP(p.EndIp)) != 1
}

func (p *Pool4) CheckConflictWithReservedPool4(reservedPool *ReservedPool4) bool {
	return gohelperip.IP(reservedPool.EndIp).Cmp(gohelperip.IP(p.BeginIp)) != -1 &&
		gohelperip.IP(reservedPool.BeginIp).Cmp(gohelperip.IP(p.EndIp)) != 1
}

func (p *Pool4) Contains(ip string) bool {
	if ip_, err := gohelperip.ParseIPv4(ip); err != nil {
		return false
	} else {
		return p.CheckConflictWithAnother(&Pool4{BeginIp: ip_, EndIp: ip_})
	}
}

func (p *Pool4) Equals(another *Pool4) bool {
	return p.Subnet4 == another.Subnet4 &&
		p.BeginAddress == another.BeginAddress &&
		p.EndAddress == another.EndAddress
}

func (p *Pool4) String() string {
	if p.BeginAddress != "" {
		return p.BeginAddress + PoolDelimiter + p.EndAddress
	} else {
		return ""
	}
}

func (p *Pool4) Validate() error {
	if err := util.ValidateStrings(util.RegexpTypeComma, p.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, p.Comment)
	}

	if p.Template != "" {
		return nil
	}

	return p.ValidateAddress()
}

func (p *Pool4) ParseAddressWithTemplate(tx restdb.Transaction, subnet *Subnet4) error {
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

func (p *Pool4) setAddrAndCapacity(beginIp, endIp net.IP, capacity uint64) {
	p.BeginAddress = beginIp.String()
	p.EndAddress = endIp.String()
	p.BeginIp = beginIp
	p.EndIp = endIp
	p.Capacity = capacity
}

func parsePool4FromTemplate(tx restdb.Transaction, template string, subnet *Subnet4) (net.IP, net.IP, uint64, error) {
	var templates []*Pool4Template
	if err := tx.Fill(map[string]interface{}{"name": template}, &templates); err != nil {
		return nil, nil, 0, errorno.ErrDBError(errorno.ErrDBNameQuery, template, pg.Error(err).Error())
	}

	if len(templates) != 1 {
		return nil, nil, 0, errorno.ErrNotFound(errorno.ErrNameTemplate, template)
	}

	subnetIpUint32 := gohelperip.IPv4ToUint32(subnet.Ipnet.IP)
	beginUint32 := subnetIpUint32 + uint32(templates[0].BeginOffset)
	endUint32 := beginUint32 + uint32(templates[0].Capacity-1)
	beginIp := gohelperip.IPv4FromUint32(beginUint32)
	endIp := gohelperip.IPv4FromUint32(endUint32)
	if !subnet.Ipnet.Contains(beginIp) || !subnet.Ipnet.Contains(endIp) {
		return nil, nil, 0, errorno.ErrInvalidRange(template,
			beginIp.String(), endIp.String())
	}

	return beginIp, endIp, templates[0].Capacity, nil
}

func (p *Pool4) ValidateAddress() error {
	beginIp, endIp, capacity, err := validPool4(p.BeginAddress, p.EndAddress)
	if err != nil {
		return err
	}

	p.setAddrAndCapacity(beginIp, endIp, capacity)
	return nil
}

func validPool4(beginAddr, endAddr string) (net.IP, net.IP, uint64, error) {
	beginIp, err := gohelperip.ParseIPv4(beginAddr)
	if err != nil {
		return nil, nil, 0, errorno.ErrInvalidParams(errorno.ErrNameIpv4, beginAddr)
	}

	endIp, err := gohelperip.ParseIPv4(endAddr)
	if err != nil {
		return nil, nil, 0, errorno.ErrInvalidParams(errorno.ErrNameIpv4, endAddr)
	}

	if capacity, err := calculateIpv4Pool4Capacity(beginIp, endIp); err != nil {
		return nil, nil, 0, err
	} else {
		return beginIp, endIp, capacity, nil
	}
}

func calculateIpv4Pool4Capacity(beginIp, endIp net.IP) (uint64, error) {
	endUint32 := gohelperip.IPv4ToUint32(endIp)
	beginUint32 := gohelperip.IPv4ToUint32(beginIp)
	if endUint32 < beginUint32 {
		return 0, errorno.ErrBiggerThan(errorno.ErrNameIp,
			beginIp.String(), endIp.String())
	} else {
		return uint64(endUint32) - uint64(beginUint32) + 1, nil
	}
}
