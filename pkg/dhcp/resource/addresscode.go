package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

var TableAddressCode = restdb.ResourceDBType(&AddressCode{})

type AddressCode struct {
	restresource.ResourceBase `json:",inline"`
	HwAddress                 string `json:"hwAddress" db:"uk"`
	Duid                      string `json:"duid" db:"uk"`
	Code                      string `json:"code" rest:"required=true"`
	CodeBegin                 uint32 `json:"codeBegin" rest:"required=true"`
	CodeEnd                   uint32 `json:"codeEnd" rest:"required=true"`
	Comment                   string `json:"comment"`
}

func (a *AddressCode) Validate() error {
	if err := util.ValidateStrings(util.RegexpTypeComma, a.Comment); err != nil {
		return errorno.ErrInvalidParams(errorno.ErrNameComment, a.Comment)
	}

	if (a.HwAddress == "" && a.Duid == "") || (a.HwAddress != "" && a.Duid != "") {
		return errorno.ErrOnlyOne(string(errorno.ErrNameMac), string(errorno.ErrNameDuid))
	}

	if a.HwAddress != "" {
		if hw, err := util.NormalizeMac(a.HwAddress); err != nil {
			return errorno.ErrInvalidParams(errorno.ErrNameMac, a.HwAddress)
		} else {
			a.HwAddress = hw
		}
	} else {
		if err := parseDUID(a.Duid); err != nil {
			return err
		}
	}

	return a.ValidateCode()
}

func (a *AddressCode) ValidateCode() error {
	if a.Code == "" {
		return errorno.ErrEmpty(string(errorno.ErrNameAddressCode))
	}

	if a.CodeBegin < 65 || a.CodeBegin > 128 ||
		a.CodeEnd < a.CodeBegin || a.CodeEnd > 128 || a.CodeEnd%4 != 0 {
		return errorno.ErrInvalidAddressCode()
	}

	if a.CodeEnd-a.CodeBegin+1 != uint32(len(a.Code))*4-(3-(a.CodeEnd-a.CodeBegin)%4) {
		return errorno.ErrMismatchAddressCode(a.Code, a.CodeBegin, a.CodeEnd)
	}

	return nil
}
