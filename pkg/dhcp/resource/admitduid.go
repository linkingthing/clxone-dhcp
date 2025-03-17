package resource

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-utils/excel"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableAdmitDuid = restdb.ResourceDBType(&AdmitDuid{})

type AdmitDuid struct {
	restresource.ResourceBase `json:",inline"`
	Duid                      string `json:"duid" rest:"required=true" db:"uk"`
	IsAdmitted                bool   `json:"isAdmitted"`
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

type AdmitDuids struct {
	Ids []string `json:"ids"`
}

func (a AdmitDuid) GetActions() []restresource.Action {
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
			Input: &AdmitDuids{},
		},
	}
}
