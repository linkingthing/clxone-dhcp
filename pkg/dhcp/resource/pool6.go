package resource

import (
	"fmt"
	"math/big"
	"net"

	gohelperip "github.com/cuityhj/gohelper/ip"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TablePool6 = restdb.ResourceDBType(&Pool6{})

type Pool6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string `json:"-" db:"ownby"`
	BeginAddress              string `json:"beginAddress" rest:"description=immutable"`
	BeginIp                   net.IP `json:"-"`
	EndAddress                string `json:"endAddress" rest:"description=immutable"`
	EndIp                     net.IP `json:"-"`
	Capacity                  uint64 `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64 `json:"usedCount" rest:"description=readonly" db:"-"`
	Template                  string `json:"template" db:"-"`
}

func (p Pool6) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet6{}}
}

func (p Pool6) GetActions() []restresource.Action {
	return []restresource.Action{
		restresource.Action{
			Name:   ActionNameValidTemplate,
			Input:  &TemplateInfo{},
			Output: &TemplatePool{},
		},
	}
}

func (p *Pool6) CheckConflictWithAnother(another *Pool6) bool {
	return gohelperip.IP(another.EndIp).Cmp(gohelperip.IP(p.BeginIp)) != -1 &&
		gohelperip.IP(another.BeginIp).Cmp(gohelperip.IP(p.EndIp)) != 1
}

func (p *Pool6) CheckConflictWithReservedPool6(reservedPool *ReservedPool6) bool {
	return gohelperip.IP(reservedPool.EndIp).Cmp(gohelperip.IP(p.BeginIp)) != -1 &&
		gohelperip.IP(reservedPool.BeginIp).Cmp(gohelperip.IP(p.EndIp)) != 1
}

func (p *Pool6) Contains(ip string) bool {
	if ip_, err := gohelperip.ParseIPv6(ip); err != nil {
		return false
	} else {
		return p.CheckConflictWithAnother(&Pool6{BeginIp: ip_, EndIp: ip_})
	}
}

func (p *Pool6) Equals(another *Pool6) bool {
	return p.Subnet6 == another.Subnet6 &&
		p.BeginAddress == another.BeginAddress &&
		p.EndAddress == another.EndAddress
}

func (p *Pool6) String() string {
	if p.BeginAddress != "" {
		return p.BeginAddress + "-" + p.EndAddress
	} else {
		return ""
	}
}

func (p *Pool6) Validate() error {
	if p.Template != "" {
		return nil
	}

	return p.ValidateAddress()
}

func (p *Pool6) ParseAddressWithTemplate(tx restdb.Transaction, subnet *Subnet6) error {
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

func (p *Pool6) setAddrAndCapacity(beginIp, endIp net.IP, capacity uint64) {
	p.BeginAddress = beginIp.String()
	p.EndAddress = endIp.String()
	p.BeginIp = beginIp
	p.EndIp = endIp
	p.Capacity = capacity
}

func parsePool6FromTemplate(tx restdb.Transaction, template string, subnet *Subnet6) (net.IP, net.IP, uint64, error) {
	var templates []*Pool6Template
	if err := tx.Fill(map[string]interface{}{"name": template}, &templates); err != nil {
		return nil, nil, 0, err
	}

	if len(templates) != 1 {
		return nil, nil, 0, fmt.Errorf("no found pool template %s", template)
	}

	subnetIpBigInt := gohelperip.IPv6ToBigInt(subnet.Ipnet.IP)
	beginBigInt := big.NewInt(0).Add(subnetIpBigInt, big.NewInt(int64(templates[0].BeginOffset)))
	endBigInt := big.NewInt(0).Add(beginBigInt, big.NewInt(int64(templates[0].Capacity-1)))
	beginIp := net.IP(beginBigInt.Bytes())
	endIp := net.IP(endBigInt.Bytes())
	if subnet.Ipnet.Contains(beginIp) == false || subnet.Ipnet.Contains(endIp) == false {
		return nil, nil, 0, fmt.Errorf("template %s pool %s-%s not belongs to subnet %s",
			template, beginIp.String(), endIp.String(), subnet.Subnet)
	}

	return beginIp, endIp, templates[0].Capacity, nil
}

func (p *Pool6) ValidateAddress() error {
	beginIp, endIp, capacity, err := validPool6(p.BeginAddress, p.EndAddress)
	if err != nil {
		return err
	}

	p.setAddrAndCapacity(beginIp, endIp, capacity)
	return nil
}

func validPool6(beginAddr, endAddr string) (net.IP, net.IP, uint64, error) {
	beginIp, err := gohelperip.ParseIPv6(beginAddr)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("pool begin address %s is invalid: %s",
			beginAddr, err.Error())
	}

	endIp, err := gohelperip.ParseIPv6(endAddr)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("pool end address %s is invalid: %s",
			beginAddr, err.Error())
	}

	if capacity, err := calculateIpv6Pool6Capacity(beginIp, endIp); err != nil {
		return nil, nil, 0, err
	} else {
		return beginIp, endIp, capacity, nil
	}
}

const MaxUint64 uint64 = 1844674407370955165

func calculateIpv6Pool6Capacity(begin, end net.IP) (uint64, error) {
	beginBigInt := gohelperip.IPv6ToBigInt(begin)
	endBigInt := gohelperip.IPv6ToBigInt(end)
	return CalculateIpv6Pool6CapacityWithBigInt(beginBigInt, endBigInt)
}

func CalculateIpv6Pool6CapacityWithBigInt(beginBigInt, endBigInt *big.Int) (uint64, error) {
	if endBigInt.Cmp(beginBigInt) == -1 {
		return 0, fmt.Errorf("begin address %s bigger than end address %s",
			beginBigInt.String(), endBigInt.String())
	}

	capacity := big.NewInt(0).Sub(endBigInt, beginBigInt)
	if capacity_ := big.NewInt(0).Add(capacity, big.NewInt(1)); capacity_.IsUint64() {
		return capacity_.Uint64(), nil
	} else {
		return MaxUint64, nil
	}
}
