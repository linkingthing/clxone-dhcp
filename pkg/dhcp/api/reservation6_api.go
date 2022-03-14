package api

import (
	"fmt"

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
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	reservation := ctx.Resource.(*resource.Reservation6)
	if err := reservation.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create reservation params invalid: %s", err.Error()))
	}
	retreservation, err := r.Service.Create(subnet, reservation)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create reservation %s failed: %s",
				reservation.String(), err.Error()))
	}

	return retreservation, nil
}

func (r *Reservation6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	reservations, err := service.ListReservation6s(subnetID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list reservations with subnet %s failed: %s",
				subnetID, err.Error()))
	}
	return reservations, nil
}

func (r *Reservation6Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	reservationID := ctx.Resource.GetID()
	reservation, err := r.Service.Get(subnetID, reservationID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get reservation %s with subnetID %s failed: %s",
				reservationID, subnetID, err.Error()))
	}

	return reservation, nil
}

func (r *Reservation6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	reservation := ctx.Resource.(*resource.Reservation6)
	if err := r.Service.Delete(subnet, reservation); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete reservation %s with subnet %s failed: %s",
				reservation.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func (r *Reservation6Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation := ctx.Resource.(*resource.Reservation6)
	retreservation, err := r.Service.Update(reservation)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update reservation6 %s with subnet %s failed: %s",
				reservation.String(), ctx.Resource.GetParent().GetID(), err.Error()))
	}
	return retreservation, nil
}
