package resource

import (
	restresource "github.com/linkingthing/gorest/resource"
)

type LeaseTotal struct {
	restresource.ResourceBase `json:",inline"`
	NodeName                  string               `json:"nodeName"`
	Values                    []ValueWithTimestamp `json:"values"`
}

func (l LeaseTotal) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{DhcpServer{}}
}

func (l LeaseTotal) GetActions() []restresource.Action {
	return exportActions
}
