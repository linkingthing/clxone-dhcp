package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type Dhcp struct {
	restresource.ResourceBase `json:",inline"`
}
