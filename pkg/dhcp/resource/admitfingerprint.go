package resource

import (
	"github.com/linkingthing/clxone-utils/excel"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableAdmitFingerprint = restdb.ResourceDBType(&AdmitFingerprint{})

type AdmitFingerprint struct {
	restresource.ResourceBase `json:",inline"`
	ClientType                string `json:"clientType" rest:"required=true" db:"uk"`
	IsAdmitted                bool   `json:"isAdmitted"`
	Comment                   string `json:"comment"`
}

func (a AdmitFingerprint) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Admit{}}
}

func (a *AdmitFingerprint) Validate() error {
	if len(a.ClientType) == 0 || util.ValidateStrings(util.RegexpTypeCommon, a.ClientType) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameClientClass, a.ClientType)
	} else if err := util.ValidateStrings(util.RegexpTypeComma, a.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, a.Comment)
	} else {
		return nil
	}
}

type AdmitFingerprints struct {
	Ids []string `json:"ids"`
}

func (a AdmitFingerprint) GetActions() []restresource.Action {
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
			Input: &AdmitFingerprints{},
		},
	}
}
