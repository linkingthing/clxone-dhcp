package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TableClientClass6 = restdb.ResourceDBType(&ClientClass6{})

type ClientClass6 struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" rest:"required=true,description=immutable" db:"uk"`
	Regexp                    string `json:"regexp" rest:"required=true"`
}

func (c *ClientClass6) Validate() error {
	if len(c.Name) == 0 || len(c.Regexp) == 0 {
		return ErrNameOrRegexpMissing
	} else {
		return nil
	}
}
