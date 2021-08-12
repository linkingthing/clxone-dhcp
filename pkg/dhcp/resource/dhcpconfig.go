package resource

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableDhcpConfig = restdb.ResourceDBType(&DhcpConfig{})

const (
	MinValidLifetime = 3600

	DefaultMinValidLifetime uint32 = 10800
	DefaultMaxValidLifetime uint32 = 28800
	DefaultValidLifetime    uint32 = 14400
)

type DhcpConfig struct {
	restresource.ResourceBase `json:",inline"`
	Identify                  string   `json:"-" db:"uk"`
	ValidLifetime             uint32   `json:"validLifetime"`
	MaxValidLifetime          uint32   `json:"maxValidLifetime"`
	MinValidLifetime          uint32   `json:"minValidLifetime"`
	DomainServers             []string `json:"domainServers"`
}

func (config *DhcpConfig) Validate() error {
	if err := util.CheckIPsValid(config.DomainServers...); err != nil {
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
