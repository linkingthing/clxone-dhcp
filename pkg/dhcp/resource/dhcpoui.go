package resource

import (
	"fmt"
	"net"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableDhcpOui = restdb.ResourceDBType(&DhcpOui{})

type DhcpOui struct {
	restresource.ResourceBase `json:",inline"`
	Oui                       string `json:"oui" rest:"required=true"`
	Organization              string `json:"organization" rest:"required=true"`
	IsReadOnly                bool   `json:"isReadOnly"`
}

func (d *DhcpOui) Validate() error {
	if _, err := net.ParseMAC(d.Oui + ":00:00:00"); err != nil {
		return fmt.Errorf("invlaid oui %s, it should be prefix 24bit of mac", d.Oui)
	} else if len(d.Organization) == 0 {
		return fmt.Errorf("oui organization is required")
	} else {
		d.IsReadOnly = false
		return util.ValidateStrings(d.Organization)
	}
}
