package api

import (
	"fmt"

	resterror "github.com/zdnscloud/gorest/error"

	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)


var TableHeaderLease = []string{"日期", "租赁总数"}

func getLease(ctx *MetricContext) (*resource.Dhcp, *resterror.APIError) {
	ctx.MetricName = MetricNameDHCPLeasesTotal
	leaseValues, err := getValuesFromPrometheus(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get node %s leases failed: %s", ctx.NodeIP, err.Error()))
	}

	dhcp := &resource.Dhcp{Lease: resource.Lease{Values: leaseValues}}
	dhcp.SetID(resource.ResourceIDLease)
	return dhcp, nil
}

func exportLease(ctx *MetricContext) (interface{}, *resterror.APIError) {
	ctx.MetricName = MetricNameDHCPLeasesTotal
	ctx.TableHeader = TableHeaderLease
	return exportTwoColumns(ctx)
}
