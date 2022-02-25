package api

import (
	"fmt"

	gohelperip "github.com/cuityhj/gohelper/ip"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type SubnetLease4Api struct {
	Service *service.SubnetLease4Service
}

func NewSubnetLease4Api() *SubnetLease4Api {
	return &SubnetLease4Api{Service: service.NewSubnetLease4Service()}
}

func (l *SubnetLease4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ret, err := l.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnetlease4 failed: %s", err.Error()))
	}
	return ret, nil
}

func (l *SubnetLease4Api) Delete(ctx *restresource.Context) *resterror.APIError {
	subnetId := ctx.Resource.GetParent().GetID()
	leaseId := ctx.Resource.GetID()
	_, err := gohelperip.ParseIPv4(leaseId)
	if err != nil {
		return resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("subnet %s lease4 id %s is invalid: %v",
				subnetId, leaseId, err.Error()))
	}

	if err := l.Service.Delete(subnetId, leaseId); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete lease %s with subnet4 %s failed: %s",
				leaseId, subnetId, err.Error()))
	}

	return nil
}
