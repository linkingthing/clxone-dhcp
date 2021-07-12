package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type LeaseTotal struct {
	restresource.ResourceBase `json:",inline"`
	Values                    []ValueWithTimestamp `json:"values"`
}

func (l LeaseTotal) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{DhcpServer{}}
}

func (l LeaseTotal) GetActions() []restresource.Action {
	return exportActions
}
