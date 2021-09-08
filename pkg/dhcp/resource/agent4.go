package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type Agent4 struct {
	restresource.ResourceBase `json:",inline"`
	Ip                        string `json:"ip"`
}
