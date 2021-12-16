package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TableRateLimitMac = restdb.ResourceDBType(&RateLimitMac{})

type RateLimitMac struct {
	restresource.ResourceBase `json:",inline"`
	HwAddress                 string `json:"hwAddress" rest:"required=true"`
	RateLimit                 uint32 `json:"rateLimit" rest:"required=true"`
	Comment                   string `json:"comment"`
}

func (a RateLimitMac) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{RateLimit{}}
}
