package resource

import (
	"fmt"
	"net"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TablePool4 = restdb.ResourceDBType(&Pool4{})

type Pool4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet4                   string `json:"-" db:"ownby"`
	BeginAddress              string `json:"beginAddress" rest:"description=immutable"`
	EndAddress                string `json:"endAddress" rest:"description=immutable"`
	Capacity                  uint64 `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64 `json:"usedCount" rest:"description=readonly" db:"-"`
	Template                  string `json:"template" db:"-"`
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
	if util.OneIpLessThanAnother(another.EndAddress, p.BeginAddress) ||
		util.OneIpLessThanAnother(p.EndAddress, another.BeginAddress) {
		return false
	}

	return true
}

func (p *Pool4) CheckConflictWithReservedPool4(reservedPool *ReservedPool4) bool {
	if util.OneIpLessThanAnother(reservedPool.EndAddress, p.BeginAddress) ||
		util.OneIpLessThanAnother(p.EndAddress, reservedPool.BeginAddress) {
		return false
	}

	return true
}

func (p *Pool4) Contains(ip string) bool {
	return p.CheckConflictWithAnother(&Pool4{BeginAddress: ip, EndAddress: ip})
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

type Pool4s []*Pool4

func (p Pool4s) Len() int {
	return len(p)
}

func (p Pool4s) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Pool4s) Less(i, j int) bool {
	return util.OneIpLessThanAnother(p[i].BeginAddress, p[j].BeginAddress)
}

func (p *Pool4) Validate() error {
	if p.Template != "" {
		return nil
	}

	return p.ValidateAddress()
}

func (p *Pool4) ParseAddressWithTemplate(tx restdb.Transaction, subnet *Subnet4) error {
	if p.Template == "" {
		return nil
	}

	pool, capacity, err := parsePool4FromTemplate(tx, p.Template, subnet)
	if err != nil {
		return err
	}

	p.BeginAddress = pool.BeginAddress
	p.EndAddress = pool.EndAddress
	p.Capacity = capacity
	return nil
}

func parsePool4FromTemplate(tx restdb.Transaction, template string, subnet *Subnet4) (*TemplatePool, uint64, error) {
	var templates []*Pool4Template
	if err := tx.Fill(map[string]interface{}{"name": template}, &templates); err != nil {
		return nil, 0, err
	}

	if len(templates) != 1 {
		return nil, 0, fmt.Errorf("no found pool template %s", template)
	}

	subnetIpUint32, _ := util.Ipv4ToUint32(subnet.Ipnet.IP)
	beginUint32 := subnetIpUint32 + uint32(templates[0].BeginOffset)
	endUint32 := beginUint32 + uint32(templates[0].Capacity-1)
	begin := util.Ipv4FromUint32(beginUint32)
	end := util.Ipv4FromUint32(endUint32)
	if subnet.Ipnet.Contains(begin) == false || subnet.Ipnet.Contains(end) == false {
		return nil, 0, fmt.Errorf("template %s pool %s-%s not belongs to subnet %s",
			template, begin.String(), end.String(), subnet.Subnet)
	}

	return &TemplatePool{
		BeginAddress: begin.String(),
		EndAddress:   end.String(),
	}, templates[0].Capacity, nil
}

func (p *Pool4) ValidateAddress() error {
	beginAddr, endAddr, capacity, err := validPool4(p.BeginAddress, p.EndAddress)
	if err != nil {
		return err
	}

	p.BeginAddress = beginAddr
	p.EndAddress = endAddr
	p.Capacity = capacity
	return nil
}

func validPool4(beginAddr, endAddr string) (string, string, uint64, error) {
	begin, isv4, err := util.ParseIP(beginAddr)
	if err != nil {
		return "", "", 0, fmt.Errorf("pool begin address %s is invalid", beginAddr)
	} else if isv4 == false {
		return "", "", 0, fmt.Errorf("pool begin address %s is not ipv4", beginAddr)
	}

	end, isv4, err := util.ParseIP(endAddr)
	if err != nil {
		return "", "", 0, fmt.Errorf("pool end address %s is invalid", endAddr)
	} else if isv4 == false {
		return "", "", 0, fmt.Errorf("pool end address %s is not ipv4", endAddr)
	}

	capacity := ipv4Pool4Capacity(begin, end)
	if capacity <= 0 {
		return "", "", 0, fmt.Errorf("invalid pool capacity with begin-address %s and end-address %s",
			beginAddr, endAddr)
	}

	return begin.String(), end.String(), capacity, nil
}

func ipv4Pool4Capacity(begin, end net.IP) uint64 {
	endUint32, _ := util.Ipv4ToUint32(end)
	beginUint32, _ := util.Ipv4ToUint32(begin)
	return uint64(endUint32 - beginUint32 + 1)
}
