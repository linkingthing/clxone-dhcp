package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type Reservation4Api struct {
	Service *service.Reservation4Service
}

func NewReservation4Api() *Reservation4Api {
	return &Reservation4Api{Service: service.NewReservation4Service()}
}

func (r *Reservation4Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation := ctx.Resource.(*resource.Reservation4)
	if err := r.Service.Create(ctx.Resource.GetParent().(*resource.Subnet4), reservation); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return reservation, nil
}

func (r *Reservation4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	reservations, err := r.Service.List(ctx.Resource.GetParent().(*resource.Subnet4))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return reservations, nil
}

func (r *Reservation4Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation, err := r.Service.Get(ctx.Resource.GetParent().(*resource.Subnet4), ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return reservation, nil
}

func (r *Reservation4Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := r.Service.Delete(
		ctx.Resource.GetParent().(*resource.Subnet4),
		ctx.Resource.(*resource.Reservation4)); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (r *Reservation4Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation := ctx.Resource.(*resource.Reservation4)
	if err := r.Service.Update(ctx.Resource.GetParent().GetID(), reservation); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return reservation, nil
}
