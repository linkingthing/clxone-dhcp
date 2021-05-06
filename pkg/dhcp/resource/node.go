package resource

import (
	"time"

	"github.com/zdnscloud/gorest/resource"
)

type Node struct {
	resource.ResourceBase `json:",inline"`

	Ip           string    `json:"ip"`
	Ipv6s        []string  `json:"ipv6s"`
	Macs         []string  `json:"macs"`
	Roles        []string  `json:"roles"`
	HostName     string    `json:"hostName"`
	NodeIsAlive  bool      `json:"nodeIsAlive"`
	DhcpIsAlive  bool      `json:"dhcpIsAlive"`
	DnsIsAlive   bool      `json:"dnsIsAlive"`
	CpuRatio     string    `json:"cpuRatio"`
	MemRatio     string    `json:"memRatio"`
	Master       string    `json:"master"`
	ControllerIp string    `json:"controllerIP"`
	StartTime    time.Time `json:"startTime"`
	UpdateTime   time.Time `json:"updateTime"`
	Vip          string    `json:"vip"`
}
