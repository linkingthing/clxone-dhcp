package resource

import (
	"net"
	"net/url"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/clxone-utils/excel"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	ClientClassStrategyAnd = "and"
	ClientClassStrategyOr  = "or"
)

var TableSubnet4 = restdb.ResourceDBType(&Subnet4{})

type Subnet4 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string    `json:"subnet" rest:"required=true,description=immutable" db:"suk"`
	Ipnet                     net.IPNet `json:"-" db:"suk"`
	SubnetId                  uint64    `json:"subnetId" rest:"description=readonly" db:"suk"`
	Tags                      string    `json:"tags"`
	IfaceName                 string    `json:"ifaceName"`
	WhiteClientClassStrategy  string    `json:"whiteClientClassStrategy"`
	WhiteClientClasses        []string  `json:"whiteClientClasses"`
	BlackClientClassStrategy  string    `json:"blackClientClassStrategy"`
	BlackClientClasses        []string  `json:"blackClientClasses"`
	ValidLifetime             uint32    `json:"validLifetime"`
	MaxValidLifetime          uint32    `json:"maxValidLifetime"`
	MinValidLifetime          uint32    `json:"minValidLifetime"`
	NextServer                string    `json:"nextServer"`
	SubnetMask                string    `json:"subnetMask"`
	Routers                   []string  `json:"routers"`
	DomainServers             []string  `json:"domainServers"`
	TftpServer                string    `json:"tftpServer"`
	Bootfile                  string    `json:"bootfile"`
	RelayAgentCircuitId       string    `json:"relayAgentCircuitId"`
	RelayAgentRemoteId        string    `json:"relayAgentRemoteId"`
	RelayAgentAddresses       []string  `json:"relayAgentAddresses"`
	Ipv6OnlyPreferred         uint32    `json:"ipv6OnlyPreferred"`
	CapWapACAddresses         []string  `json:"capWapACAddresses"`
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
			Name:  excel.ActionNameImport,
			Input: &excel.ImportFile{},
		},
		restresource.Action{
			Name:   excel.ActionNameExport,
			Output: &excel.ExportFile{},
		},
		restresource.Action{
			Name:   excel.ActionNameExportTemplate,
			Output: &excel.ExportFile{},
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
		return errorno.ErrInvalidParams(errorno.ErrNamePrefix, s.Subnet)
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
			return err
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

	if err := util.ValidateStrings(util.RegexpTypeCommon, s.Tags); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameName, s.Tags)
	}
	if err := util.ValidateStrings(util.RegexpTypeCommon, s.IfaceName); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameIfName, s.IfaceName)
	}

	if s.SubnetMask != "" {
		if err := gohelperip.CheckIPv4sValid(s.SubnetMask); err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameNetworkMask, s.SubnetMask)
		}
	}

	if s.NextServer != "" {
		if err := gohelperip.CheckIPv4sValid(s.NextServer); err != nil {
			return errorno.ErrInvalidAddress(s.NextServer)
		}
	}

	if s.Ipv6OnlyPreferred != 0 && s.Ipv6OnlyPreferred < 300 {
		return errorno.ErrIpv6Preferred()
	}

	if err := util.ValidateStrings(util.RegexpTypeSpace, s.RelayAgentCircuitId); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameRelayAgentCircuitId, s.RelayAgentCircuitId)
	}

	if err := util.ValidateStrings(util.RegexpTypeSpace, s.RelayAgentRemoteId); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameRelayAgentRemoteId, s.RelayAgentRemoteId)
	}

	if err := checkCommonOptions(true, s.DomainServers, s.RelayAgentAddresses, s.CapWapACAddresses); err != nil {
		return err
	}

	if err := checkClientClassStrategy(s.WhiteClientClassStrategy, len(s.WhiteClientClasses) != 0); err != nil {
		return err
	}

	if err := checkClientClassStrategy(s.BlackClientClassStrategy, len(s.BlackClientClasses) != 0); err != nil {
		return err
	}

	if err := checkClientClass4s(s.WhiteClientClasses, s.BlackClientClasses, clientClass4s); err != nil {
		return err
	}

	if err := checkIpsValidWithVersion(true, s.Routers); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameGateway, s.Routers)
	}

	if err := checkLifetimeValid(s.ValidLifetime, s.MinValidLifetime, s.MaxValidLifetime); err != nil {
		return err
	}

	return checkNodesValid(s.Nodes)
}

func checkTFTPValid(tftpServer, bootfile string) error {
	if len(bootfile) > 128 {
		return errorno.ErrBiggerThan(errorno.ErrNameBootFile, len(bootfile), 128)
	}

	if tftpServer != "" {
		if len(tftpServer) > 64 {
			return errorno.ErrBiggerThan(errorno.ErrNameTftpServer, len(tftpServer), 64)
		}

		if _, err := url.Parse(tftpServer); err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameTftpServer, tftpServer)
		}
	}

	if err := util.ValidateStrings(util.RegexpTypeSlash, tftpServer); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameTftpServer, tftpServer)
	}
	if err := util.ValidateStrings(util.RegexpTypeSlash, bootfile); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameBootFile, bootfile)
	}
	return nil
}

func checkCommonOptions(isv4 bool, domainServers, relayAgents, acAddresses []string) error {
	if err := checkIpsValidWithVersion(isv4, domainServers); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameDNS, domainServers)
	}

	if err := checkIpsValidWithVersion(isv4, relayAgents); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameRelayAgentAddresses, relayAgents)
	}

	if err := checkIpsValidWithVersion(isv4, acAddresses); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameCapWapACAddresses, acAddresses)
	}

	return nil
}

func checkIpsValidWithVersion(isv4 bool, ips []string) error {
	for _, ip := range ips {
		if _, err := gohelperip.ParseIP(ip, isv4); err != nil {
			return errorno.ErrInvalidAddress(ip)
		}
	}

	return nil
}

func checkClientClassStrategy(strategy string, needCheck bool) error {
	if needCheck {
		if strategy != ClientClassStrategyAnd && strategy != ClientClassStrategyOr {
			return errorno.ErrNotInScope(errorno.ErrNameClientClassStrategy,
				string(errorno.ErrNameClientClassStrategyAnd), string(errorno.ErrNameClientClassStrategyOr))
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

	if err != nil {
		return
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameClientClass), pg.Error(err).Error())
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
		if _, ok := clientClassSet[clientClassName]; !ok {
			return errorno.ErrNotFound(errorno.ErrNameClientClass, clientClassName)
		}

		if _, ok := clientClassNameSet[clientClassName]; ok {
			return errorno.ErrDuplicate(errorno.ErrNameClientClass, clientClassName)
		} else {
			clientClassNameSet[clientClassName] = struct{}{}
		}
	}

	return nil
}

func checkNodesValid(nodes []string) error {
	for _, node := range nodes {
		if net.ParseIP(node) == nil {
			return errorno.ErrInvalidAddress(node)
		}
	}

	return nil
}

func (s *Subnet4) CheckConflictWithAnother(another *Subnet4) bool {
	return s.Ipnet.Contains(another.Ipnet.IP) || another.Ipnet.Contains(s.Ipnet.IP)
}
