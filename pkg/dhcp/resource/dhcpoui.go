package resource

import (
	"net"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableDhcpOui = restdb.ResourceDBType(&DhcpOui{})

type DhcpOui struct {
	restresource.ResourceBase `json:",inline"`
	Oui                       string `json:"oui" rest:"required=true" db:"uk"`
	Organization              string `json:"organization" rest:"required=true"`
	IsReadOnly                bool   `json:"isReadOnly"`
}

func (d *DhcpOui) Validate() error {
	if _, err := net.ParseMAC(d.Oui + ":00:00:00"); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameOui, d.Oui)
	} else if len(d.Organization) == 0 {
		return errorno.ErrEmpty(string(errorno.ErrNameOrganization))
	} else {
		d.IsReadOnly = false
		return util.ValidateStrings(util.RegexpTypeCommon, d.Organization)
	}
}
