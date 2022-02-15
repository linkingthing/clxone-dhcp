package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type AdmitApi struct {
	Service *service.AdmitService
}

func NewAdmitApi() *AdmitApi {
	return &AdmitApi{Service: service.NewAdmitService()}
}

func (d *AdmitApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	admit, err := d.Service.List()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list admit failed: %s", err.Error()))
	}
	return admit, nil
}

func (d *AdmitApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitId := ctx.Resource.(*resource.Admit).GetID()
	admit, err := d.Service.Get(admitId)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get admit failed: %s", err.Error()))
	}
	return admit.(*resource.Admit), nil
}

func (d *AdmitApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	adt := ctx.Resource.(*resource.Admit)
	admit, err := d.Service.Update(adt)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update dhcp admit failed: %s", err.Error()))
	}
	return admit, nil
}
