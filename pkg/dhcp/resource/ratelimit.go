package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TableRateLimit = restdb.ResourceDBType(&RateLimit{})

var DefaultRateLimit = &RateLimit{Enabled: false}

type RateLimit struct {
	restresource.ResourceBase `json:",inline"`
	Enabled                   bool `json:"enabled" rest:"required=true"`
}
