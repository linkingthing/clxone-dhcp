package resource

import (
	"fmt"
	"net"
	"net/url"

	gohelperip "github.com/cuityhj/gohelper/ip"
	csvutil "github.com/linkingthing/clxone-utils/csv"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableSubnet4 = restdb.ResourceDBType(&Subnet4{})

type Subnet4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string    `json:"subnet" rest:"required=true,description=immutable" db:"suk"`
	Ipnet                     net.IPNet `json:"-" db:"suk"`
	SubnetId                  uint64    `json:"subnetId" rest:"description=readonly" db:"suk"`
	ValidLifetime             uint32    `json:"validLifetime"`
	MaxValidLifetime          uint32    `json:"maxValidLifetime"`
	MinValidLifetime          uint32    `json:"minValidLifetime"`
	SubnetMask                string    `json:"subnetMask"`
	DomainServers             []string  `json:"domainServers"`
	Routers                   []string  `json:"routers"`
	WhiteClientClasses        []string  `json:"whiteClientClasses"`
	BlackClientClasses        []string  `json:"blackClientClasses"`
	TftpServer                string    `json:"tftpServer"`
	Bootfile                  string    `json:"bootfile"`
	RelayAgentAddresses       []string  `json:"relayAgentAddresses"`
	IfaceName                 string    `json:"ifaceName"`
	NextServer                string    `json:"nextServer"`
	Ipv6OnlyPreferred         uint32    `json:"ipv6OnlyPreferred"`
	Tags                      string    `json:"tags"`
	NodeIds                   []string  `json:"nodeIds" db:"-"`
	NodeNames                 []string  `json:"nodeNames" db:"-"`
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
			Name:  csvutil.ActionNameImportCSV,
			Input: &csvutil.ImportFile{},
		},
		restresource.Action{
			Name:   csvutil.ActionNameExportCSV,
			Output: &csvutil.ExportFile{},
		},
		restresource.Action{
			Name:   csvutil.ActionNameExportCSVTemplate,
			Output: &csvutil.ExportFile{},
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

func (s *Subnet4) Contains(ip string) bool {
	if ipv4, err := gohelperip.ParseIPv4(ip); err != nil {
		return false
	} else {
		return s.Ipnet.Contains(ipv4)
	}
}

func (s *Subnet4) Validate(dhcpConfig *DhcpConfig, clientClass4s []*ClientClass4) error {
	ipnet, err := gohelperip.ParseCIDRv4(s.Subnet)
	if err != nil {
		return fmt.Errorf("subnet %s invalid: %s", s.Subnet, err.Error())
	}

	s.Ipnet = *ipnet
	s.Subnet = ipnet.String()
	if err := s.setSubnetDefaultValue(dhcpConfig); err != nil {
		return err
	}

	return s.ValidateParams(clientClass4s)
}

func (s *Subnet4) setSubnetDefaultValue(dhcpConfig *DhcpConfig) (err error) {
	if s.ValidLifetime != 0 && s.MinValidLifetime != 0 &&
		s.MaxValidLifetime != 0 && len(s.DomainServers) != 0 {
		return
	}

	if dhcpConfig == nil {
		dhcpConfig, err = GetDhcpConfig(true)
		if err != nil {
			return fmt.Errorf("get dhcp global config failed: %s", err.Error())
		}
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

	return
}

func (s *Subnet4) ValidateParams(clientClass4s []*ClientClass4) error {
	if err := checkTFTPValid(s.TftpServer, s.Bootfile); err != nil {
		return err
	}

	if err := util.ValidateStrings(util.RegexpTypeCommon, s.Tags, s.IfaceName); err != nil {
		return err
	}

	if s.SubnetMask != "" {
		if err := gohelperip.CheckIPv4sValid(s.SubnetMask); err != nil {
			return fmt.Errorf("subnet4 mask invalid: %s", err.Error())
		}
	}

	if s.NextServer != "" {
		if err := gohelperip.CheckIPv4sValid(s.NextServer); err != nil {
			return fmt.Errorf("subnet4 next server invalid: %s", err.Error())
		}
	}

	if s.Ipv6OnlyPreferred != 0 && s.Ipv6OnlyPreferred < 300 {
		return fmt.Errorf("subnet4 ipv6-only preferred must not be less than 300")
	}

	if err := checkCommonOptions(true, s.DomainServers, s.RelayAgentAddresses); err != nil {
		return err
	}

	if err := checkClientClass4s(s.WhiteClientClasses, s.BlackClientClasses, clientClass4s); err != nil {
		return err
	}

	if err := checkIpsValidWithVersion(true, s.Routers); err != nil {
		return err
	}

	if err := checkLifetimeValid(s.ValidLifetime, s.MinValidLifetime, s.MaxValidLifetime); err != nil {
		return err
	}

	return checkNodesValid(s.Nodes)
}

func checkTFTPValid(tftpServer, bootfile string) error {
	if len(bootfile) > 128 {
		return fmt.Errorf("bootfile must not bigger than 128")
	}

	if tftpServer != "" {
		if len(tftpServer) > 64 {
			return fmt.Errorf("tftp-server must not bigger than 64")
		}

		if _, err := url.Parse(tftpServer); err != nil {
			return fmt.Errorf("parse tftp server failed: %s", err.Error())
		}
	}

	return util.ValidateStrings(util.RegexpTypeSlash, tftpServer, bootfile)
}

func checkCommonOptions(isv4 bool, domainServers, relayAgents []string) error {
	if err := checkIpsValidWithVersion(isv4, domainServers); err != nil {
		return err
	}

	if err := checkIpsValidWithVersion(isv4, relayAgents); err != nil {
		return err
	}

	return nil
}

func checkIpsValidWithVersion(isv4 bool, ips []string) error {
	for _, ip := range ips {
		if _, err := gohelperip.ParseIP(ip, isv4); err != nil {
			return fmt.Errorf("ip %s invalid: %s", ip, err.Error())
		}
	}

	return nil
}

func checkClientClass4s(whiteClientClasses, blackClientClasses []string, clientClass4s []*ClientClass4) (err error) {
	if len(whiteClientClasses) == 0 && len(blackClientClasses) == 0 {
		return
	}

	if len(clientClass4s) == 0 {
		clientClass4s, err = GetClientClass4s()
	}

	clientClass4Set := make(map[string]struct{}, len(clientClass4s))
	for _, clientClass4 := range clientClass4s {
		clientClass4Set[clientClass4.Name] = struct{}{}
	}

	if err = checkClientClassesValid(whiteClientClasses, clientClass4Set); err != nil {
		return
	}

	return checkClientClassesValid(blackClientClasses, clientClass4Set)
}

func GetClientClass4s() ([]*ClientClass4, error) {
	var clientClass4s []*ClientClass4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &clientClass4s)
	}); err != nil {
		return nil, pg.Error(err)
	} else {
		return clientClass4s, nil
	}
}

func checkClientClassesValid(clientClassNames []string, clientClassSet map[string]struct{}) error {
	if len(clientClassNames) == 0 {
		return nil
	}

	clientClassNameSet := make(map[string]struct{}, len(clientClassNames))
	for _, clientClassName := range clientClassNames {
		if _, ok := clientClassNameSet[clientClassName]; ok {
			return fmt.Errorf("duplicate client class %s", clientClassName)
		} else {
			clientClassNameSet[clientClassName] = struct{}{}
		}
	}

	for _, clientClassName := range clientClassNames {
		if _, ok := clientClassSet[clientClassName]; !ok {
			return fmt.Errorf("no found client class %s", clientClassName)
		}
	}

	return nil
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
