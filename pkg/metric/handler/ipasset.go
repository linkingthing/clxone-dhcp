package handler

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"

	"github.com/zdnscloud/cement/errgroup"
	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	assetresource "github.com/linkingthing/clxone-dhcp/pkg/asset/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	dhcpresource "github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/esclient"
	ipamresource "github.com/linkingthing/clxone-dhcp/pkg/ipam/resource"
	metricresource "github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	TopBrowedHistory  = "topBrowedHistory"
	FilterNamePeriod  = "period"
	FilterNameTopSize = "top"
)

func getIpAssetInfo(ctx *restresource.Context, ip net.IP) (interface{}, *resterr.APIError) {
	ipstr := ip.String()
	var auditAssets []*assetresource.AuditAsset
	var devices []*assetresource.Device
	var nodes []*metricresource.Node
	var subnets []*dhcpresource.Subnet
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{
			"ip": ipstr, "orderby": "update_time desc", "offset": 0, "limit": 10,
		}, &auditAssets); err != nil {
			return err
		}

		var subnetConds map[string]interface{}
		var assetConds map[string]interface{}
		if len(auditAssets) != 0 {
			subnetConds = map[string]interface{}{"id": auditAssets[0].Subnet}
			assetConds = map[string]interface{}{"mac": auditAssets[0].Mac}
		}

		if err := tx.Fill(subnetConds, &subnets); err != nil {
			return err
		}

		if err := tx.Fill(assetConds, &devices); err != nil {
			return err
		}

		return tx.Fill(nil, &nodes)
	}); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError,
			fmt.Sprintf("list ip portraits from db failed: %s", err.Error()))
	}

	ipAsset := metricresource.IpAssetInfo{}
	networkInfos, err := ipamresource.FindNetworkInfos(ipstr)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError,
			fmt.Sprintf("find network info with ip %s failed: %s", ipstr, err.Error()))
	}

	if networkInfo, ok := networkInfos[ipstr]; ok {
		ipAsset.Subnet = ipamNetworkInfoToMetricSubnetInfo(networkInfo)
	}

	if ipAsset.Subnet.Subnet == "" {
		if subnet, ok := getClosestSubnet(subnets, ip); ok {
			ipAsset.Subnet = dhcpSubnetToMetricSubnetInfo(subnet)
		}
	}

	for i, auditAsset := range auditAssets {
		if i == 0 {
			ipAsset.IpState = auditAssetToIpStateInfo(auditAssets[0])
		}

		if auditAsset.Mac != "" {
			ipAsset.AllocatedHistories = append(ipAsset.AllocatedHistories,
				metricresource.AllocatedHistory{
					Mac:     auditAsset.Mac,
					IpType:  string(auditAsset.IpType),
					IpState: string(auditAsset.IpState),
					Time:    restresource.ISOTime(auditAsset.UpdateTime),
				})
		}
	}

	sort.Sort(ipAsset.AllocatedHistories)
	isv4 := ip.To4() != nil
	for _, device := range devices {
		if device.ContainsIp(ipstr, isv4) {
			ipAsset.Device = deviceToAssetInfo(device)
			break
		}
	}

	esTimeRange := getElasticsearchMustRange(ctx.GetFilters())
	resp, err := requestElasticsearch(TopBrowedHistory, esclient.DomainKeyWord,
		esclient.ElasticsearchMustMatch{Match: map[string]string{esclient.SrcIpKeyWord: ipstr}},
		esTimeRange, getTopSize(ctx.GetFilters()), nodes)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError,
			fmt.Sprintf("request elasticsearch failed: %s", err.Error()))
	}

	if resp != nil {
		if tops, ok := resp.Aggregations[TopBrowedHistory]; ok {
			for _, bucket := range tops.Buckets {
				ipAsset.BrowsedHistories = append(ipAsset.BrowsedHistories,
					metricresource.BrowsedHistory{
						BrowsedDomain: metricresource.BrowsedDomain{
							Domain: bucket.Key,
							Count:  bucket.DocCount,
						},
					})
			}
		}
	}

	domainNum := len(ipAsset.BrowsedHistories)
	if domainNum > 10 {
		domainNum = 10
	}

	resultCh, _ := errgroup.Batch(ipAsset.BrowsedHistories[:domainNum],
		func(browsedHistory interface{}) (interface{}, error) {
			history := browsedHistory.(metricresource.BrowsedHistory)
			resp, err := requestElasticsearch(TopBrowser, esclient.SrcIpKeyWord,
				esclient.ElasticsearchMustMatch{Match: map[string]string{
					esclient.DomainKeyWord: history.BrowsedDomain.Domain}},
				esTimeRange, 5, nodes)
			if err != nil {
				log.Infof("request top ip for domain %s failed: %s", history.BrowsedDomain.Domain, err.Error())
				return nil, nil
			}

			var topIps []metricresource.TopIpInfo
			if resp != nil {
				if tops, ok := resp.Aggregations[TopBrowser]; ok {
					for _, bucket := range tops.Buckets {
						if bucket.Key != ipstr {
							topIps = append(topIps, metricresource.TopIpInfo{
								Ip:    bucket.Key,
								Count: bucket.DocCount,
							})
						}
					}
				}
			}

			if len(topIps) != 0 {
				return &metricresource.BrowsedHistory{
					BrowsedDomain: history.BrowsedDomain,
					BrowserTopIps: topIps,
				}, nil
			} else {
				return nil, nil
			}
		})

	for result := range resultCh {
		if history, ok := result.(*metricresource.BrowsedHistory); ok {
			for i, browsedHistory := range ipAsset.BrowsedHistories {
				if browsedHistory.BrowsedDomain.Domain == history.BrowsedDomain.Domain {
					ipAsset.BrowsedHistories[i].BrowserTopIps = history.BrowserTopIps
					break
				}
			}
		}
	}

	return []*metricresource.AssetSearch{
		&metricresource.AssetSearch{
			AssetType: metricresource.AssetTypeIp,
			IpAsset:   ipAsset,
		}}, nil
}

func getClosestSubnet(subnets []*dhcpresource.Subnet, ip net.IP) (*dhcpresource.Subnet, bool) {
	var maxPrefixLen int
	var subnet *dhcpresource.Subnet
	for _, subnet_ := range subnets {
		if subnet_.Ipnet.Contains(ip) {
			if ones, _ := subnet_.Ipnet.Mask.Size(); ones > maxPrefixLen {
				subnet = subnet_
				maxPrefixLen = ones
			}
		}
	}

	return subnet, subnet != nil
}

func dhcpSubnetToMetricSubnetInfo(subnet *dhcpresource.Subnet) metricresource.SubnetInfo {
	return metricresource.SubnetInfo{
		Subnet:       subnet.Subnet,
		NetworkType:  subnet.NetworkType,
		SemanticName: subnet.Tags,
	}
}

func ipamNetworkInfoToMetricSubnetInfo(networkInfo *ipamresource.NetworkInfo) metricresource.SubnetInfo {
	return metricresource.SubnetInfo{
		Subnet:       networkInfo.Prefix,
		NetworkType:  networkInfo.NetworkType,
		SemanticName: networkInfo.SemanticName,
	}
}

func auditAssetToIpStateInfo(a *assetresource.AuditAsset) metricresource.IpStateInfo {
	return metricresource.IpStateInfo{
		IpType:  string(a.IpType),
		IpState: string(a.IpState),
	}
}

func deviceToAssetInfo(d *assetresource.Device) metricresource.DeviceInfo {
	return metricresource.DeviceInfo{
		Name:              d.Name,
		DeviceType:        string(d.DeviceType),
		Mac:               d.Mac,
		UplinkEquipment:   d.UplinkEquipment,
		UplinkPort:        d.UplinkPort,
		VlanId:            d.VlanId,
		ComputerRoom:      d.ComputerRoom,
		ComputerRack:      d.ComputerRack,
		DeployedService:   d.DeployedService,
		Department:        d.Department,
		ResponsiblePerson: d.ResponsiblePerson,
		Telephone:         d.Telephone,
	}
}

func getTopSize(filters []restresource.Filter) uint32 {
	topSize := uint32(10)
	if sizeStr, ok := util.GetFilterValueWithEqModifierFromFilters(FilterNameTopSize, filters); ok {
		if size, err := strconv.ParseUint(sizeStr, 10, 64); err == nil {
			topSize = uint32(size)
		}
	}

	return topSize
}

func getElasticsearchMustRange(filters []restresource.Filter) *esclient.ElasticsearchMustRange {
	if periodStr, ok := util.GetFilterValueWithEqModifierFromFilters(FilterNamePeriod, filters); ok {
		if period, err := strconv.ParseUint(periodStr, 10, 64); err == nil {
			to := time.Now()
			from := to.AddDate(0, 0, -int(period))
			return &esclient.ElasticsearchMustRange{
				Range: esclient.ElasticsearchRange{
					Timestamp: esclient.ElasticsearchRangeTimestamp{
						GTE:    from.Format(util.TimeFormatYMDHM),
						LTE:    to.Format(util.TimeFormatYMDHM),
						Format: esclient.TimestampFormat,
					},
				},
			}
		}
	}

	return nil
}

func requestElasticsearch(aggsKey, aggsTermField string, queryMatch esclient.ElasticsearchMustMatch, queryRange *esclient.ElasticsearchMustRange, topSize uint32, nodes []*metricresource.Node) (*esclient.ElasticsearchResponse, error) {
	dnsIndex := metricresource.GenESDNSIndex(nodes)
	if dnsIndex == "" {
		return nil, nil
	}

	queryBoolMust := []interface{}{queryMatch}
	if queryRange != nil {
		queryBoolMust = append(queryBoolMust, queryRange)
	}

	req := &esclient.ElasticsearchRequest{
		Size: 0,
		Query: esclient.ElasticsearchQuery{
			Bool: esclient.ElasticsearchBool{
				Must: queryBoolMust,
			},
		},
		Aggs: map[string]esclient.ElasticsearchAggs{
			aggsKey: esclient.ElasticsearchAggs{
				Term: esclient.ElasticsearchAggsTerm{
					Field: aggsTermField,
					Order: esclient.AggsTermOrder,
					Size:  topSize,
				},
			},
		},
	}

	return esclient.GetESClient().Request(req, dnsIndex)
}
