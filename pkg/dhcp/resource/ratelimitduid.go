package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TableRateLimitDuid = restdb.ResourceDBType(&RateLimitDuid{})

type RateLimitDuid struct {
	restresource.ResourceBase `json:",inline"`
	Duid                      string `json:"duid" rest:"required=true"`
	Limit                     uint32 `json:"limit" rest:"required=true"`
	Comment                   string `json:"comment"`
}

func (a RateLimitDuid) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{RateLimit{}}
}
