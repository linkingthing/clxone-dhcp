package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TablePinger = restdb.ResourceDBType(&Pinger{})

var DefaultPinger = &Pinger{
	Enabled: false,
}

type Pinger struct {
	restresource.ResourceBase `json:",inline"`
	Enabled                   bool   `json:"enabled" rest:"required=true"`
	Timeout                   uint32 `json:"timeout" rest:"required=true"`
}
