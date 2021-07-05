package resource

import (
	restresource "github.com/zdnscloud/gorest/resource"
)

type SubnetUsedRatio struct {
	restresource.ResourceBase `json:",inline"`
	Subnets                   []SubnetUsage `json:"subnets"`
}

type SubnetUsage struct {
	Subnet     string               `json:"subnet"`
	UsedRatios []RatioWithTimestamp `json:"usedRatios"`
}

type RatioWithTimestamp struct {
	Timestamp restresource.ISOTime `json:"timestamp"`
	Ratio     string               `json:"ratio"`
}

func (s SubnetUsedRatio) GetParents() []restresource.ResourceKind {
	return []restresource.ResourceKind{Dhcp{}}
}

func (s SubnetUsedRatio) GetActions() []restresource.Action {
	return exportActions
}

type SubnetUsages []SubnetUsage

func (s SubnetUsages) Len() int {
	return len(s)
}

func (s SubnetUsages) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SubnetUsages) Less(i, j int) bool {
	siUsedRatio := s[i].getFirstUsedRatio()
	sjUsedRatio := s[j].getFirstUsedRatio()
	if siUsedRatio == sjUsedRatio {
		return s[i].Subnet < s[j].Subnet
	} else {
		return siUsedRatio < sjUsedRatio
	}
}

func (s SubnetUsage) getFirstUsedRatio() string {
	for _, u := range s.UsedRatios {
		return u.Ratio
	}

	return ""
}
