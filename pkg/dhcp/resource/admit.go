package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TableAdmit = restdb.ResourceDBType(&Admit{})

var DefaultAdmit = &Admit{Enabled: false}

type Admit struct {
	restresource.ResourceBase `json:",inline"`
	Enabled                   bool `json:"enabled" rest:"required=true"`
}
