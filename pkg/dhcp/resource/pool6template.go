package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
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
	if len(p.Name) == 0 || util.ValidateStrings(util.RegexpTypeCommon, p.Name) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameName, p.Name)
	} else if p.BeginOffset <= 0 || p.BeginOffset >= 2147483647 {
		return errorno.ErrNotInRange(errorno.ErrNameOffset, 1, 2147483646)
	} else if p.Capacity <= 0 || p.Capacity >= 2147483647 {
		return errorno.ErrNotInRange(errorno.ErrNameCapacity, 1, 2147483646)
	} else if err := util.ValidateStrings(util.RegexpTypeComma, p.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, p.Comment)
	}
	return nil
}
