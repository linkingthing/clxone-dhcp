package resource

import (
	"github.com/linkingthing/clxone-utils/excel"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableRateLimitMac = restdb.ResourceDBType(&RateLimitMac{})

type RateLimitMac struct {
	restresource.ResourceBase `json:",inline"`
	HwAddress                 string `json:"hwAddress" rest:"required=true" db:"uk"`
	RateLimit                 uint32 `json:"rateLimit" rest:"required=true"`
	Comment                   string `json:"comment"`
}

func (r RateLimitMac) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{RateLimit{}}
}

func (r *RateLimitMac) Validate() error {
	if err := util.ValidateStrings(util.RegexpTypeComma, r.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, r.Comment)
	} else if hw, err := util.NormalizeMac(r.HwAddress); err != nil {
		return err
	} else {
		r.HwAddress = hw
	}
	return nil
}

type RateLimitMacs struct {
	Ids []string `json:"ids"`
}

func (r RateLimitMac) GetActions() []restresource.Action {
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
			Input: &RateLimitMacs{},
		},
	}
}
