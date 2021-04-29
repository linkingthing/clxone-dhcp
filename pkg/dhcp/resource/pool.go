package resource

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"net"

	agentutil "github.com/linkingthing/ddi-agent/pkg/dhcp/util"
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TablePool = restdb.ResourceDBType(&Pool{})

type Pool struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string         `json:"-" db:"ownby"`
	BeginAddress              string         `json:"beginAddress" rest:"description=immutable" db:"uk"`
	EndAddress                string         `json:"endAddress" rest:"description=immutable" db:"uk"`
	DomainServers             []string       `json:"domainServers"`
	Routers                   []string       `json:"routers"`
	ClientClass               string         `json:"clientClass"`
	Capacity                  uint64         `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string         `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64         `json:"usedCount" rest:"description=readonly" db:"-"`
	Version                   util.IPVersion `json:"version" rest:"description=readonly"`
	Template                  string         `json:"template" db:"-"`
}

func (p Pool) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Subnet{}}
}

const ActionNameValidTemplate = "valid_template"

type TemplateInfo struct {
	Template string `json:"template"`
}

type TemplatePool struct {
	BeginAddress string `json:"beginAddress"`
	EndAddress   string `json:"endAddress"`
}

func (p Pool) GetActions() []restresource.Action {
	return []restresource.Action{
		restresource.Action{
			Name:   ActionNameValidTemplate,
			Input:  &TemplateInfo{},
			Output: &TemplatePool{},
		},
	}
}

func (p *Pool) CheckConflictWithAnother(another *Pool) bool {
	if agentutil.OneIpLessThanAnother(another.EndAddress, p.BeginAddress) ||
		agentutil.OneIpLessThanAnother(p.EndAddress, another.BeginAddress) {
		return false
	}

	return true
}

func (p *Pool) Contains(ip string) bool {
	return p.CheckConflictWithAnother(&Pool{BeginAddress: ip, EndAddress: ip})
}

func (p *Pool) Equals(another *Pool) bool {
	return p.Subnet == another.Subnet && p.BeginAddress == another.BeginAddress && p.EndAddress == another.EndAddress
}

func (p *Pool) String() string {
	if p.BeginAddress != "" {
		return p.BeginAddress + "-" + p.EndAddress
	} else {
		return ""
	}
}

type Pools []*Pool

func (p Pools) Len() int {
	return len(p)
}

func (p Pools) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Pools) Less(i, j int) bool {
	return agentutil.OneIpLessThanAnother(p[i].BeginAddress, p[j].BeginAddress)
}

func (p *Pool) Validate() error {
	if p.Template != "" {
		return nil
	}

	if err := p.ValidateAddress(); err != nil {
		return err
	}

	return p.ValidateParams()
}

func (p *Pool) ParseAddressWithTemplate(tx restdb.Transaction, subnet *Subnet) error {
	if p.Template == "" {
		return nil
	}

	var templates []*PoolTemplate
	if err := tx.Fill(map[string]interface{}{"name": p.Template}, &templates); err != nil {
		return err
	}

	if len(templates) != 1 {
		return fmt.Errorf("no found pool template %s", p.Template)
	}

	if subnet.Version != templates[0].Version {
		return fmt.Errorf("template %s version is diff from subnet %s version", templates[0].Name, subnet.Subnet)
	}

	subnetIp := subnet.Ipnet.IP
	if subnet.Version == util.IPVersion4 {
		subnetIpUint32, _ := agentutil.Ipv4ToUint32(subnetIp)
		beginUint32 := subnetIpUint32 + uint32(templates[0].BeginOffset)
		endUint32 := beginUint32 + uint32(templates[0].Capacity)
		p.BeginAddress = ipv4FromUint32(beginUint32)
		p.EndAddress = ipv4FromUint32(endUint32)
	} else {
		subnetIpBigInt, _ := agentutil.Ipv6ToBigInt(subnetIp)
		beginBigInt := big.NewInt(0).Add(subnetIpBigInt, big.NewInt(int64(templates[0].BeginOffset)))
		endBigInt := big.NewInt(0).Add(beginBigInt, big.NewInt(int64(templates[0].Capacity)))
		p.BeginAddress = net.IP(beginBigInt.Bytes()).String()
		p.EndAddress = net.IP(endBigInt.Bytes()).String()
	}

	p.Capacity = templates[0].Capacity
	p.Version = templates[0].Version
	p.DomainServers = templates[0].DomainServers
	p.Routers = templates[0].Routers
	p.ClientClass = templates[0].ClientClass
	return nil
}

func ipv4FromUint32(val uint32) string {
	addr := make([]byte, 4)
	binary.BigEndian.PutUint32(addr, val)
	return net.IP(addr).String()
}

func (p *Pool) ValidateAddress() error {
	begin, beginIsv4, err := util.ParseIP(p.BeginAddress)
	if err != nil {
		return fmt.Errorf("pool begin address %s is invalid", p.BeginAddress)
	}

	end, endIsv4, err := util.ParseIP(p.EndAddress)
	if err != nil {
		return fmt.Errorf("pool end address %s is invalid", p.EndAddress)
	}

	if beginIsv4 != endIsv4 {
		return fmt.Errorf("pool begin address %s version is diff from end address %s", p.BeginAddress, p.EndAddress)
	}

	if beginIsv4 {
		p.Version = util.IPVersion4
		p.Capacity = ipv4PoolCapacity(begin, end)
	} else {
		p.Version = util.IPVersion6
		p.Capacity = ipv6PoolCapacity(begin, end)
	}

	if p.Capacity <= 0 {
		return fmt.Errorf("invalid pool capacity with begin-address %s and end-address %s", p.BeginAddress, p.EndAddress)
	}

	p.BeginAddress = begin.String()
	p.EndAddress = end.String()
	return nil
}

func ipv4PoolCapacity(begin, end net.IP) uint64 {
	endUint32, _ := agentutil.Ipv4ToUint32(end)
	beginUint32, _ := agentutil.Ipv4ToUint32(begin)
	return uint64(endUint32 - beginUint32 + 1)
}

const MaxUint64 uint64 = 1844674407370955165

func ipv6PoolCapacity(begin, end net.IP) uint64 {
	beginBigInt, _ := agentutil.Ipv6ToBigInt(begin)
	endBigInt, _ := agentutil.Ipv6ToBigInt(end)
	capacity := big.NewInt(0).Sub(endBigInt, beginBigInt)
	if capacity_ := big.NewInt(0).Add(capacity, big.NewInt(1)); capacity_.IsUint64() {
		return capacity_.Uint64()
	} else {
		return MaxUint64
	}
}

func (p *Pool) ValidateParams() error {
	return checkCommonOptions(p.Version, p.ClientClass, p.DomainServers, p.Routers)
}
