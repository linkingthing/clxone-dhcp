package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type RegionPortrait struct {
	restresource.ResourceBase `json:",inline"`
	Province                  string `json:"province,omitempty"`
	Hits                      uint64 `json:"hits,omitempty"`
}
