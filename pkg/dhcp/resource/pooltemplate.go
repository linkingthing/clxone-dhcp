package resource

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TablePoolTemplate = restdb.ResourceDBType(&PoolTemplate{})

type PoolTemplate struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string         `json:"name" rest:"required=true" db:"uk"`
	Version                   util.IPVersion `json:"version" rest:"required=true"`
	BeginOffset               uint64         `json:"beginOffset" rest:"required=true"`
	Capacity                  uint64         `json:"capacity" rest:"required=true"`
	DomainServers             []string       `json:"domainServers"`
	Routers                   []string       `json:"routers"`
	ClientClass               string         `json:"clientClass"`
	Comment                   string         `json:"comment"`
}

func (p *PoolTemplate) Validate() error {
	if p.Version.Validate() == false {
		return fmt.Errorf("invalid ip version %v, it only support 4 or 6", p.Version)
	}

	if p.Version == util.IPVersion4 {
		if p.BeginOffset <= 0 || p.BeginOffset >= 65535 || p.Capacity <= 0 || p.Capacity >= 65535 {
			return fmt.Errorf("offset %v or capacity %v should in (0, 65535)", p.BeginOffset, p.Capacity)
		}
	} else {
		if p.BeginOffset <= 0 || p.BeginOffset >= 2147483647 || p.Capacity <= 0 || p.Capacity >= 2147483647 {
			return fmt.Errorf("offset %v or capacity %v should in (0, 2147483647)", p.BeginOffset, p.Capacity)
		}
	}

	return checkCommonOptions(p.Version, p.ClientClass, p.DomainServers, p.Routers)
}
