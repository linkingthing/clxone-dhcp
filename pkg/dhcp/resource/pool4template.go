package resource

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

var TablePool4Template = restdb.ResourceDBType(&Pool4Template{})

type Pool4Template struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" rest:"required=true" db:"uk"`
	BeginOffset               uint64 `json:"beginOffset" rest:"required=true"`
	Capacity                  uint64 `json:"capacity" rest:"required=true"`
	Comment                   string `json:"comment"`
}

func (p *Pool4Template) Validate() error {
	if p.BeginOffset <= 0 || p.BeginOffset >= 65535 || p.Capacity <= 0 || p.Capacity >= 65535 {
		return fmt.Errorf("offset %v or capacity %v should in (0, 65535)", p.BeginOffset, p.Capacity)
	} else {
		return nil
	}
}