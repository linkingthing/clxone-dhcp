package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type SubnetLease6Api struct {
	Service *service.SubnetLease6Service
}

func NewSubnetLease6Api() *SubnetLease6Api {
	return &SubnetLease6Api{Service: service.NewSubnetLease6Service()}
}

func (h *SubnetLease6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ip, _ := util.GetFilterValueWithEqModifierFromFilters(
		util.FilterNameIp, ctx.GetFilters())

	subnetLease6s, err := h.Service.List(ctx.Resource.GetParent().(*resource.Subnet6), ip)
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

func (r *SubnetLease6Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	return nil, nil
}

func (s *SubnetLease6Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionBatchDelete:
		return s.actionBatchDelete(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (s *SubnetLease6Api) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	input, ok := ctx.Resource.GetAction().Input.(*resource.BatchDeleteLeasesInput)
	if !ok {
		return nil, resterror.NewAPIError(resterror.ServerError, "action batch delete input invalid")
	}

	if err := s.Service.BatchDeleteLease6s(ctx.Resource.GetParent().GetID(), input.Addresses); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	} else {
		return nil, nil
	}
}
