package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableAdmitMac = restdb.ResourceDBType(&AdmitMac{})

type AdmitMac struct {
	restresource.ResourceBase `json:",inline"`
	HwAddress                 string `json:"hwAddress" rest:"required=true" db:"uk"`
	Comment                   string `json:"comment"`
}

func (a AdmitMac) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Admit{}}
}

func (a *AdmitMac) Validate() error {
	if hw, err := util.NormalizeMac(a.HwAddress); err != nil {
		return err
	} else {
		a.HwAddress = hw
	}
	if err := util.ValidateStrings(util.RegexpTypeComma, a.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, a.Comment)
	}
	return nil
}
