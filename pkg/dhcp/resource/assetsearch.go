package resource

import (
	"time"

	restresource "github.com/zdnscloud/gorest/resource"
)

type AssetType string

const (
	AssetTypeIp     AssetType = "ip"
	AssetTypeDomain AssetType = "domain"
)

type AssetSearch struct {
	restresource.ResourceBase `json:",inline"`
	AssetType                 AssetType       `json:"assetType"`
	IpAsset                   IpAssetInfo     `json:"ipAsset"`
	DomainAsset               DomainAssetInfo `json:"domainAsset"`
}

type IpAssetInfo struct {
	Subnet             SubnetInfo         `json:"subnet,omitempty"`
	IpState            IpStateInfo        `json:"ipState,omitempty"`
	Device             DeviceInfo         `json:"device,omitempty"`
	AllocatedHistories AllocatedHistories `json:"allocatedHistories,omitempty"`
	BrowsedHistories   []BrowsedHistory   `json:"browsedHistories,omitempty"`
}

type SubnetInfo struct {
	Subnet       string `json:"subnet,omitempty"`
	NetworkType  string `json:"networkType,omitempty"`
	SemanticName string `json:"semanticName,omitempty"`
}

type IpStateInfo struct {
	IpType  string `json:"ipType,omitempty"`
	IpState string `json:"ipState,omitempty"`
}

type DeviceInfo struct {
	Name              string `json:"name,omitempty"`
	DeviceType        string `json:"deviceType,omitempty"`
	Mac               string `json:"mac,omitempty"`
	UplinkEquipment   string `json:"uplinkEquipment,omitempty"`
	UplinkPort        string `json:"uplinkPort,omitempty"`
	VlanId            int    `json:"vlanId,omitempty"`
	ComputerRoom      string `json:"computerRoom,omitempty"`
	ComputerRack      string `json:"computerRack,omitempty"`
	DeployedService   string `json:"deployedService,omitempty"`
	Department        string `json:"department,omitempty"`
	ResponsiblePerson string `json:"responsiblePerson,omitempty"`
	Telephone         string `json:"telephone,omitempty"`
}

type BrowsedDomain struct {
	Domain string `json:"domain,omitempty"`
	Count  uint64 `json:"count,omitempty"`
}

type AllocatedHistory struct {
	Mac     string               `json:"mac,omitempty"`
	IpType  string               `json:"ipType,omitempty"`
	IpState string               `json:"ipState,omitempty"`
	Time    restresource.ISOTime `json:"time,omitempty"`
}

type AllocatedHistories []AllocatedHistory

func (a AllocatedHistories) Len() int {
	return len(a)
}

func (a AllocatedHistories) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a AllocatedHistories) Less(i, j int) bool {
	return time.Time(a[i].Time).After(time.Time(a[j].Time))
}

type BrowsedHistory struct {
	BrowsedDomain BrowsedDomain `json:"browsedDomain,omitempty"`
	BrowserTopIps []TopIpInfo   `json:"browserTopIps,omitempty"`
}

type DomainAssetInfo struct {
	AuthRRs AuthRRs    `json:"authrrs,omitempty"`
	Ips     BrowserIps `json:"ips,omitempty"`
}

type AuthRR struct {
	View   string `json:"view,omitempty"`
	Zone   string `json:"zone,omitempty"`
	RRName string `json:"rrName,omitempty"`
	RRType string `json:"rrType,omitempty"`
	TTL    uint32 `json:"ttl,omitempty"`
	Rdata  string `json:"rdata,omitempty"`
}

type AuthRRs []AuthRR

func (rrs AuthRRs) Len() int {
	return len(rrs)
}

func (rrs AuthRRs) Swap(i, j int) {
	rrs[i], rrs[j] = rrs[j], rrs[i]
}

func (rrs AuthRRs) Less(i, j int) bool {
	if rrs[i].View == rrs[j].View {
		return rrs[i].Zone < rrs[j].Zone
	} else {
		return rrs[i].View < rrs[j].View
	}
}

type BrowserIp struct {
	TopIp          TopIpInfo       `json:"topIp,omitempty"`
	Subnet         SubnetInfo      `json:"subnet,omitempty"`
	IpState        IpStateInfo     `json:"state,omitempty"`
	Device         DeviceInfo      `json:"device,omitempty"`
	BrowsedDomains []BrowsedDomain `json:"browsedDomains,omitempty"`
}

type TopIpInfo struct {
	Ip    string `json:"ip,omitempty"`
	Count uint64 `json:"count,omitempty"`
}

type BrowserIps []BrowserIp

func (ips BrowserIps) Len() int {
	return len(ips)
}

func (ips BrowserIps) Swap(i, j int) {
	ips[i], ips[j] = ips[j], ips[i]
}

func (ips BrowserIps) Less(i, j int) bool {
	if ips[i].TopIp.Count == ips[j].TopIp.Count {
		return ips[i].TopIp.Ip < ips[j].TopIp.Ip
	} else {
		return ips[i].TopIp.Count > ips[j].TopIp.Count
	}
}
