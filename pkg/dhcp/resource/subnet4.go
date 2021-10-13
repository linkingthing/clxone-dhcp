package resource

import (
	"fmt"
	"net"
	"net/url"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableSubnet4 = restdb.ResourceDBType(&Subnet4{})

type Subnet4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string    `json:"subnet" rest:"required=true,description=immutable" db:"suk"`
	Ipnet                     net.IPNet `json:"-"`
	SubnetId                  uint64    `json:"subnetId" rest:"description=readonly" db:"suk"`
	ValidLifetime             uint32    `json:"validLifetime"`
	MaxValidLifetime          uint32    `json:"maxValidLifetime"`
	MinValidLifetime          uint32    `json:"minValidLifetime"`
	SubnetMask                string    `json:"subnetMask"`
	DomainServers             []string  `json:"domainServers"`
	Routers                   []string  `json:"routers"`
	ClientClass               string    `json:"clientClass"`
	TftpServer                string    `json:"tftpServer"`
	Bootfile                  string    `json:"bootfile"`
	RelayAgentAddresses       []string  `json:"relayAgentAddresses"`
	IfaceName                 string    `json:"ifaceName"`
	NextServer                string    `json:"nextServer"`
	Tags                      string    `json:"tags"`
	Nodes                     []string  `json:"nodes"`
	Capacity                  uint64    `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string    `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64    `json:"usedCount" rest:"description=readonly" db:"-"`
}

const (
	ActionNameUpdateNodes     = "update_nodes"
	ActionNameCouldBeCreated  = "could_be_created"
	ActionNameListWithSubnets = "list_with_subnets"
)

func (s Subnet4) GetActions() []restresource.Action {
	return []restresource.Action{
		restresource.Action{
			Name:  util.ActionNameImportCSV,
			Input: &util.ImportFile{},
		},
		restresource.Action{
			Name:   util.ActionNameExportCSV,
			Output: &util.ExportFile{},
		},
		restresource.Action{
			Name:   util.ActionNameExportCSVTemplate,
			Output: &util.ExportFile{},
		},
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
			Output: &Subnet4ListOutput{},
		},
	}
}

type SubnetNode struct {
	Nodes []string `json:"nodes"`
}

type CouldBeCreatedSubnet struct {
	Subnet string `json:"subnet"`
}

type SubnetListInput struct {
	Subnets []string `json:"subnets"`
}

type Subnet4ListOutput struct {
	Subnet4s []*Subnet4 `json:"subnet4s"`
}

func (s *Subnet4) Validate() error {
	_, ipnet, err := util.ParseCIDR(s.Subnet, true)
	if err != nil {
		return fmt.Errorf("subnet %s invalid: %s", s.Subnet, err.Error())
	}

	s.Ipnet = *ipnet
	s.Subnet = ipnet.String()
	if err := s.setSubnetDefaultValue(); err != nil {
		return err
	}

	return s.ValidateParams()
}

func (s *Subnet4) setSubnetDefaultValue() error {
	if s.ValidLifetime != 0 && s.MinValidLifetime != 0 && s.MaxValidLifetime != 0 && len(s.DomainServers) != 0 {
		return nil
	}

	dhcpConfig, err := getDhcpConfig(true)
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

	if len(s.DomainServers) == 0 {
		s.DomainServers = dhcpConfig.DomainServers
	}

	return nil
}

func (s *Subnet4) ValidateParams() error {
	if err := checkTFTPServer(s.TftpServer); err != nil {
		return err
	}

	if err := util.CheckIPsValidWithVersion(true, s.RelayAgentAddresses...); err != nil {
		return fmt.Errorf("subnet relay agent addresses invalid: %s", err.Error())
	}

	if err := checkCommonOptions(true, s.ClientClass, s.DomainServers, s.Routers); err != nil {
		return err
	}

	if err := checkLifetimeValid(s.ValidLifetime, s.MinValidLifetime, s.MaxValidLifetime); err != nil {
		return err
	}

	return checkNodesValid(s.Nodes)
}

func checkTFTPServer(tftpServer string) error {
	if tftpServer != "" {
		if _, err := url.Parse(tftpServer); err != nil {
			return fmt.Errorf("parse tftp server failed: %s", err.Error())
		}
	}

	return nil
}

func checkCommonOptions(isv4 bool, clientClass string, domainServers, routers []string) error {
	if err := util.CheckIPsValidWithVersion(isv4, routers...); err != nil {
		return fmt.Errorf("routers %v invalid: %s", routers, err.Error())
	}

	if err := util.CheckIPsValidWithVersion(isv4, domainServers...); err != nil {
		return fmt.Errorf("domain servers %v invalid: %s", domainServers, err.Error())
	}

	if err := checkClientClassValid(isv4, clientClass); err != nil {
		return fmt.Errorf("client class %s invalid: %s", clientClass, err.Error())
	}

	return nil
}

func checkClientClassValid(isv4 bool, clientClass string) error {
	if clientClass == "" {
		return nil
	}

	tableName := TableClientClass4
	if isv4 == false {
		tableName = TableClientClass6
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(tableName, map[string]interface{}{"name": clientClass}); err != nil {
			return err
		} else if exists == false {
			return fmt.Errorf("no found client class %s in db", clientClass)
		} else {
			return nil
		}
	})
}

func checkNodesValid(nodes []string) error {
	for _, node := range nodes {
		if net.ParseIP(node) == nil {
			return fmt.Errorf("invalid node %s", node)
		}
	}

	return nil
}

func (s *Subnet4) CheckConflictWithAnother(another *Subnet4) bool {
	return s.Ipnet.Contains(another.Ipnet.IP) || another.Ipnet.Contains(s.Ipnet.IP)
}
