package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
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

func (p *PingerApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	pingers, err := p.Service.List()
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pingers, nil
}

func (p *PingerApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pinger, err := p.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pinger, nil
}

func (p *PingerApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pinger := ctx.Resource.(*resource.Pinger)
	if err := p.Service.Update(pinger); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return pinger, nil
}
