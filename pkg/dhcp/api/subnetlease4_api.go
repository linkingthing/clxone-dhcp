package api

import (
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

func (h *SubnetLease4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetLease4s, err := h.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return subnetLease4s, nil
}

func (h *SubnetLease4Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := h.Service.Delete(ctx.Resource.GetParent().GetID(), ctx.Resource.GetID()); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}
