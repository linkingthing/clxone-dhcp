package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableAddressCodeLayout = restdb.ResourceDBType(&AddressCodeLayout{})

type AddressCodeLayout struct {
	restresource.ResourceBase `json:",inline"`
	AddressCode               string                      `json:"-" db:"ownby,uk"`
	Label                     string                      `json:"label" db:"uk" rest:"required=true"`
	BeginBit                  uint32                      `json:"beginBit" rest:"required=true"`
	EndBit                    uint32                      `json:"endBit" rest:"required=true"`
	Segments                  []*AddressCodeLayoutSegment `json:"segments" db:"-"`
}

func (a AddressCodeLayout) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{AddressCode{}}
}

func (a *AddressCodeLayout) Validate() error {
	if util.ValidateStrings(util.RegexpTypeCommon, a.Label) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameName, a.Label)
	}

	if a.BeginBit < 65 || a.BeginBit > 128 ||
		a.EndBit < a.BeginBit || a.EndBit > 128 || a.EndBit%4 != 0 {
		return errorno.ErrInvalidAddressCode()
	}

	return nil
}
