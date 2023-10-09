package resource

import (
	"net"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableAsset = restdb.ResourceDBType(&Asset{})

type Asset struct {
	restresource.ResourceBase `json:",inline"`
	HwAddress                 string `json:"hwAddress" db:"uk" rest:"required=true"`
	AssetType                 string `json:"assetType" rest:"required=true"`
	Manufacturer              string `json:"manufacturer" rest:"required=true"`
	Model                     string `json:"model" rest:"required=true"`
	AccessNetworkTime         string `json:"accessNetworkTime" rest:"required=true"`
}

func (a *Asset) Validate() error {
	if _, err := net.ParseMAC(a.HwAddress); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameMac, a.HwAddress)
	}

	if util.ValidateStrings(util.RegexpTypeCommon, a.AssetType) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameAssetType, a.AssetType)
	}

	if util.ValidateStrings(util.RegexpTypeCommon, a.Manufacturer) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameManufacturer, a.Manufacturer)
	}

	if util.ValidateStrings(util.RegexpTypeCommon, a.Model) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameModel, a.Model)
	}

	if util.ValidateStrings(util.RegexpTypeCommon, a.AccessNetworkTime) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameAccessNetworkTime, a.AccessNetworkTime)
	}

	return nil
}
