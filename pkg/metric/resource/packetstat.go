package resource

import (
	restresource "github.com/linkingthing/gorest/resource"
)

type PacketStat struct {
	restresource.ResourceBase `json:",inline"`
	Packets                   []Packet `json:"packets"`
}

type Packet struct {
	Type   string               `json:"type"`
	Values []ValueWithTimestamp `json:"values"`
}

func (p PacketStat) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{DhcpSentry{}}
}

func (p PacketStat) GetActions() []restresource.Action {
	return exportActions
}
