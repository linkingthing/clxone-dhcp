package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type Reservation6Api struct {
	Service *service.Reservation6Service
}

func NewReservation6Api() *Reservation6Api {
	return &Reservation6Api{Service: service.NewReservation6Service()}
}

func (r *Reservation6Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation := ctx.Resource.(*resource.Reservation6)
	if err := r.Service.Create(ctx.Resource.GetParent().(*resource.Subnet6), reservation); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return reservation, nil
}

func (r *Reservation6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	reservations, err := r.Service.List(ctx.Resource.GetParent().GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return reservations, nil
}

func (r *Reservation6Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation, err := r.Service.Get(ctx.Resource.GetParent().(*resource.Subnet6), ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return reservation, nil
}

func (r *Reservation6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := r.Service.Delete(
		ctx.Resource.GetParent().(*resource.Subnet6),
		ctx.Resource.(*resource.Reservation6)); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}

func (r *Reservation6Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation := ctx.Resource.(*resource.Reservation6)
	if err := r.Service.Update(ctx.Resource.GetParent().GetID(), reservation); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return reservation, nil
}
