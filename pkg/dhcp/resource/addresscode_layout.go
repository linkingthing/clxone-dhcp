package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

var TableAddressCodeLayout = restdb.ResourceDBType(&AddressCodeLayout{})

type LabelType string

const (
	LabelTypeAssetType         LabelType = "asset_type"
	LabelTypeManufacturer      LabelType = "manufacturer"
	LabelTypeModel             LabelType = "model"
	LabelTypeAccessNetworkTime LabelType = "access_network_time"
)

func (l LabelType) Validate() bool {
	return l == LabelTypeAssetType || l == LabelTypeManufacturer ||
		l == LabelTypeModel || l == LabelTypeAccessNetworkTime
}

type AddressCodeLayout struct {
	restresource.ResourceBase `json:",inline"`
	AddressCode               string                      `json:"-" db:"ownby,uk"`
	Label                     LabelType                   `json:"label" db:"uk" rest:"required=true"`
	BeginBit                  uint32                      `json:"beginBit" rest:"required=true"`
	EndBit                    uint32                      `json:"endBit" rest:"required=true"`
	Segments                  []*AddressCodeLayoutSegment `json:"segments" db:"-"`
}

func (a AddressCodeLayout) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{AddressCode{}}
}

func (a *AddressCodeLayout) Validate() error {
	if !a.Label.Validate() {
		return errorno.ErrNotInScope(errorno.ErrNameLabel,
			string(errorno.ErrNameAssetType), string(errorno.ErrNameManufacturer),
			string(errorno.ErrNameModel), string(errorno.ErrNameAccessNetworkTime))
	}

	if a.BeginBit < 65 || a.BeginBit > 128 ||
		a.EndBit < a.BeginBit || a.EndBit > 128 || a.EndBit%4 != 0 {
		return errorno.ErrInvalidAddressCode()
	}

	return nil
}
