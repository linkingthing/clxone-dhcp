package resource

import (
	"net"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TableDhcpOui = restdb.ResourceDBType(&DhcpOui{})

type DhcpOui struct {
	restresource.ResourceBase `json:",inline"`
	Oui                       string `json:"oui" rest:"required=true"`
	Organization              string `json:"organization" rest:"required=true"`
	IsReadOnly                bool   `json:"isReadOnly"`
}

func (d *DhcpOui) Validate() error {
	d.IsReadOnly = false
	_, err := net.ParseMAC(d.Oui + ":00:00:00")
	return err
}
