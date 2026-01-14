package resource

import (
	"strings"

	gohelperip "github.com/cuityhj/gohelper/ip"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

var TableDhcpConfig = restdb.ResourceDBType(&DhcpConfig{})

const (
	MinValidLifetime        uint32 = 3600
	DefaultMinValidLifetime uint32 = 10800
	DefaultMaxValidLifetime uint32 = 28800
	DefaultValidLifetime    uint32 = 14400
)

var DefaultDhcpConfig = &DhcpConfig{
	ValidLifetime:    DefaultValidLifetime,
	MinValidLifetime: DefaultMinValidLifetime,
	MaxValidLifetime: DefaultMaxValidLifetime,
}

type DhcpConfig struct {
	restresource.ResourceBase `json:",inline"`
	ValidLifetime             uint32   `json:"validLifetime"`
	MaxValidLifetime          uint32   `json:"maxValidLifetime"`
	MinValidLifetime          uint32   `json:"minValidLifetime"`
	DomainServers             []string `json:"domainServers"`
	DomainSearchList          []string `json:"domainSearchList"`
	Routers                   []string `json:"routers"`
	Subnet4WhiteClientClasses []string `json:"subnet4WhiteClientClasses"`
	Subnet4BlackClientClasses []string `json:"subnet4BlackClientClasses"`
	Subnet6WhiteClientClasses []string `json:"subnet6WhiteClientClasses"`
	Subnet6BlackClientClasses []string `json:"subnet6BlackClientClasses"`
}

func (config *DhcpConfig) Validate() error {
	if err := gohelperip.CheckIPsValid(config.DomainServers...); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameIp, strings.Join(config.DomainServers, ","))
	}

	if err := checkDomainSearchList(config.DomainSearchList); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameDomainSearchList, config.DomainSearchList)
	}

	if err := checkIpsValidWithVersion(true, config.Routers); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameGateway, config.Routers)
	}

	if err := checkClientClass4s(config.Subnet4WhiteClientClasses, config.Subnet4BlackClientClasses, nil); err != nil {
		return err
	}

	if err := checkClientClass6s(config.Subnet6WhiteClientClasses, config.Subnet6BlackClientClasses, nil); err != nil {
		return err
	}

	return checkLifetimeValid(config.ValidLifetime, config.MinValidLifetime, config.MaxValidLifetime)
}

func checkLifetimeValid(validLifetime, minValidLifetime, maxValidLifetime uint32) error {
	if minValidLifetime < MinValidLifetime {
		return errorno.ErrMinLifetime(MinValidLifetime)
	}

	if minValidLifetime > maxValidLifetime {
		return errorno.ErrMinLifetime(MinValidLifetime)
	}

	if validLifetime < minValidLifetime || validLifetime > maxValidLifetime {
		return errorno.ErrDefaultLifetime()
	}

	return nil
}

func GetDhcpConfig(isv4 bool) (*DhcpConfig, error) {
	var configs []*DhcpConfig
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &configs)
	}); err != nil {
		return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameConfig), pg.Error(err).Error())
	}

	if len(configs) != 0 {
		var defaultDomains []string
		for _, domain := range configs[0].DomainServers {
			if _, err := gohelperip.ParseIP(domain, isv4); err == nil {
				defaultDomains = append(defaultDomains, domain)
			}
		}

		configs[0].DomainServers = defaultDomains
		return configs[0], nil
	} else {
		return DefaultDhcpConfig, nil
	}
}
