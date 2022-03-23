package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TableAdmitDuid = restdb.ResourceDBType(&AdmitDuid{})

type AdmitDuid struct {
	restresource.ResourceBase `json:",inline"`
	Duid                      string `json:"duid" rest:"required=true"`
	Comment                   string `json:"comment"`
}

const (
	FieldDuid = "duid"
)

func (a AdmitDuid) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Admit{}}
}

func (a *AdmitDuid) Validate() error {
	if err := parseDUID(a.Duid); err != nil {
		return err
	} else {
		return nil
	}
}
