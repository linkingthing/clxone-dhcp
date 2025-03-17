package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type SubnetLease6Api struct {
	Service *service.SubnetLease6Service
}

func NewSubnetLease6Api() *SubnetLease6Api {
	return &SubnetLease6Api{Service: service.NewSubnetLease6Service()}
}

func (l *SubnetLease6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ip, _ := util.GetFilterValueWithEqModifierFromFilters(
		util.FilterNameIp, ctx.GetFilters())

	subnetLease6s, err := l.Service.List(ctx.Resource.GetParent().(*resource.Subnet6), ip)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return subnetLease6s, nil
}

func (l *SubnetLease6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := l.Service.Delete(ctx.Resource.GetParent().GetID(), ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (l *SubnetLease6Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	return nil, nil
}

func (l *SubnetLease6Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionBatchDelete:
		return l.actionBatchDelete(ctx)
	case resource.ActionListToReservation:
		return l.actionListToReservation(ctx)
	case resource.ActionDynamicToReservation:
		return l.actionDynamicToReservation(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameLease, errorno.ErrName(ctx.Resource.GetAction().Name)))
	}
}

func (l *SubnetLease6Api) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	input, ok := ctx.Resource.GetAction().Input.(*resource.BatchDeleteLeasesInput)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameLease, resource.ActionBatchDelete))
	}

	if err := l.Service.BatchDeleteLease6s(ctx.Resource.GetParent().GetID(), input.Addresses); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return nil, nil
	}
}

func (l *SubnetLease6Api) actionListToReservation(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	util.SetIgnoreAuditLog(ctx)
	input, ok := ctx.Resource.GetAction().Input.(*resource.ConvToReservationInput)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameLease, resource.ActionListToReservation))
	}

	output, err := l.Service.ActionListToReservation(ctx.Resource.GetParent().(*resource.Subnet6), input)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return output, nil
}

func (l *SubnetLease6Api) actionDynamicToReservation(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	input, ok := ctx.Resource.GetAction().Input.(*resource.ConvToReservationInput)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameLease, resource.ActionDynamicToReservation))
	}

	if err := l.Service.ActionDynamicToReservation(ctx.Resource.GetParent().(*resource.Subnet6), input); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return nil, nil
}
