package resource

import (
	"fmt"
	"net"

	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

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
		return err
	}

	if (a.HwAddress == "" && a.Duid == "") || (a.HwAddress != "" && a.Duid != "") {
		return fmt.Errorf("hw-address %s and duid %s must has only one",
			a.HwAddress, a.Duid)
	}

	if a.HwAddress != "" {
		if _, err := net.ParseMAC(a.HwAddress); err != nil {
			return err
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
		return fmt.Errorf("address code missing code")
	}

	if a.CodeBegin < 65 || a.CodeBegin > 128 ||
		a.CodeEnd < a.CodeBegin || a.CodeEnd > 128 || a.CodeEnd%4 != 0 {
		return fmt.Errorf("address code begin %d must in [65, 128] and end %d must in [68 72 76 80 84 88 92 96 100 104 108 112 116 120 124 128]",
			a.CodeBegin, a.CodeEnd)
	}

	if a.CodeEnd-a.CodeBegin+1 != uint32(len(a.Code))*4-(3-(a.CodeEnd-a.CodeBegin)%4) {
		return fmt.Errorf("code %s length no match with begin %d and end %d",
			a.Code, a.CodeBegin, a.CodeEnd)
	}

	return nil
}
