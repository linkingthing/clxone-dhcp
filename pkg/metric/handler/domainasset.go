package handler

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/zdnscloud/cement/errgroup"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/g53"
	restdb "github.com/zdnscloud/gorest/db"
	resterr "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	assetresource "github.com/linkingthing/clxone-dhcp/pkg/asset/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	dhcpresource "github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dnsresource "github.com/linkingthing/clxone-dhcp/pkg/dns/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/esclient"
	ipamresource "github.com/linkingthing/clxone-dhcp/pkg/ipam/resource"
	metricresource "github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

const (
	TopBrowser = "topBrowser"
	RootName   = "@"

	GetAuditAssetsWithIpsSql = "select * from gr_audit_asset where id in (select id from (select distinct on(ip) ip, id from gr_audit_asset where ip in (%s) order by ip, update_time desc) id)"
	GetAssetsWithMacsSql     = "select * from gr_device where mac in (%s)"
	GetSubnetsWithIdSql      = "select * from gr_subnet where id in (%s)"
)

func getDomainAssetInfo(ctx *restresource.Context, domain string) (interface{}, *resterr.APIError) {
	domainName, err := g53.NameFromString(domain)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.InvalidFormat,
			fmt.Sprintf("invalid domain %s: %s", domain, err.Error()))
	}

	domain = domainName.String(true)
	rrName, zoneName, err := splitRRAndZoneFromDomainName(domainName, domain)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.InvalidFormat, err.Error())
	}

	var nodes []*metricresource.Node
	var rrs []*dnsresource.AuthRr
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.FillEx(&rrs,
			"select * from gr_auth_rr where name like $1 and zone like $2 or name = '@' and zone = $3",
			rrName+"%", "%"+zoneName, domain); err != nil {
			return err
		}

		return tx.Fill(nil, &nodes)
	}); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError,
			fmt.Sprintf("list domain portraits from db failed: %s", err.Error()))
	}

	var authrrs metricresource.AuthRRs
	for _, rr := range rrs {
		if rr.Name+"."+rr.Zone == domain ||
			(rr.Name == RootName && (rr.Zone == domain || rr.Zone == RootName)) {
			authrrs = append(authrrs, metricresource.AuthRR{
				View:   rr.View,
				Zone:   rr.Zone,
				RRName: rr.Name,
				RRType: string(rr.RrType),
				TTL:    rr.Ttl,
				Rdata:  rr.Rdata,
			})
		}
	}
	sort.Sort(authrrs)
	assetSearch := &metricresource.AssetSearch{
		AssetType: metricresource.AssetTypeDomain,
		DomainAsset: metricresource.DomainAssetInfo{
			AuthRRs: authrrs,
		},
	}

	esTimeRange := getElasticsearchMustRange(ctx.GetFilters())
	resp, err := requestElasticsearch(TopBrowser, esclient.SrcIpKeyWord,
		esclient.ElasticsearchMustMatch{Match: map[string]string{esclient.DomainKeyWord: domain}},
		esTimeRange, getTopSize(ctx.GetFilters()), nodes)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError,
			fmt.Sprintf("request elasticsearch failed: %s", err.Error()))
	}

	if resp == nil {
		return []*metricresource.AssetSearch{assetSearch}, nil
	}

	var ips []string
	var browserIps metricresource.BrowserIps
	if tops, ok := resp.Aggregations[TopBrowser]; ok {
		for _, bucket := range tops.Buckets {
			ips = append(ips, bucket.Key)
			browserIps = append(browserIps, metricresource.BrowserIp{
				TopIp: metricresource.TopIpInfo{
					Ip:    bucket.Key,
					Count: bucket.DocCount,
				},
			})
		}
	}

	if len(ips) == 0 {
		return []*metricresource.AssetSearch{assetSearch}, nil
	}

	ipNum := len(ips)
	if ipNum > 10 {
		ipNum = 10
	}

	resultCh, _ := errgroup.Batch(ips[:ipNum], func(ip interface{}) (interface{}, error) {
		srcIp := ip.(string)
		resp, err := requestElasticsearch(TopBrowedHistory, esclient.DomainKeyWord,
			esclient.ElasticsearchMustMatch{Match: map[string]string{esclient.SrcIpKeyWord: srcIp}},
			esTimeRange, 5, nodes)
		if err != nil {
			log.Infof("request top domain for src ip %s failed: %s", srcIp, err.Error())
			return nil, nil
		}

		var browsedDomains []metricresource.BrowsedDomain
		if resp != nil {
			if tops, ok := resp.Aggregations[TopBrowedHistory]; ok {
				for _, bucket := range tops.Buckets {
					if bucket.Key != domain {
						browsedDomains = append(browsedDomains,
							metricresource.BrowsedDomain{
								Domain: bucket.Key,
								Count:  bucket.DocCount,
							})
					}
				}
			}
		}

		if len(browsedDomains) != 0 {
			return &metricresource.BrowserIp{
				TopIp:          metricresource.TopIpInfo{Ip: srcIp},
				BrowsedDomains: browsedDomains}, nil
		} else {
			return nil, nil
		}
	})

	for result := range resultCh {
		if browserIp, ok := result.(*metricresource.BrowserIp); ok {
			for i, browserIp_ := range browserIps {
				if browserIp_.TopIp.Ip == browserIp.TopIp.Ip {
					browserIps[i].BrowsedDomains = browserIp.BrowsedDomains
					break
				}
			}
		}
	}

	var auditAssets []*assetresource.AuditAsset
	var devices []*assetresource.Device
	var subnets []*dhcpresource.Subnet
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		ipIndex, ipArgs := genSqlArgsAndIndex(ips)
		if err := tx.FillEx(&auditAssets, fmt.Sprintf(GetAuditAssetsWithIpsSql, ipIndex),
			ipArgs...); err != nil {
			return err
		}

		if len(auditAssets) == len(ips) {
			var macs []string
			var subnetIds []string
			for _, auditAsset := range auditAssets {
				macs = append(macs, auditAsset.Mac)
				if slice.SliceIndex(subnetIds, auditAsset.Subnet) == -1 {
					subnetIds = append(subnetIds, auditAsset.Subnet)
				}
			}

			subnetIndex, subnetAgrs := genSqlArgsAndIndex(subnetIds)
			if err := tx.FillEx(&subnets, fmt.Sprintf(GetSubnetsWithIdSql, subnetIndex),
				subnetAgrs...); err != nil {
				return err
			}

			macIndex, macArgs := genSqlArgsAndIndex(macs)
			return tx.FillEx(&devices, fmt.Sprintf(GetAssetsWithMacsSql, macIndex), macArgs...)
		} else {
			if err := tx.Fill(nil, &subnets); err != nil {
				return err
			}

			return tx.Fill(nil, &devices)
		}
	}); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError,
			fmt.Sprintf("list domain portraits from db failed: %s", err.Error()))
	}

	networkInfos, err := ipamresource.FindNetworkInfos(ips...)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError,
			fmt.Sprintf("find network infos with ips %v failed: %s", ips, err.Error()))
	}

	for i, browserIp := range browserIps {
		if networkInfo, ok := networkInfos[browserIp.TopIp.Ip]; ok {
			browserIps[i].Subnet = ipamNetworkInfoToMetricSubnetInfo(networkInfo)
		}

		ip := net.ParseIP(browserIp.TopIp.Ip)
		if browserIps[i].Subnet.Subnet == "" {
			if subnet, ok := getClosestSubnet(subnets, ip); ok {
				browserIps[i].Subnet = dhcpSubnetToMetricSubnetInfo(subnet)
			}
		}

		for _, auditAsset := range auditAssets {
			if browserIp.TopIp.Ip == auditAsset.Ip {
				browserIps[i].IpState = auditAssetToIpStateInfo(auditAsset)
				browserIps[i].Subnet = metricresource.SubnetInfo{
					Subnet: auditAsset.Ipnet,
				}
			}
		}

		for _, device := range devices {
			if device.ContainsIp(browserIp.TopIp.Ip, ip.To4() != nil) {
				browserIps[i].Device = deviceToAssetInfo(device)
			}
		}
	}

	sort.Sort(browserIps)
	assetSearch.DomainAsset.Ips = browserIps
	return []*metricresource.AssetSearch{assetSearch}, nil
}

func splitRRAndZoneFromDomainName(domainName *g53.Name, domain string) (string, string, error) {
	var rrName, zoneName string
	switch domainName.LabelCount() {
	case 1:
		rrName = RootName
		zoneName = RootName
	case 2:
		rrName = RootName
		zoneName = domain
	default:
		if domainPrefix, err := domainName.Split(0, 1); err != nil {
			return "", "", fmt.Errorf("split domain %s failed: %s", domain, err.Error())
		} else {
			rrName = domainPrefix.String(true)
		}

		if domainSuffix, err := domainName.Parent(domainName.LabelCount() - 2); err != nil {
			return "", "", fmt.Errorf("get domain %s parent failed: %s", domain, err.Error())
		} else {
			zoneName = domainSuffix.String(true)
		}
	}

	return rrName, zoneName, nil
}

func genSqlArgsAndIndex(args []string) (string, []interface{}) {
	var indexes []string
	var sqlAgrs []interface{}
	for i, arg := range args {
		indexes = append(indexes, "$"+strconv.Itoa(i+1))
		sqlAgrs = append(sqlAgrs, arg)
	}

	return strings.Join(indexes, ","), sqlAgrs
}
