package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type AssetApi struct {
	Service *service.AssetService
}

func NewAssetApi() *AssetApi {
	return &AssetApi{Service: service.NewAssetService()}
}

func (d *AssetApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	asset := ctx.Resource.(*resource.Asset)
	if err := d.Service.Create(asset); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return asset, nil
}

func (d *AssetApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	duids, err := d.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		service.OrderByCreateTime, resource.SqlColumnHwAddress, resource.SqlColumnAssetType,
		resource.SqlColumnManufacturer, resource.SqlColumnModel, resource.SqlColumnAccessNetworkTime))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return duids, nil
}

func (d *AssetApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	asset, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return asset, nil
}

func (d *AssetApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := d.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (d *AssetApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	asset := ctx.Resource.(*resource.Asset)
	if err := d.Service.Update(asset); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return asset, nil
}
