package resource

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableAdmitDuid = restdb.ResourceDBType(&AdmitDuid{})

type AdmitDuid struct {
	restresource.ResourceBase `json:",inline"`
	Duid                      string `json:"duid" rest:"required=true" db:"uk"`
	Comment                   string `json:"comment"`
}

func (a AdmitDuid) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Admit{}}
}

func (a *AdmitDuid) Validate() error {
	if err := parseDUID(a.Duid); err != nil {
		return err
	}
	if err := util.ValidateStrings(util.RegexpTypeComma, a.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, a.Comment)
	}
	return nil
}
