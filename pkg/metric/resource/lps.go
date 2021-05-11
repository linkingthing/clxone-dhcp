package resource

import restresource "github.com/zdnscloud/gorest/resource"

const ResourceIDLPS = "lps"

type Lps struct {
	Values []ValueWithTimestamp `json:"values"`
}

type ValueWithTimestamp struct {
	Timestamp restresource.ISOTime `json:"timestamp"`
	Value     uint64               `json:"value"`
}
