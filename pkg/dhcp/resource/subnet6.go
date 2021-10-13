package resource

import (
	"fmt"
	"net"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableSubnet6 = restdb.ResourceDBType(&Subnet6{})

type Subnet6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string    `json:"subnet" rest:"required=true,description=immutable" db:"suk"`
	Ipnet                     net.IPNet `json:"-"`
	SubnetId                  uint64    `json:"subnetId" rest:"description=readonly" db:"suk"`
	ValidLifetime             uint32    `json:"validLifetime"`
	MaxValidLifetime          uint32    `json:"maxValidLifetime"`
	MinValidLifetime          uint32    `json:"minValidLifetime"`
	PreferredLifetime         uint32    `json:"preferredLifetime"`
	DomainServers             []string  `json:"domainServers"`
	ClientClass               string    `json:"clientClass"`
	IfaceName                 string    `json:"ifaceName"`
	RelayAgentAddresses       []string  `json:"relayAgentAddresses"`
	RelayAgentInterfaceId     string    `json:"relayAgentInterfaceId"`
	Tags                      string    `json:"tags"`
	Nodes                     []string  `json:"nodes"`
	Capacity                  uint64    `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string    `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64    `json:"usedCount" rest:"description=readonly" db:"-"`
}

func (s Subnet6) GetActions() []restresource.Action {
	return []restresource.Action{
		restresource.Action{
			Name:  ActionNameUpdateNodes,
			Input: &SubnetNode{},
		},
		restresource.Action{
			Name:  ActionNameCouldBeCreated,
			Input: &CouldBeCreatedSubnet{},
		},
		restresource.Action{
			Name:   ActionNameListWithSubnets,
			Input:  &SubnetListInput{},
			Output: &Subnet6ListOutput{},
		},
	}
}

type Subnet6ListOutput struct {
	Subnet6s []*Subnet6 `json:"subnet6s"`
}

func (s *Subnet6) Validate() error {
	_, ipnet, err := util.ParseCIDR(s.Subnet, false)
	if err != nil {
		return fmt.Errorf("subnet %s invalid: %s", s.Subnet, err.Error())
	}

	s.Ipnet = *ipnet
	s.Subnet = ipnet.String()
	if err := s.setSubnet6DefaultValue(); err != nil {
		return err
	}

	return s.ValidateParams()
}

func (s *Subnet6) setSubnet6DefaultValue() error {
	if s.ValidLifetime != 0 && s.MinValidLifetime != 0 && s.MaxValidLifetime != 0 && len(s.DomainServers) != 0 {
		return nil
	}

	dhcpConfig, err := getDhcpConfig(false)
	if err != nil {
		return fmt.Errorf("get dhcp global config failed: %s", err.Error())
	}

	if s.ValidLifetime == 0 {
		s.ValidLifetime = dhcpConfig.ValidLifetime
	}

	if s.MinValidLifetime == 0 {
		s.MinValidLifetime = dhcpConfig.MinValidLifetime
	}

	if s.MaxValidLifetime == 0 {
		s.MaxValidLifetime = dhcpConfig.MaxValidLifetime
	}

	if s.PreferredLifetime == 0 {
		s.PreferredLifetime = dhcpConfig.ValidLifetime
	}

	if len(s.DomainServers) == 0 {
		s.DomainServers = dhcpConfig.DomainServers
	}

	return nil
}

func (s *Subnet6) ValidateParams() error {
	if err := util.CheckIPsValidWithVersion(false, s.RelayAgentAddresses...); err != nil {
		return fmt.Errorf("subnet relay agent addresses invalid: %s", err.Error())
	}

	if err := checkCommonOptions(false, s.ClientClass, s.DomainServers, nil); err != nil {
		return err
	}

	if err := checkLifetimeValid(s.ValidLifetime, s.MinValidLifetime, s.MaxValidLifetime); err != nil {
		return err
	}

	if err := checkPreferredLifetime(s.PreferredLifetime, s.ValidLifetime, s.MinValidLifetime); err != nil {
		return err
	}

	return checkNodesValid(s.Nodes)
}

func checkPreferredLifetime(preferredLifetime, validLifetime, minValidLifetime uint32) error {
	if preferredLifetime > validLifetime || preferredLifetime < minValidLifetime {
		return fmt.Errorf("preferred lifetime should in [%d, %d]", minValidLifetime, validLifetime)
	}

	return nil
}

func (s *Subnet6) CheckConflictWithAnother(another *Subnet6) bool {
	return s.Ipnet.Contains(another.Ipnet.IP) || another.Ipnet.Contains(s.Ipnet.IP)
}
