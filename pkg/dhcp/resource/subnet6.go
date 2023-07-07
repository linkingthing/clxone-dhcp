package resource

import (
	"math/big"
	"net"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/clxone-utils/excel"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const MaxUint64String = "18446744073709551616"

var TableSubnet6 = restdb.ResourceDBType(&Subnet6{})

type Subnet6 struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string    `json:"subnet" rest:"required=true,description=immutable" db:"suk"`
	Ipnet                     net.IPNet `json:"-" db:"suk"`
	SubnetId                  uint64    `json:"subnetId" rest:"description=readonly" db:"suk"`
	ValidLifetime             uint32    `json:"validLifetime"`
	MaxValidLifetime          uint32    `json:"maxValidLifetime"`
	MinValidLifetime          uint32    `json:"minValidLifetime"`
	PreferredLifetime         uint32    `json:"preferredLifetime"`
	DomainServers             []string  `json:"domainServers"`
	WhiteClientClasses        []string  `json:"whiteClientClasses"`
	BlackClientClasses        []string  `json:"blackClientClasses"`
	IfaceName                 string    `json:"ifaceName"`
	RelayAgentAddresses       []string  `json:"relayAgentAddresses"`
	RelayAgentInterfaceId     string    `json:"relayAgentInterfaceId"`
	Tags                      string    `json:"tags"`
	NodeIds                   []string  `json:"nodeIds" db:"-"`
	NodeNames                 []string  `json:"nodeNames" db:"-"`
	Nodes                     []string  `json:"nodes"`
	RapidCommit               bool      `json:"rapidCommit"`
	UseEui64                  bool      `json:"useEui64"`
	UseAddressCode            bool      `json:"useAddressCode"`
	Capacity                  string    `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string    `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64    `json:"usedCount" rest:"description=readonly" db:"-"`
}

func (s Subnet6) GetActions() []restresource.Action {
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
			Output: &Subnet6ListOutput{},
		},
	}
}

type Subnet6ListOutput struct {
	Subnet6s []*Subnet6 `json:"subnet6s"`
}

func (s *Subnet6) Contains(ip string) bool {
	if ipv6, err := gohelperip.ParseIPv6(ip); err != nil {
		return false
	} else {
		return s.Ipnet.Contains(ipv6)
	}
}

func (s *Subnet6) Validate(dhcpConfig *DhcpConfig, clientClass6s []*ClientClass6) error {
	ipnet, err := gohelperip.ParseCIDRv6(s.Subnet)
	if err != nil {
		return errorno.ErrParseCIDR(s.Subnet)
	}

	s.Ipnet = *ipnet
	s.Subnet = ipnet.String()
	maskSize, _ := s.Ipnet.Mask.Size()
	if s.UseEui64 {
		if s.UseAddressCode {
			return errorno.ErrEui64Conflict()
		}

		if maskSize != 64 {
			return errorno.ErrExpect("EUI64", 64, maskSize)
		}
		s.Capacity = MaxUint64String
	} else {
		if s.UseAddressCode {
			if maskSize < 64 {
				return errorno.ErrAddressCodeMask()
			}

			if maskSize == 64 {
				s.Capacity = MaxUint64String
			} else {
				s.Capacity = new(big.Int).Lsh(big.NewInt(1), 128-uint(maskSize)).String()
			}
		} else {
			s.Capacity = "0"
		}
	}

	if err := s.setSubnet6DefaultValue(dhcpConfig); err != nil {
		return err
	}

	return s.ValidateParams(clientClass6s)
}

func (s *Subnet6) setSubnet6DefaultValue(dhcpConfig *DhcpConfig) (err error) {
	if s.ValidLifetime != 0 && s.MinValidLifetime != 0 &&
		s.MaxValidLifetime != 0 && len(s.DomainServers) != 0 {
		return
	}

	if dhcpConfig == nil {
		dhcpConfig, err = GetDhcpConfig(false)
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

	if s.PreferredLifetime == 0 {
		s.PreferredLifetime = dhcpConfig.ValidLifetime
	}

	if len(s.DomainServers) == 0 {
		s.DomainServers = dhcpConfig.DomainServers
	}

	return
}

func (s *Subnet6) ValidateParams(clientClass6s []*ClientClass6) error {
	if err := util.ValidateStrings(util.RegexpTypeCommon, s.Tags, s.IfaceName); err != nil {
		return err
	}

	if err := util.ValidateStrings(util.RegexpTypeSlash, s.RelayAgentInterfaceId); err != nil {
		return err
	}

	if err := checkCommonOptions(false, s.DomainServers, s.RelayAgentAddresses); err != nil {
		return err
	}

	if err := checkClientClass6s(s.WhiteClientClasses, s.BlackClientClasses, clientClass6s); err != nil {
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

func checkClientClass6s(whiteClientClasses, blackClientClasses []string, clientClass6s []*ClientClass6) (err error) {
	if len(whiteClientClasses) == 0 && len(blackClientClasses) == 0 {
		return
	}

	if len(clientClass6s) == 0 {
		clientClass6s, err = GetClientClass6s()
	}

	clientClass6Set := make(map[string]struct{}, len(clientClass6s))
	for _, clientClass6 := range clientClass6s {
		clientClass6Set[clientClass6.Name] = struct{}{}
	}

	if err = checkClientClassesValid(whiteClientClasses, clientClass6Set); err != nil {
		return
	}

	return checkClientClassesValid(blackClientClasses, clientClass6Set)
}

func GetClientClass6s() ([]*ClientClass6, error) {
	var clientClass6s []*ClientClass6
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &clientClass6s)
	}); err != nil {
		return nil, pg.Error(err)
	} else {
		return clientClass6s, nil
	}
}

func checkPreferredLifetime(preferredLifetime, validLifetime, minValidLifetime uint32) error {
	if preferredLifetime > validLifetime || preferredLifetime < minValidLifetime {
		return errorno.ErrNotInScope(errorno.ErrNameLifetime,
			minValidLifetime, validLifetime)
	}

	return nil
}

func (s *Subnet6) CheckConflictWithAnother(another *Subnet6) bool {
	return s.Ipnet.Contains(another.Ipnet.IP) || another.Ipnet.Contains(s.Ipnet.IP)
}

func IsCapacityZero(capacity string) bool {
	return capacity == "0" || capacity == ""
}

func (s *Subnet6) AddCapacityWithString(capacityForAdd string) string {
	capacityForAddBigInt, _ := new(big.Int).SetString(capacityForAdd, 10)
	return s.AddCapacityWithBigInt(capacityForAddBigInt)
}

func (s *Subnet6) AddCapacityWithBigInt(capacityForAdd *big.Int) string {
	s.Capacity = AddCapacityWithBigInt(s.Capacity, capacityForAdd)
	return s.Capacity
}

func AddCapacityWithBigInt(capacity string, capacityForAdd *big.Int) string {
	if capacity == "" || capacityForAdd == nil {
		return ""
	} else {
		capacityBigInt, _ := new(big.Int).SetString(capacity, 10)
		return capacityBigInt.Add(capacityBigInt, capacityForAdd).String()
	}
}

func (s *Subnet6) SubCapacityWithString(capacityForSub string) string {
	capacityForSubBigInt, _ := new(big.Int).SetString(capacityForSub, 10)
	return s.SubCapacityWithBigInt(capacityForSubBigInt)
}

func (s *Subnet6) SubCapacityWithBigInt(capacityForSub *big.Int) string {
	s.Capacity = SubCapacityWithBigInt(s.Capacity, capacityForSub)
	return s.Capacity
}

func SubCapacityWithBigInt(capacity string, capacityForSub *big.Int) string {
	if capacity == "" || capacityForSub == nil {
		return ""
	} else {
		capacityBigInt, _ := new(big.Int).SetString(capacity, 10)
		return capacityBigInt.Sub(capacityBigInt, capacityForSub).String()
	}
}
