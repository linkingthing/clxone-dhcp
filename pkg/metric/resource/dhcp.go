package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type DhcpSentry struct {
	restresource.ResourceBase `json:",inline"`
}

type DhcpServer struct {
	restresource.ResourceBase `json:",inline"`
}
