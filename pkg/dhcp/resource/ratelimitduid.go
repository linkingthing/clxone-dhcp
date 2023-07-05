package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableRateLimitDuid = restdb.ResourceDBType(&RateLimitDuid{})

type RateLimitDuid struct {
	restresource.ResourceBase `json:",inline"`
	Duid                      string `json:"duid" rest:"required=true" db:"uk"`
	RateLimit                 uint32 `json:"rateLimit" rest:"required=true"`
	Comment                   string `json:"comment"`
}

func (r RateLimitDuid) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{RateLimit{}}
}

func (r *RateLimitDuid) Validate() error {
	if err := parseDUID(r.Duid); err != nil {
		return err
	}
	if err := util.ValidateStrings(util.RegexpTypeComma, r.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, r.Comment)
	}
	return nil
}
