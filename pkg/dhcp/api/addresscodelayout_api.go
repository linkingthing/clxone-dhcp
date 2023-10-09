package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type AddressCodeLayoutApi struct {
	Service *service.AddressCodeLayoutService
}

func NewAddressCodeLayoutApi() *AddressCodeLayoutApi {
	return &AddressCodeLayoutApi{Service: service.NewAddressCodeLayoutService()}
}

func (d *AddressCodeLayoutApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode := ctx.Resource.(*resource.AddressCodeLayout)
	if err := d.Service.Create(ctx.Resource.GetParent().GetID(), addressCode); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}

func (d *AddressCodeLayoutApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	duids, err := d.Service.List(ctx.Resource.GetParent().GetID(),
		util.GenStrConditionsFromFilters(ctx.GetFilters(),
			resource.SqlColumnBeginBit, resource.SqlColumnLabel))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return duids, nil
}

func (d *AddressCodeLayoutApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}

func (d *AddressCodeLayoutApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := d.Service.Delete(ctx.Resource.GetParent().GetID(), ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (d *AddressCodeLayoutApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode := ctx.Resource.(*resource.AddressCodeLayout)
	if err := d.Service.Update(ctx.Resource.GetParent().GetID(), addressCode); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}
