package resource

import (
	"net"

	"github.com/linkingthing/clxone-utils/excel"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableAsset = restdb.ResourceDBType(&Asset{})

const (
	ActionNameBatchDelete = "batch_delete"

	AssetUnknownValue = "unknown"
)

type Asset struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" rest:"required=true"`
	HwAddress                 string `json:"hwAddress" db:"uk" rest:"required=true"`
	AssetType                 string `json:"assetType"`
	Manufacturer              string `json:"manufacturer"`
	Model                     string `json:"model"`
	AccessNetworkTime         string `json:"accessNetworkTime"`
}

type Assets struct {
	Ids []string `json:"ids"`
}

func (a Asset) GetActions() []restresource.Action {
	return []restresource.Action{
		restresource.Action{
			Name:  excel.ActionNameImport,
			Input: &excel.ImportFile{},
		},
		restresource.Action{
			Name:   excel.ActionNameExport,
			Output: &excel.ExportFile{},
		},
		restresource.Action{
			Name:   excel.ActionNameExportTemplate,
			Output: &excel.ExportFile{},
		},
		restresource.Action{
			Name:  ActionNameBatchDelete,
			Input: &Assets{},
		},
	}
}

func (a *Asset) Validate() error {
	if _, err := net.ParseMAC(a.HwAddress); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameMac, a.HwAddress)
	}

	if util.ValidateStrings(util.RegexpTypeCommon, a.Name) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameName, a.Name)
	}

	if util.ValidateStrings(util.RegexpTypeSpace, a.AssetType) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameAssetType, a.AssetType)
	} else if a.AssetType == "" {
		a.AssetType = AssetUnknownValue
	}

	if util.ValidateStrings(util.RegexpTypeSpace, a.Manufacturer) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameManufacturer, a.Manufacturer)
	} else if a.Manufacturer == "" {
		a.Manufacturer = AssetUnknownValue
	}

	if util.ValidateStrings(util.RegexpTypeSpace, a.Model) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameModel, a.Model)
	} else if a.Model == "" {
		a.Model = AssetUnknownValue
	}

	if util.ValidateStrings(util.RegexpTypeSpace, a.AccessNetworkTime) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameAccessNetworkTime, a.AccessNetworkTime)
	} else if a.AccessNetworkTime == "" {
		a.AccessNetworkTime = AssetUnknownValue
	}

	return nil
}

func (a *Asset) Diff(another *Asset) bool {
	return a.HwAddress != another.HwAddress ||
		a.AssetType != another.AssetType ||
		a.Manufacturer != another.Manufacturer ||
		a.Model != another.Model ||
		a.AccessNetworkTime != another.AccessNetworkTime
}

func (a *Asset) Equal(another *Asset) bool {
	return a.HwAddress == another.HwAddress &&
		a.AssetType == another.AssetType &&
		a.Manufacturer == another.Manufacturer &&
		a.Model == another.Model &&
		a.AccessNetworkTime == another.AccessNetworkTime
}
