package resource

import (
	"strconv"
	"strings"

	"github.com/linkingthing/clxone-utils/excel"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableAddressCodeLayoutSegment = restdb.ResourceDBType(&AddressCodeLayoutSegment{})

type AddressCodeLayoutSegment struct {
	restresource.ResourceBase `json:",inline"`
	AddressCodeLayout         string `json:"-" db:"ownby,uk"`
	Code                      string `json:"code" db:"uk" rest:"required=true"`
	Value                     string `json:"value" rest:"required=true"`
}

func (a AddressCodeLayoutSegment) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{AddressCodeLayout{}}
}

type AddressCodeLayoutSegments struct {
	Codes []string `json:"codes"`
}

func (a AddressCodeLayoutSegment) GetActions() []restresource.Action {
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
			Input: &AddressCodeLayoutSegments{},
		},
	}
}

func (a *AddressCodeLayoutSegment) Validate(layout *AddressCodeLayout) error {
	if util.ValidateStrings(util.RegexpTypeSpace, a.Value) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameCode, a.Value)
	}

	if a.Code == "" {
		return errorno.ErrEmpty(string(errorno.ErrNameAddressCode))
	}

	for i := range a.Code {
		if _, err := strconv.ParseUint(a.Code[i:i+1], 16, 4); err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameAddressCode, a.Code)
		}
	}

	if layout.EndBit-layout.BeginBit+1 != uint32(len(a.Code))*4-(3-(layout.EndBit-layout.BeginBit)%4) {
		return errorno.ErrMismatchAddressCode(a.Code, layout.BeginBit, layout.EndBit)
	}

	a.Code = strings.ToLower(a.Code)
	return nil
}
