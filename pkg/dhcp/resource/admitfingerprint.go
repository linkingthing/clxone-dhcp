package resource

import (
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"
)

var TableAdmitFingerprint = restdb.ResourceDBType(&AdmitFingerprint{})

type AdmitFingerprint struct {
	restresource.ResourceBase `json:",inline"`
	ClientType                string `json:"clientType" rest:"required=true"`
}

func (a AdmitFingerprint) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Admit{}}
}
