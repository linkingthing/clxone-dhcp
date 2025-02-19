package resource

import (
	"math/big"
	"net"
	"unicode/utf8"

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
	Tags                      string    `json:"tags"`
	IfaceName                 string    `json:"ifaceName"`
	WhiteClientClassStrategy  string    `json:"whiteClientClassStrategy"`
	WhiteClientClasses        []string  `json:"whiteClientClasses"`
	BlackClientClassStrategy  string    `json:"blackClientClassStrategy"`
	BlackClientClasses        []string  `json:"blackClientClasses"`
	ValidLifetime             uint32    `json:"validLifetime"`
	MaxValidLifetime          uint32    `json:"maxValidLifetime"`
	MinValidLifetime          uint32    `json:"minValidLifetime"`
	PreferredLifetime         uint32    `json:"preferredLifetime"`
	RelayAgentInterfaceId     string    `json:"relayAgentInterfaceId"`
	DomainServers             []string  `json:"domainServers"`
	InformationRefreshTime    uint32    `json:"informationRefreshTime"`
	CapWapACAddresses         []string  `json:"capWapACAddresses"`
	RelayAgentAddresses       []string  `json:"relayAgentAddresses"`
	RapidCommit               bool      `json:"rapidCommit"`
	EmbedIpv4                 bool      `json:"embedIpv4"`
	UseEui64                  bool      `json:"useEui64"`
	AddressCode               string    `json:"addressCode"`
	AddressCodeName           string    `json:"addressCodeName" db:"-"`
	Nodes                     []string  `json:"nodes"`
	NodeIds                   []string  `json:"nodeIds" db:"-"`
	NodeNames                 []string  `json:"nodeNames" db:"-"`
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

func (s *Subnet6) Validate(dhcpConfig *DhcpConfig, clientClass6s []*ClientClass6, addressCodes []*AddressCode) error {
	ipnet, err := gohelperip.ParseCIDRv6(s.Subnet)
	if err != nil {
		return errorno.ErrParseCIDR(s.Subnet)
	}

	s.Ipnet = *ipnet
	s.Subnet = ipnet.String()

	if err := s.setSubnet6DefaultValue(dhcpConfig); err != nil {
		return err
	}

	if err := s.ValidateParams(clientClass6s, addressCodes); err != nil {
		return err
	}

	return s.checkAutoGenAddrFactor(GetIpnetMaskSize(s.Ipnet))
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

func (s *Subnet6) ValidateParams(clientClass6s []*ClientClass6, addressCodes []*AddressCode) error {
	if utf8.RuneCountInString(s.Tags) > MaxNameLength {
		return errorno.ErrExceedResourceMaxCount(errorno.ErrNameName, errorno.ErrNameCharacter, MaxNameLength)
	}
	if err := util.ValidateStrings(util.RegexpTypeCommon, s.Tags); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameName, s.Tags)
	}
	if err := util.ValidateStrings(util.RegexpTypeCommon, s.IfaceName); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameIfName, s.IfaceName)
	}

	if err := util.ValidateStrings(util.RegexpTypeSlash, s.RelayAgentInterfaceId); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameRelayAgentIf, s.RelayAgentInterfaceId)
	}

	if s.InformationRefreshTime != 0 && s.InformationRefreshTime < 600 {
		return errorno.ErrInformationRefreshTime()
	}

	if err := checkCommonOptions(false, s.DomainServers, s.RelayAgentAddresses, s.CapWapACAddresses); err != nil {
		return err
	}

	if err := checkClientClassStrategy(s.WhiteClientClassStrategy, len(s.WhiteClientClasses) != 0); err != nil {
		return err
	}

	if err := checkClientClassStrategy(s.BlackClientClassStrategy, len(s.BlackClientClasses) != 0); err != nil {
		return err
	}

	if err := checkClientClass6s(s.WhiteClientClasses, s.BlackClientClasses, clientClass6s); err != nil {
		return err
	}

	if addrCodeId, addrCodeName, err := checkAddressCode(s.AddressCode, s.AddressCodeName, addressCodes); err != nil {
		return err
	} else {
		s.AddressCode = addrCodeId
		s.AddressCodeName = addrCodeName
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

	if err != nil {
		return
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
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameClientClass), pg.Error(err).Error())
	} else {
		return clientClass6s, nil
	}
}

func checkAddressCode(addressCodeId, addressCodeName string, addressCodes []*AddressCode) (string, string, error) {
	if addressCodeId == "" && addressCodeName == "" {
		return addressCodeId, addressCodeName, nil
	}

	var err error
	if len(addressCodes) == 0 {
		if addressCodeId != "" {
			addressCodes, err = GetAddressCodes(map[string]interface{}{restdb.IDField: addressCodeId})
		} else {
			addressCodes, err = GetAddressCodes(map[string]interface{}{SqlColumnName: addressCodeName})
		}
	}

	if err != nil {
		return addressCodeId, addressCodeName, err
	}

	for _, addressCode := range addressCodes {
		if addressCode.GetID() == addressCodeId || addressCode.Name == addressCodeName {
			return addressCode.GetID(), addressCode.Name, nil
		}
	}

	return addressCodeId, addressCodeName, errorno.ErrNotFound(errorno.ErrNameAddressCode, addressCodeName)
}

func GetAddressCodes(condition map[string]interface{}) ([]*AddressCode, error) {
	var addressCodes []*AddressCode
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(condition, &addressCodes)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery,
			string(errorno.ErrNameAddressCode), pg.Error(err).Error())
	}

	return addressCodes, nil
}

func checkPreferredLifetime(preferredLifetime, validLifetime, minValidLifetime uint32) error {
	if preferredLifetime > validLifetime || preferredLifetime < minValidLifetime {
		return errorno.ErrNotInRange(errorno.ErrNamePreferLifetime,
			minValidLifetime, validLifetime)
	}

	return nil
}

func GetIpnetMaskSize(ipnet net.IPNet) uint32 {
	size, _ := ipnet.Mask.Size()
	return uint32(size)
}

func (s *Subnet6) checkAutoGenAddrFactor(maskSize uint32) error {
	if s.CheckAutoGenAddrFactorConflict() {
		return errorno.ErrAutoGenAddrFactorConflict()
	}

	if s.UseEui64 || s.EmbedIpv4 {
		if maskSize != 64 {
			return errorno.ErrExpect(errorno.ErrNameEUI64, 64, maskSize)
		}

		s.Capacity = MaxUint64String
	} else if s.AddressCode != "" {
		if maskSize < 64 {
			return errorno.ErrAddressCodeMask()
		}

		s.Capacity = new(big.Int).Lsh(big.NewInt(1), 128-uint(maskSize)).String()
	} else {
		s.Capacity = "0"
	}

	return nil
}

func (s *Subnet6) CheckAutoGenAddrFactorConflict() bool {
	return (s.UseEui64 && s.EmbedIpv4) ||
		(s.UseEui64 && s.AddressCode != "") ||
		(s.EmbedIpv4 && s.AddressCode != "")
}

func (s *Subnet6) CanNotHasPools() bool {
	return s.UseEui64 || s.EmbedIpv4 || s.AddressCode != ""
}

func (s *Subnet6) UseAddressCode() bool {
	return s.AddressCode != ""
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
	if capacity == "" || capacityForAdd == nil || capacityForAdd.Sign() == 0 {
		return capacity
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
	if capacity == "" || capacityForSub == nil || capacityForSub.Sign() == 0 {
		return capacity
	} else {
		capacityBigInt, _ := new(big.Int).SetString(capacity, 10)
		return capacityBigInt.Sub(capacityBigInt, capacityForSub).String()
	}
}
