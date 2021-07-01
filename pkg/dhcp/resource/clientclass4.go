package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

var TableClientClass4 = restdb.ResourceDBType(&ClientClass4{})

type ClientClass4 struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" rest:"required=true,description=immutable" db:"uk"`
	Code                      uint32 `json:"code" rest:"required=true"`
	Regexp                    string `json:"regexp" rest:"required=true"`
}

type ClientClass4s []*ClientClass4

func (c ClientClass4s) Len() int {
	return len(c)
}

func (c ClientClass4s) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c ClientClass4s) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}
