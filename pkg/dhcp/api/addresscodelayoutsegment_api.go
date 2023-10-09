package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type AddressCodeLayoutSegmentApi struct {
	Service *service.AddressCodeLayoutSegmentService
}

func NewAddressCodeLayoutSegmentApi() *AddressCodeLayoutSegmentApi {
	return &AddressCodeLayoutSegmentApi{Service: service.NewAddressCodeLayoutSegmentService()}
}

func (d *AddressCodeLayoutSegmentApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode := ctx.Resource.(*resource.AddressCodeLayoutSegment)
	if err := d.Service.Create(ctx.Resource.GetParent().GetParent().GetID(),
		ctx.Resource.GetParent().GetID(), addressCode); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}

func (d *AddressCodeLayoutSegmentApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	duids, err := d.Service.List(ctx.Resource.GetParent().GetID(),
		util.GenStrConditionsFromFilters(ctx.GetFilters(),
			resource.SqlColumnCode, resource.SqlColumnCode, resource.SqlColumnValue))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return duids, nil
}

func (d *AddressCodeLayoutSegmentApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}

func (d *AddressCodeLayoutSegmentApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := d.Service.Delete(ctx.Resource.GetParent().GetParent().GetID(),
		ctx.Resource.GetParent().GetID(), ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (d *AddressCodeLayoutSegmentApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode := ctx.Resource.(*resource.AddressCodeLayoutSegment)
	if err := d.Service.Update(ctx.Resource.GetParent().GetParent().GetID(),
		ctx.Resource.GetParent().GetID(), addressCode); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}
