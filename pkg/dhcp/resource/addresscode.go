package resource

import (
	"unicode/utf8"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableAddressCode = restdb.ResourceDBType(&AddressCode{})

type AddressCode struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" db:"uk" rest:"required=true"`
	Comment                   string `json:"comment"`
}

func (a *AddressCode) Validate() error {
	if util.ValidateStrings(util.RegexpTypeCommon, a.Name) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameName, a.Name)
	}

	if util.ValidateStrings(util.RegexpTypeComma, a.Comment) != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, a.Comment)
	} else if utf8.RuneCountInString(a.Comment) > MaxCommentLength {
		return errorno.ErrExceedMaxCount(errorno.ErrNameComment, MaxCommentLength)
	}

	return nil
}
