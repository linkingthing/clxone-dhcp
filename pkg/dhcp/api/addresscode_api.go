package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type AddressCodeApi struct {
	Service *service.AddressCodeService
}

func NewAddressCodeApi() *AddressCodeApi {
	return &AddressCodeApi{Service: service.NewAddressCodeService()}
}

func (d *AddressCodeApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode := ctx.Resource.(*resource.AddressCode)
	if err := d.Service.Create(addressCode); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}

func (d *AddressCodeApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	duids, err := d.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		service.OrderByCreateTime, resource.SqlColumnDuid,
		resource.SqlColumnHwAddress, resource.SqlColumnCode))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return duids, nil
}

func (d *AddressCodeApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}

func (d *AddressCodeApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := d.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (d *AddressCodeApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode := ctx.Resource.(*resource.AddressCode)
	if err := d.Service.Update(addressCode); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}
