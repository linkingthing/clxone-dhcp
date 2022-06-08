package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableRateLimitDuid = restdb.ResourceDBType(&RateLimitDuid{})

type RateLimitDuid struct {
	restresource.ResourceBase `json:",inline"`
	Duid                      string `json:"duid" rest:"required=true"`
	RateLimit                 uint32 `json:"rateLimit" rest:"required=true"`
	Comment                   string `json:"comment"`
}

func (r RateLimitDuid) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{RateLimit{}}
}

func (r *RateLimitDuid) Validate() error {
	if err := parseDUID(r.Duid); err != nil {
		return err
	} else {
		return util.ValidateStrings(r.Comment)
	}
}
