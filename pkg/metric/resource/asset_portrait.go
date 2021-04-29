package resource

import (
	assetresource "github.com/linkingthing/clxone-dhcp/pkg/asset/resource"
	restresource "github.com/zdnscloud/gorest/resource"
)

type AssetPortrait struct {
	restresource.ResourceBase `json:",inline"`
	DeviceTotal               int64                  `json:"deviceTotal"`
	OnlineTotal               int64                  `json:"onlineTotal"`
	OfflineTotal              int64                  `json:"offlineTotal"`
	AbnormalTotal             int64                  `json:"abnormalTotal"`
	StateStatistics           []*AssetStateStatistic `json:"assetStateStatistic"`
}

type AssetStateStatistic struct {
	Region   string `json:"region"`
	Online   int64  `json:"online"`
	Offline  int64  `json:"offline"`
	Abnormal int64  `json:"abnormal"`
}

type AssetPortraitCount struct {
	restresource.ResourceBase `json:",inline"`
	Region                    string
	DeviceState               assetresource.DeviceState
	Count                     int64
}
