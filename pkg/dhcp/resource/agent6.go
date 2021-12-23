package resource

import (
	restresource "github.com/linkingthing/gorest/resource"
)

type Agent6 struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name"`
	Ip                        string `json:"ip"`
}
