package resource

import (
	"github.com/linkingthing/gorest/resource"
)

type Node struct {
	resource.ResourceBase `json:",inline"`

	Ip       string `json:"ip"`
	Hostname string `json:"hostname"`
}
