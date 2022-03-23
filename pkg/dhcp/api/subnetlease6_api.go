package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type SubnetLease6Api struct {
	Service *service.SubnetLease6Service
}

func NewSubnetLease6Api() *SubnetLease6Api {
	return &SubnetLease6Api{Service: service.NewSubnetLease6Service()}
}

func (h *SubnetLease6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetLease6s, err := h.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return subnetLease6s, nil
}

func (h *SubnetLease6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := h.Service.Delete(ctx.Resource.GetParent().GetID(), ctx.Resource.GetID()); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}
