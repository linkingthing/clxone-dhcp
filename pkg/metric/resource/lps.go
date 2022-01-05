package resource

import (
	restresource "github.com/linkingthing/gorest/resource"
)

type Lps struct {
	restresource.ResourceBase `json:",inline"`
	NodeName                  string               `json:"nodeName"`
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
