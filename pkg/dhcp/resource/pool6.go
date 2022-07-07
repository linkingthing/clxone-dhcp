package resource

import (
	"fmt"
	"math/big"
	"net"
	"strconv"

	gohelperip "github.com/cuityhj/gohelper/ip"
	pg "github.com/linkingthing/clxone-utils/postgresql"
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
	Capacity                  string `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64 `json:"usedCount" rest:"description=readonly" db:"-"`
	Template                  string `json:"template" db:"-"`
	Comment                   string `json:"comment"`
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
	if err := CheckCommentValid(p.Comment); err != nil {
		return err
	}

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

func (p *Pool6) setAddrAndCapacity(beginIp, endIp net.IP, capacity string) {
	p.BeginAddress = beginIp.String()
	p.EndAddress = endIp.String()
	p.BeginIp = beginIp
	p.EndIp = endIp
	p.Capacity = capacity
}

func parsePool6FromTemplate(tx restdb.Transaction, template string, subnet *Subnet6) (net.IP, net.IP, string, error) {
	var templates []*Pool6Template
	if err := tx.Fill(map[string]interface{}{"name": template}, &templates); err != nil {
		return nil, nil, "", pg.Error(err)
	}

	if len(templates) != 1 {
		return nil, nil, "", fmt.Errorf("no found pool6 template %s", template)
	}

	subnetIpBigInt := gohelperip.IPv6ToBigInt(subnet.Ipnet.IP)
	beginBigInt := new(big.Int).Add(subnetIpBigInt, big.NewInt(int64(templates[0].BeginOffset)))
	endBigInt := new(big.Int).Add(beginBigInt, big.NewInt(int64(templates[0].Capacity-1)))
	beginIp := gohelperip.IPv6FromBigInt(beginBigInt)
	endIp := gohelperip.IPv6FromBigInt(endBigInt)
	if !subnet.Ipnet.Contains(beginIp) || !subnet.Ipnet.Contains(endIp) {
		return nil, nil, "", fmt.Errorf("template6 %s pool6 %s-%s not belongs to subnet6 %s",
			template, beginIp.String(), endIp.String(), subnet.Subnet)
	}

	return beginIp, endIp, strconv.FormatUint(templates[0].Capacity, 10), nil
}

func (p *Pool6) ValidateAddress() error {
	beginIp, endIp, capacity, err := validPool6(p.BeginAddress, p.EndAddress)
	if err != nil {
		return err
	}

	p.setAddrAndCapacity(beginIp, endIp, capacity)
	return nil
}

func validPool6(beginAddr, endAddr string) (net.IP, net.IP, string, error) {
	beginIp, err := gohelperip.ParseIPv6(beginAddr)
	if err != nil {
		return nil, nil, "", fmt.Errorf("pool6 begin address %s is invalid: %s",
			beginAddr, err.Error())
	}

	endIp, err := gohelperip.ParseIPv6(endAddr)
	if err != nil {
		return nil, nil, "", fmt.Errorf("pool6 end address %s is invalid: %s",
			beginAddr, err.Error())
	}

	if capacity, err := calculateIpv6Pool6Capacity(beginIp, endIp); err != nil {
		return nil, nil, "", err
	} else {
		return beginIp, endIp, capacity, nil
	}
}

func calculateIpv6Pool6Capacity(begin, end net.IP) (string, error) {
	beginBigInt := gohelperip.IPv6ToBigInt(begin)
	endBigInt := gohelperip.IPv6ToBigInt(end)
	if capacity, err := CalculateIpv6Pool6CapacityWithBigInt(beginBigInt, endBigInt); err != nil {
		return "", err
	} else {
		return capacity.String(), nil
	}
}

func CalculateIpv6Pool6CapacityWithBigInt(beginBigInt, endBigInt *big.Int) (*big.Int, error) {
	if endBigInt.Cmp(beginBigInt) == -1 {
		return nil, fmt.Errorf("begin address %s bigger than end address %s",
			beginBigInt.String(), endBigInt.String())
	}

	return new(big.Int).Add(new(big.Int).Sub(endBigInt, beginBigInt),
		big.NewInt(1)), nil
}

func (p *Pool6) AddCapacityWithBigInt(capacityForAdd *big.Int) string {
	p.Capacity = AddCapacityWithBigInt(p.Capacity, capacityForAdd)
	return p.Capacity
}

func (p *Pool6) SubCapacityWithBigInt(capacityForSub *big.Int) string {
	p.Capacity = SubCapacityWithBigInt(p.Capacity, capacityForSub)
	return p.Capacity
}
