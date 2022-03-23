package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type PingerApi struct {
	Service *service.PingerService
}

func NewPingerApi() (*PingerApi, error) {
	s, err := service.NewPingerService()
	if err != nil {
		return nil, err
	}

	return &PingerApi{Service: s}, nil
}

func (d *PingerApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	pingers, err := d.Service.List()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pingers, nil
}

func (d *PingerApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pinger, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pinger, nil
}

func (d *PingerApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pinger := ctx.Resource.(*resource.Pinger)
	if err := d.Service.Update(pinger); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return pinger, nil
}
