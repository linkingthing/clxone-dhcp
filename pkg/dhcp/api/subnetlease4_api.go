package api

import (
	"fmt"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type SubnetLease4Api struct {
	Service *service.SubnetLease4Service
}

func NewSubnetLease4Api() *SubnetLease4Api {
	return &SubnetLease4Api{Service: service.NewSubnetLease4Service()}
}

func (h *SubnetLease4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ip, _ := util.GetFilterValueWithEqModifierFromFilters(
		util.FilterNameIp, ctx.GetFilters())

	subnetLease4s, err := h.Service.List(ctx.Resource.GetParent().(*resource.Subnet4), ip)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return subnetLease4s, nil
}

func (h *SubnetLease4Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := h.Service.BatchDeleteLease4s(
		(ctx.Resource.GetParent().(*resource.Subnet4)).GetID(),
		[]string{ctx.Resource.GetID()}); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}

func (s *SubnetLease4Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionBatchDelete:
		return s.actionBatchDelete(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (s *SubnetLease4Api) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	input, ok := ctx.Resource.GetAction().Input.(*resource.BatchDeleteInput)
	if !ok {
		return nil, resterror.NewAPIError(resterror.ServerError, "action batch delete input invalid")
	}

	if err := s.Service.BatchDeleteLease4s(input.Subnet, input.Ids); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	} else {
		return nil, nil
	}
}
