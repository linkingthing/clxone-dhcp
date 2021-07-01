package resource

import (
	"fmt"
	"math/big"
	"net"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TablePool6 = restdb.ResourceDBType(&Pool6{})

type Pool6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet6                   string `json:"-" db:"ownby"`
	BeginAddress              string `json:"beginAddress" rest:"description=immutable"`
	EndAddress                string `json:"endAddress" rest:"description=immutable"`
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
	if util.OneIpLessThanAnother(another.EndAddress, p.BeginAddress) ||
		util.OneIpLessThanAnother(p.EndAddress, another.BeginAddress) {
		return false
	}

	return true
}

func (p *Pool6) CheckConflictWithReservedPool6(reservedPool *ReservedPool6) bool {
	if util.OneIpLessThanAnother(reservedPool.EndAddress, p.BeginAddress) ||
		util.OneIpLessThanAnother(p.EndAddress, reservedPool.BeginAddress) {
		return false
	}

	return true
}

func (p *Pool6) Contains(ip string) bool {
	return p.CheckConflictWithAnother(&Pool6{BeginAddress: ip, EndAddress: ip})
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

type Pool6s []*Pool6

func (p Pool6s) Len() int {
	return len(p)
}

func (p Pool6s) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Pool6s) Less(i, j int) bool {
	return util.OneIpLessThanAnother(p[i].BeginAddress, p[j].BeginAddress)
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

	pool, capacity, err := parsePool6FromTemplate(tx, p.Template, subnet.Ipnet.IP)
	if err != nil {
		return err
	}

	p.BeginAddress = pool.BeginAddress
	p.EndAddress = pool.EndAddress
	p.Capacity = capacity
	return nil
}

func parsePool6FromTemplate(tx restdb.Transaction, template string, subnetIp net.IP) (*TemplatePool, uint64, error) {
	var templates []*Pool6Template
	if err := tx.Fill(map[string]interface{}{"name": template}, &templates); err != nil {
		return nil, 0, err
	}

	if len(templates) != 1 {
		return nil, 0, fmt.Errorf("no found pool template %s", template)
	}

	subnetIpBigInt, _ := util.Ipv6ToBigInt(subnetIp)
	beginBigInt := big.NewInt(0).Add(subnetIpBigInt, big.NewInt(int64(templates[0].BeginOffset)))
	endBigInt := big.NewInt(0).Add(beginBigInt, big.NewInt(int64(templates[0].Capacity-1)))
	return &TemplatePool{
		BeginAddress: net.IP(beginBigInt.Bytes()).String(),
		EndAddress:   net.IP(endBigInt.Bytes()).String(),
	}, templates[0].Capacity, nil
}

func (p *Pool6) ValidateAddress() error {
	beginAddr, endAddr, capacity, err := validPool6(p.BeginAddress, p.EndAddress)
	if err != nil {
		return err
	}

	p.BeginAddress = beginAddr
	p.EndAddress = endAddr
	p.Capacity = capacity
	return nil
}

func validPool6(beginAddr, endAddr string) (string, string, uint64, error) {
	begin, isv4, err := util.ParseIP(beginAddr)
	if err != nil {
		return "", "", 0, fmt.Errorf("pool begin address %s is invalid", beginAddr)
	} else if isv4 {
		return "", "", 0, fmt.Errorf("pool begin address %s is not ipv6", beginAddr)
	}

	end, isv4, err := util.ParseIP(endAddr)
	if err != nil {
		return "", "", 0, fmt.Errorf("pool end address %s is invalid", endAddr)
	} else if isv4 {
		return "", "", 0, fmt.Errorf("pool end address %s is not ipv6", endAddr)
	}

	capacity := Ipv6Pool6Capacity(begin, end)
	if capacity <= 0 {
		return "", "", 0, fmt.Errorf("invalid pool capacity with begin-address %s and end-address %s",
			beginAddr, endAddr)
	}

	return begin.String(), end.String(), capacity, nil
}

const MaxUint64 uint64 = 1844674407370955165

func Ipv6Pool6Capacity(begin, end net.IP) uint64 {
	beginBigInt, _ := util.Ipv6ToBigInt(begin)
	endBigInt, _ := util.Ipv6ToBigInt(end)
	return Ipv6Pool6CapacityWithBigInt(beginBigInt, endBigInt)
}

func Ipv6Pool6CapacityWithBigInt(beginBigInt, endBigInt *big.Int) uint64 {
	capacity := big.NewInt(0).Sub(endBigInt, beginBigInt)
	if capacity_ := big.NewInt(0).Add(capacity, big.NewInt(1)); capacity_.IsUint64() {
		return capacity_.Uint64()
	} else {
		return MaxUint64
	}
}
