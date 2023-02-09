package resource

import (
	"net"
	"strings"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
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
	if hw, err := net.ParseMAC(a.HwAddress); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameMac, a.HwAddress)
	} else {
		a.HwAddress = strings.ToUpper(hw.String())
		return util.ValidateStrings(a.Comment)
	}
}
