package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableAdmitFingerprint = restdb.ResourceDBType(&AdmitFingerprint{})

type AdmitFingerprint struct {
	restresource.ResourceBase `json:",inline"`
	ClientType                string `json:"clientType" rest:"required=true" db:"uk"`
}

func (a AdmitFingerprint) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Admit{}}
}

func (a *AdmitFingerprint) Validate() error {
	if len(a.ClientType) == 0 || util.ValidateStrings(util.RegexpTypeCommon, a.ClientType) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameClientClass, a.ClientType)
	} else {
		return nil
	}
}
