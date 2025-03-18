package resource

import (
	"net"
	"strings"

	"github.com/linkingthing/clxone-utils/excel"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableDhcpOui = restdb.ResourceDBType(&DhcpOui{})

type DhcpOui struct {
	restresource.ResourceBase `json:",inline"`
	Oui                       string     `json:"oui" rest:"required=true" db:"uk"`
	Organization              string     `json:"organization" rest:"required=true"`
	DataSource                DataSource `json:"dataSource"`
}

func (d *DhcpOui) Validate() error {
	if _, err := net.ParseMAC(d.Oui + ":00:00:00"); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameOui, d.Oui)
	} else if len(d.Organization) == 0 {
		return errorno.ErrEmpty(string(errorno.ErrNameOrganization))
	} else {
		d.Oui = strings.ToUpper(d.Oui)
		d.DataSource = DataSourceManual
		return util.ValidateStrings(util.RegexpTypeSpace, d.Organization)
	}
}

type DhcpOuis struct {
	Ids []string `json:"ids"`
}

func (d DhcpOui) GetActions() []restresource.Action {
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
			Input: &DhcpOuis{},
		},
	}
}
