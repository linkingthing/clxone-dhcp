package resource

import (
	"net"

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

func (r RateLimitMac) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{RateLimit{}}
}

func (r *RateLimitMac) Validate() error {
	_, err := net.ParseMAC(r.HwAddress)
	return err
}
