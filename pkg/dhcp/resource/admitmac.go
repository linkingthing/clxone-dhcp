package resource

import (
	"net"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
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
	if _, err := net.ParseMAC(a.HwAddress); err != nil {
		return err
	} else {
		return util.ValidateStrings(a.Comment)
	}
}
