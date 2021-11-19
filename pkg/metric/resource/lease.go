package resource

import (
	restresource "github.com/linkingthing/gorest/resource"
)

type Lease struct {
	restresource.ResourceBase `json:",inline"`
	Subnets                   []SubnetLease `json:"subnets"`
}

type SubnetLease struct {
	Subnet string               `json:"subnet"`
	Values []ValueWithTimestamp `json:"values"`
}

func (l Lease) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{DhcpServer{}}
}

func (l Lease) GetActions() []restresource.Action {
	return exportActions
}
