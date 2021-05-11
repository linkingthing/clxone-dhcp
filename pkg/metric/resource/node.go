package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type Node struct {
	resource.ResourceBase `json:",inline"`

	Ip    string   `json:"ip"`
	Ipv6s []string `json:"ipv6s"`
}