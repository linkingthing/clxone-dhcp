package resource

import restresource "github.com/zdnscloud/gorest/resource"

type AppTrend struct {
	restresource.ResourceBase `json:",inline"`
	Domain                    string          `json:"domain,omitempty"`
	Hits                      uint64          `json:"hits,omitempty"`
	Trend                     []*AppTrendCell `json:"trend,omitempty"`
}

type AppTrendCell struct {
	Date string `json:"date,omitempty"`
	Hits uint64 `json:"hits,omitempty"`
}
