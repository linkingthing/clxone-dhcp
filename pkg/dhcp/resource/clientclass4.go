package resource

import (
	"fmt"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableClientClass4 = restdb.ResourceDBType(&ClientClass4{})

type ClientClass4 struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" rest:"required=true,description=immutable" db:"uk"`
	Regexp                    string `json:"regexp" rest:"required=true"`
}

var ErrNameOrRegexpMissing = fmt.Errorf("clientclass name and regexp are required")

func (c *ClientClass4) Validate() error {
	if len(c.Name) == 0 || len(c.Regexp) == 0 {
		return errorno.ErrEmpty(string(errorno.ErrNameName), string(errorno.ErrNameRegexp))
	}
	return util.ValidateStrings(c.Name, c.Regexp)
}
