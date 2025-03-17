package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type AdmitApi struct {
	Service *service.AdmitService
}

func NewAdmitApi() (*AdmitApi, error) {
	s, err := service.NewAdmitService()
	if err != nil {
		return nil, err
	}

	return &AdmitApi{Service: s}, nil
}

func (a *AdmitApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	admits, err := a.Service.List()
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admits, nil
}

func (a *AdmitApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admit, err := a.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admit, nil
}

func (a *AdmitApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admit := ctx.Resource.(*resource.Admit)
	if err := a.Service.Update(admit); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admit, nil
}
