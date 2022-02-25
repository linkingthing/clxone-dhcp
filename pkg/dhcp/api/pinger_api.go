package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type PingerApi struct {
	Service *service.PingerService
}

func NewPingerApi() *PingerApi {
	return &PingerApi{Service: service.NewPingerService()}
}

func (d *PingerApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	pingers, err := d.Service.List()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp pinger from db failed: %s", err.Error()))
	}
	return pingers, nil
}

func (d *PingerApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pingerID := ctx.Resource.(*resource.Pinger).GetID()
	pinger, err := d.Service.Get(pingerID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pinger from db failed: %s", err.Error()))
	}

	return pinger.(*resource.Pinger), nil
}

func (d *PingerApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pinger := ctx.Resource.(*resource.Pinger)
	retPinger, err := d.Service.Update(pinger)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update dhcp pinger failed: %s", err.Error()))
	}
	return retPinger, nil
}
