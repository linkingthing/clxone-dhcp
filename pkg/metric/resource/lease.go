package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type Lease struct {
	restresource.ResourceBase `json:",inline"`
	Values                    []ValueWithTimestamp `json:"values"`
}

func (l Lease) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Dhcp{}}
}

func (l Lease) GetActions() []restresource.Action {
	return exportActions
}
