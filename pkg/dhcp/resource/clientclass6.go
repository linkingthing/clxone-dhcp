package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

var TableClientClass6 = restdb.ResourceDBType(&ClientClass6{})

type ClientClass6 struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" rest:"required=true,description=immutable" db:"uk"`
	Code                      uint32 `json:"code" rest:"required=true"`
	Regexp                    string `json:"regexp" rest:"required=true"`
}

type ClientClass6s []*ClientClass6

func (c ClientClass6s) Len() int {
	return len(c)
}

func (c ClientClass6s) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c ClientClass6s) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}
