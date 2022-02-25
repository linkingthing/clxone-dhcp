package api

import (
	"fmt"

	gohelperip "github.com/cuityhj/gohelper/ip"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type SubnetLease6Api struct {
	Service *service.SubnetLease6Service
}

func NewSubnetLease6Api() *SubnetLease6Api {
	return &SubnetLease6Api{Service: service.NewSubnetLease6Service()}
}

func (l *SubnetLease6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ip, hasAddressFilter := util.GetFilterValueWithEqModifierFromFilters(
		util.FilterNameIp, ctx.GetFilters())
	if hasAddressFilter {
		if _, err := gohelperip.ParseIPv6(ip); err != nil {
			return nil, nil
		}
	}
	ret, err := l.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnetlease6 failed: %s", err.Error()))
	}
	return ret, nil
}

func (l *SubnetLease6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	subnetId := ctx.Resource.GetParent().GetID()
	leaseId := ctx.Resource.GetID()
	_, err := gohelperip.ParseIPv6(leaseId)
	if err != nil {
		return resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("subnet %s lease6 id %s is invalid: %s",
				subnetId, leaseId, err.Error()))
	}

	if err := l.Service.Delete(subnetId, leaseId); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete lease %s with subnet6 %s failed: %s",
				leaseId, subnetId, err.Error()))
	}

	return nil
}
