package resource

import (
	"net"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TableAdmitMac = restdb.ResourceDBType(&AdmitMac{})

type AdmitMac struct {
	restresource.ResourceBase `json:",inline"`
	HwAddress                 string `json:"hwAddress" rest:"required=true"`
	Comment                   string `json:"comment"`
}

func (a AdmitMac) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Admit{}}
}

func (a *AdmitMac) Validate() error {
	_, err := net.ParseMAC(a.HwAddress)
	return err
}
