package resource

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TablePool6Template = restdb.ResourceDBType(&Pool6Template{})

type Pool6Template struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" rest:"required=true" db:"uk"`
	BeginOffset               uint64 `json:"beginOffset" rest:"required=true"`
	Capacity                  uint64 `json:"capacity" rest:"required=true"`
	Comment                   string `json:"comment"`
}

func (p *Pool6Template) Validate() error {
	if len(p.Name) == 0 || util.ValidateStrings(p.Name) != nil {
		return fmt.Errorf("name %s is invalid", p.Name)
	} else if p.BeginOffset <= 0 || p.BeginOffset >= 2147483647 || p.Capacity <= 0 || p.Capacity >= 2147483647 {
		return fmt.Errorf("offset %v or capacity %v should in (0, 2147483647)", p.BeginOffset, p.Capacity)
	} else {
		return nil
	}
}
