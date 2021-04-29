package resource

import (
	"context"
	"fmt"
	"net"

	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

var TableSubnet = restdb.ResourceDBType(&Subnet{})

type Subnet struct {
	restresource.ResourceBase `json:",inline"`
	Subnet                    string         `json:"subnet" rest:"required=true,description=immutable" db:"uk"`
	Ipnet                     net.IPNet      `json:"-"`
	SubnetId                  uint32         `json:"-" rest:"description=readonly"`
	ValidLifetime             uint32         `json:"validLifetime"`
	MaxValidLifetime          uint32         `json:"maxValidLifetime"`
	MinValidLifetime          uint32         `json:"minValidLifetime"`
	DomainServers             []string       `json:"domainServers"`
	Routers                   []string       `json:"routers"`
	ClientClass               string         `json:"clientClass"`
	IfaceName                 string         `json:"ifaceName"`
	RelayAgentAddresses       []string       `json:"relayAgentAddresses"`
	RelayAgentInterfaceId     string         `json:"relayAgentInterfaceId"`
	Tags                      string         `json:"tags"`
	NetworkType               string         `json:"networkType"`
	Capacity                  uint64         `json:"capacity" rest:"description=readonly"`
	UsedRatio                 string         `json:"usedRatio" rest:"description=readonly" db:"-"`
	UsedCount                 uint64         `json:"usedCount" rest:"description=readonly" db:"-"`
	Version                   util.IPVersion `json:"version" rest:"description=readonly"`
}

type Subnets []*Subnet

func (s Subnets) Len() int {
	return len(s)
}

func (s Subnets) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Subnets) Less(i, j int) bool {
	return s[i].SubnetId < s[j].SubnetId
}

func (s *Subnet) Validate() error {
	ip, ipnet, err := net.ParseCIDR(s.Subnet)
	if err != nil {
		return fmt.Errorf("subnet %s invalid: %s", s.Subnet, err.Error())
	} else if ip.Equal(ipnet.IP) == false {
		return fmt.Errorf("subnet %s invalid: ip %s don`t match mask size", s.Subnet, ip.String())
	} else {
		ones, _ := ipnet.Mask.Size()
		if ip.To4() != nil {
			if ones > 32 {
				return fmt.Errorf("subnet %s invalid: ip mask size %d is bigger than 32", s.Subnet, ones)
			} else {
				s.Version = util.IPVersion4
			}
		} else {
			if ones > 64 {
				return fmt.Errorf("subnet %s invalid: ip mask size %d is bigger than 64", s.Subnet, ones)
			} else {
				s.Version = util.IPVersion6
			}
		}
	}

	s.Ipnet = *ipnet
	s.Subnet = ipnet.String()
	if err := s.setSubnetDefaultValue(); err != nil {
		return err
	}

	return s.ValidateParams()
}

func (s *Subnet) setSubnetDefaultValue() error {
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
			if _, isv4, err := util.ParseIP(domain); err == nil || (s.Version == util.IPVersion4 && isv4) ||
				(s.Version == util.IPVersion6 && isv4 == false) {
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

func (s *Subnet) ValidateParams() error {
	if err := util.CheckIPsValidWithVersion(s.Version == util.IPVersion4, s.RelayAgentAddresses...); err != nil {
		return fmt.Errorf("subnet relay agent addresses invalid: %s", err.Error())
	}

	if err := checkCommonOptions(s.Version, s.ClientClass, s.DomainServers, s.Routers); err != nil {
		return err
	}

	return checkLifetimeValid(s.ValidLifetime, s.MinValidLifetime, s.MaxValidLifetime)
}

func checkCommonOptions(version util.IPVersion, clientClass string, domainServers, routers []string) error {
	if err := util.CheckIPsValidWithVersion(version == util.IPVersion4, routers...); err != nil {
		return fmt.Errorf("routers %v invalid: %s", routers, err.Error())
	}

	if err := util.CheckIPsValidWithVersion(version == util.IPVersion4, domainServers...); err != nil {
		return fmt.Errorf("domain servers %v invalid: %s", domainServers, err.Error())
	}

	if err := checkClientClassValid(clientClass); err != nil {
		return fmt.Errorf("client class %s invalid: %s", clientClass, err.Error())
	}

	return nil
}

func checkClientClassValid(clientClass string) error {
	if clientClass == "" {
		return nil
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(TableClientClass, map[string]interface{}{"name": clientClass}); err != nil {
			return err
		} else if exists == false {
			return fmt.Errorf("no found client class %s in db", clientClass)
		} else {
			return nil
		}
	})
}

func GetAllDelegatedIpPrefix() (ipV6List []*net.IPNet, ipV4List []*net.IPNet, err error) {
	var subnets Subnets
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &subnets)
	}); err != nil {
		return nil, nil, err
	}

	for i, _ := range subnets {
		_, ipNet, err := net.ParseCIDR(subnets[i].Subnet)
		if err != nil {
			return nil, nil, err
		}
		if ipNet.IP.To4() != nil {
			ipV4List = append(ipV4List, ipNet)
		} else {
			ipV6List = append(ipV6List, ipNet)
		}
	}
	return ipV6List, ipV4List, nil
}

func IsPrefixsDhcpWithTx(prefixs []string, containEachOther bool, tx restdb.Transaction) error {
	var subnets Subnets
	if err := tx.Fill(nil, &subnets); err != nil {
		return err
	}

	for _, prefix := range prefixs {
		if _, ipnet, err := net.ParseCIDR(prefix); err != nil {
			return err
		} else {
			for _, subnet := range subnets {
				if containEachOther {
					if util.CheckPrefixsContainEachOther(subnet.Subnet, ipnet) {
						return fmt.Errorf("%s has been dhcp", subnet.Subnet)
					}
				} else if util.PrefixsContainsIpNet(subnet.Subnet, *ipnet) {
					return fmt.Errorf("%s has been dhcp", subnet.Subnet)
				}
			}
		}
	}

	return nil
}

func IsPrefixsDhcp(prefixs []string, containEachOther bool) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return IsPrefixsDhcpWithTx(prefixs, containEachOther, tx)
	})
}

func GetSubnetsLeasesCount() (map[uint32]uint64, error) {
	resp, err := grpcclient.GetDHCPGrpcClient().GetSubnetsLeasesCount(context.TODO(),
		&pb.GetSubnetsLeasesCountRequest{})
	return resp.GetSubnetsLeasesCount(), err
}

func SetSubnetLeasesUsedRatioWithLeasesCount(subnet *Subnet, subnetsLeasesCount map[uint32]uint64) {
	if subnet.Capacity != 0 {
		if leasesCount, ok := subnetsLeasesCount[subnet.SubnetId]; ok {
			subnet.UsedCount = leasesCount
			subnet.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(subnet.Capacity))
		}
	}
}

func GetSubnetsUsageMap() (map[string]*Subnet, error) {
	var subnets Subnets
	subnetMap := make(map[string]*Subnet)
	if err := db.GetResources(nil, &subnets); err != nil {
		return nil, err
	}

	subnetsLeasesCount, err := GetSubnetsLeasesCount()
	if err != nil {
		_ = log.Warnf("get subnets leases count failed: %s", err.Error())
		return nil, nil
	}

	for _, subnet := range subnets {
		SetSubnetLeasesUsedRatioWithLeasesCount(subnet, subnetsLeasesCount)
		subnetMap[subnet.Subnet] = subnet
	}

	return subnetMap, nil
}

func GetSubnetsMap() (map[string]string, error) {
	var subnets Subnets
	subnetMap := make(map[string]string)
	if err := db.GetResources(nil, &subnets); err != nil {
		return nil, err
	}
	for _, subnet := range subnets {
		subnetMap[subnet.Subnet] = subnet.Subnet
	}

	return subnetMap, nil
}

func LoadSubnetLeases(subnet *Subnet) (*pb.GetLeasesResponse, error) {
	if subnet.Version == util.IPVersion4 {
		return grpcclient.GetDHCPGrpcClient().GetSubnet4Leases(context.TODO(),
			&pb.GetSubnet4LeasesRequest{Id: subnet.SubnetId})
	} else {
		return grpcclient.GetDHCPGrpcClient().GetSubnet6Leases(context.TODO(),
			&pb.GetSubnet6LeasesRequest{Id: subnet.SubnetId})
	}
}
