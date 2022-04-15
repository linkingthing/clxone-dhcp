package resource

import (
	"fmt"
	"net"
	"strings"

	gohelperip "github.com/cuityhj/gohelper/ip"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
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
		return p.BeginAddress + "-" + p.EndAddress
	} else {
		return ""
	}
}

func (p *Pool4) Validate() error {
	if err := checkCommentValid(p.Comment); err != nil {
		return err
	}

	if p.Template != "" {
		return nil
	}

	return p.ValidateAddress()
}

func checkCommentValid(comment string) error {
	if strings.Contains(comment, ",") {
		return fmt.Errorf("comment %s contains illegal character comma", comment)
	} else {
		return nil
	}
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
		return nil, nil, 0, err
	}

	if len(templates) != 1 {
		return nil, nil, 0, fmt.Errorf("no found pool4 template %s", template)
	}

	subnetIpUint32 := gohelperip.IPv4ToUint32(subnet.Ipnet.IP)
	beginUint32 := subnetIpUint32 + uint32(templates[0].BeginOffset)
	endUint32 := beginUint32 + uint32(templates[0].Capacity-1)
	beginIp := gohelperip.IPv4FromUint32(beginUint32)
	endIp := gohelperip.IPv4FromUint32(endUint32)
	if subnet.Ipnet.Contains(beginIp) == false || subnet.Ipnet.Contains(endIp) == false {
		return nil, nil, 0, fmt.Errorf("pool4 template %s pool4 %s-%s not belongs to subnet4 %s",
			template, beginIp.String(), endIp.String(), subnet.Subnet)
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
		return nil, nil, 0, fmt.Errorf("pool4 begin address %s is invalid: %s",
			beginAddr, err.Error())
	}

	endIp, err := gohelperip.ParseIPv4(endAddr)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("pool4 end address %s is invalid: %s",
			endAddr, err.Error())
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
		return 0, fmt.Errorf("begin address %s bigger than end address %s",
			beginIp.String(), endIp.String())
	} else {
		return uint64(endUint32) - uint64(beginUint32) + 1, nil
	}
}
