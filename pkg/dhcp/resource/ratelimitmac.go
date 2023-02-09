package resource

import (
	"net"
	"strings"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableRateLimitMac = restdb.ResourceDBType(&RateLimitMac{})

type RateLimitMac struct {
	restresource.ResourceBase `json:",inline"`
	HwAddress                 string `json:"hwAddress" rest:"required=true"`
	RateLimit                 uint32 `json:"rateLimit" rest:"required=true"`
	Comment                   string `json:"comment"`
}

func (r RateLimitMac) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{RateLimit{}}
}

func (r *RateLimitMac) Validate() error {
	if err := util.ValidateStrings(r.Comment); err != nil {
<<<<<<< HEAD
		return errorno.ErrInvalidParams(errorno.ErrNameComment, r.Comment)
	} else {
		_, err = net.ParseMAC(r.HwAddress)
		if err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameMac, r.HwAddress)
		}
		return nil
=======
		return err
	} else if hw, err := net.ParseMAC(r.HwAddress); err != nil {
		return err
	} else {
		r.HwAddress = strings.ToUpper(hw.String())
>>>>>>> dev-2.3
	}
	return nil
}
