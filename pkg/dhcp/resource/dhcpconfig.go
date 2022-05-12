package resource

import (
	"fmt"

	gohelperip "github.com/cuityhj/gohelper/ip"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
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
}

func (config *DhcpConfig) Validate() error {
	if err := gohelperip.CheckIPsValid(config.DomainServers...); err != nil {
		return err
	}

	return checkLifetimeValid(config.ValidLifetime, config.MinValidLifetime, config.MaxValidLifetime)
}

func checkLifetimeValid(validLifetime, minValidLifetime, maxValidLifetime uint32) error {
	if minValidLifetime < MinValidLifetime {
		return fmt.Errorf("min-lifetime %d must not less than %d", minValidLifetime, MinValidLifetime)
	}

	if minValidLifetime > maxValidLifetime {
		return fmt.Errorf("min-lifetime must less than max-lifetime")
	}

	if validLifetime < minValidLifetime || validLifetime > maxValidLifetime {
		return fmt.Errorf("default lifetime %d is not between min-lifttime %d and max-lifetime %d",
			validLifetime, minValidLifetime, maxValidLifetime)
	}

	return nil
}

func getDhcpConfig(isv4 bool) (*DhcpConfig, error) {
	var configs []*DhcpConfig
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(nil, &configs)
	}); err != nil {
		return nil, fmt.Errorf("get dhcp global config failed: %s", pg.Error(err).Error())
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
