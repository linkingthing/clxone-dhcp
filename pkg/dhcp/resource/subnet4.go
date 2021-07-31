package resource

import (
	"fmt"
	"net"

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
	SubnetId                  uint64    `json:"-" rest:"description=readonly" db:"suk"`
	ValidLifetime             uint32    `json:"validLifetime"`
	MaxValidLifetime          uint32    `json:"maxValidLifetime"`
	MinValidLifetime          uint32    `json:"minValidLifetime"`
	SubnetMask                string    `json:"subnetMask"`
	DomainServers             []string  `json:"domainServers"`
	Routers                   []string  `json:"routers"`
	ClientClass               string    `json:"clientClass"`
	IfaceName                 string    `json:"ifaceName"`
	NextServer                string    `json:"nextServer"`
	RelayAgentAddresses       []string  `json:"relayAgentAddresses"`
	Tags                      string    `json:"tags"`
	NetworkType               string    `json:"networkType"`
	Capacity                  uint64    `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string    `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64    `json:"usedCount" rest:"description=readonly" db:"-"`
}

type Subnet4s []*Subnet4

func (s Subnet4s) Len() int {
	return len(s)
}

func (s Subnet4s) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Subnet4s) Less(i, j int) bool {
	return s[i].SubnetId < s[j].SubnetId
}

func (s *Subnet4) Validate() error {
	ip, ipnet, err := net.ParseCIDR(s.Subnet)
	if err != nil {
		return fmt.Errorf("subnet %s invalid: %s", s.Subnet, err.Error())
	} else if ip.To4() == nil {
		return fmt.Errorf("subnet %s not is ipv4", s.Subnet)
	} else if ip.Equal(ipnet.IP) == false {
		return fmt.Errorf("subnet %s invalid: ip %s don`t match mask size", s.Subnet, ip.String())
	} else {
		ones, _ := ipnet.Mask.Size()
		if ones > 32 {
			return fmt.Errorf("subnet %s invalid: ip mask size %d is bigger than 32", s.Subnet, ones)
		}
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

	var configs []*DhcpConfig
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &configs)
	}); err != nil {
		return fmt.Errorf("get dhcp global config failed: %s", err.Error())
	}

	defaultValidLifetime := DefaultValidLifetime
	defaultMinLifetime := DefaultMinValidLifetime
	defaultMaxLifetime := DefaultMaxValidLifetime
	var defaultDomains []string
	if len(configs) != 0 {
		defaultValidLifetime = configs[0].ValidLifetime
		defaultMinLifetime = configs[0].MinValidLifetime
		defaultMaxLifetime = configs[0].MaxValidLifetime
		for _, domain := range configs[0].DomainServers {
			if _, isv4, err := util.ParseIP(domain); err == nil && isv4 {
				defaultDomains = append(defaultDomains, domain)
			}
		}
	}

	if s.ValidLifetime == 0 {
		s.ValidLifetime = defaultValidLifetime
	}

	if s.MinValidLifetime == 0 {
		s.MinValidLifetime = defaultMinLifetime
	}

	if s.MaxValidLifetime == 0 {
		s.MaxValidLifetime = defaultMaxLifetime
	}

	if len(s.DomainServers) == 0 {
		s.DomainServers = defaultDomains
	}

	return nil
}

func (s *Subnet4) ValidateParams() error {
	if err := util.CheckIPsValidWithVersion(true, s.RelayAgentAddresses...); err != nil {
		return fmt.Errorf("subnet relay agent addresses invalid: %s", err.Error())
	}

	if err := checkCommonOptions(true, s.ClientClass, s.DomainServers, s.Routers); err != nil {
		return err
	}

	return checkLifetimeValid(s.ValidLifetime, s.MinValidLifetime, s.MaxValidLifetime)
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

func (s *Subnet4) CheckConflictWithAnother(another *Subnet4) bool {
	return s.Ipnet.Contains(another.Ipnet.IP) || another.Ipnet.Contains(s.Ipnet.IP)
}
