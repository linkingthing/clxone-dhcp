package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type Lps struct {
	restresource.ResourceBase `json:",inline"`
	Values                    []ValueWithTimestamp `json:"values"`
}

type ValueWithTimestamp struct {
	Timestamp restresource.ISOTime `json:"timestamp"`
	Value     uint64               `json:"value"`
}

func (l Lps) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{DhcpSentry{}}
}

func (l Lps) GetActions() []restresource.Action {
	return exportActions
}
