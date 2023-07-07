package service

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/linkingthing/cement/log"
	pg "github.com/linkingthing/clxone-utils/postgresql"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	dhcpresource "github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type SubnetUsedRatioService struct {
	prometheusAddr string
}

func NewSubnetUsedRatioService(config *config.DHCPConfig) *SubnetUsedRatioService {
	return &SubnetUsedRatioService{prometheusAddr: config.Prometheus.Addr}
}

func (h *SubnetUsedRatioService) List(ctx *restresource.Context) (interface{}, error) {
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPSubnetUsage,
		PromQuery:      PromQueryVersion,
	}

	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, err
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, err
	}

	subnets, err := getSubnetsFromDB(metricCtx.Version)
	if err != nil {
		return nil, err
	}

	nodeIpAndSubnetUsages := make(map[string]resource.SubnetUsages)
	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok {
			if subnet, ok := r.MetricLabels[string(MetricLabelSubnet)]; ok {
				if _, ok := subnets[subnet]; ok {
					subnets := nodeIpAndSubnetUsages[nodeIp]
					subnets = append(subnets, resource.SubnetUsage{
						Subnet:     subnet,
						UsedRatios: getRatiosWithTimestamp(r.Values, metricCtx.Period),
					})
					nodeIpAndSubnetUsages[nodeIp] = subnets
				}
			}
		}
	}

	nodeNames, err := service.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID()))
	if err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	}

	subnetUsedRatios := make([]*resource.SubnetUsedRatio, 0, len(nodeIpAndSubnetUsages))
	for nodeIp, subnets := range nodeIpAndSubnetUsages {
		sort.Sort(subnets)
		subnetUsedRatio := &resource.SubnetUsedRatio{Subnets: subnets, NodeName: nodeNames[nodeIp]}
		subnetUsedRatio.SetID(nodeIp)
		subnetUsedRatios = append(subnetUsedRatios, subnetUsedRatio)
	}

	return subnetUsedRatios, nil
}

func getSubnetsFromDB(version DHCPVersion) (map[string]struct{}, error) {
	subnets := make(map[string]struct{})
	if version == DHCPVersion4 {
		var subnet4s []*dhcpresource.Subnet4
		if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
			return tx.Fill(nil, &subnet4s)
		}); err != nil {
			return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV4), pg.Error(err).Error())
		}

		for _, subnet := range subnet4s {
			subnets[subnet.Subnet] = struct{}{}
		}
	} else {
		var subnet6s []*dhcpresource.Subnet6
		if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
			return tx.Fill(nil, &subnet6s)
		}); err != nil {
			return nil, errorno.ErrDBError(errorno.ErrDBNameQuery, string(errorno.ErrNameNetworkV6), pg.Error(err).Error())
		}

		for _, subnet := range subnet6s {
			subnets[subnet.Subnet] = struct{}{}
		}
	}

	return subnets, nil
}

func getRatiosWithTimestamp(values [][]interface{}, period *TimePeriod) []resource.RatioWithTimestamp {
	var ratioWithTimestamps []resource.RatioWithTimestamp
	for i := period.Begin; i <= period.End; i += period.Step {
		ratioWithTimestamps = append(ratioWithTimestamps, resource.RatioWithTimestamp{
			Timestamp: restresource.ISOTime(time.Unix(i, 0)),
			Ratio:     "0",
		})
	}

	for _, vs := range values {
		if t, s := getTimestampAndValue(vs); t != 0 {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				ratioWithTimestamps[(t-period.Begin)/period.Step].Ratio = fmt.Sprintf("%.4f", f)
			}
		}
	}

	return ratioWithTimestamps
}

func (h *SubnetUsedRatioService) Get(ctx *restresource.Context) (restresource.Resource, error) {
	subnetUsedRatio := ctx.Resource.(*resource.SubnetUsedRatio)
	metricCtx := &MetricContext{
		PrometheusAddr: h.prometheusAddr,
		MetricName:     MetricNameDHCPSubnetUsage,
		PromQuery:      PromQueryVersionNode,
		NodeIP:         subnetUsedRatio.GetID(),
	}

	if err := resetMetricContext(ctx, metricCtx); err != nil {
		return nil, err
	}

	resp, err := prometheusRequest(metricCtx)
	if err != nil {
		return nil, err
	}

	subnets, err := getSubnetsFromDB(metricCtx.Version)
	if err != nil {
		return nil, err
	}

	for _, r := range resp.Data.Results {
		if nodeIp, ok := r.MetricLabels[string(MetricLabelNode)]; ok && metricCtx.NodeIP == nodeIp {
			if subnet, ok := r.MetricLabels[string(MetricLabelSubnet)]; ok {
				if _, ok := subnets[subnet]; ok {
					subnetUsedRatio.Subnets = append(subnetUsedRatio.Subnets, resource.SubnetUsage{
						Subnet:     subnet,
						UsedRatios: getRatiosWithTimestamp(r.Values, metricCtx.Period),
					})
				}
			}
		}
	}

	if nodeNames, err := service.GetNodeNames(IsDHCPVersion4(ctx.Resource.GetParent().GetID())); err != nil {
		log.Warnf("list agent nodes failed: %s", err.Error())
	} else {
		subnetUsedRatio.NodeName = nodeNames[subnetUsedRatio.GetID()]
	}

	sort.Sort(resource.SubnetUsages(subnetUsedRatio.Subnets))
	return subnetUsedRatio, nil
}

func (h *SubnetUsedRatioService) Export(ctx *restresource.Context) (interface{}, error) {
	return exportMultiColunms(ctx, &MetricContext{
		NodeIP:         ctx.Resource.GetID(),
		PrometheusAddr: h.prometheusAddr,
		PromQuery:      PromQueryVersionNode,
		MetricName:     MetricNameDHCPSubnetUsage,
		MetricLabel:    MetricLabelSubnet,
	})
}
