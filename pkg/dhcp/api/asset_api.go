package api

import (
	"github.com/linkingthing/clxone-utils/excel"
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

func (a *AssetApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	asset := ctx.Resource.(*resource.Asset)
	if err := a.Service.Create(asset); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return asset, nil
}

func (a *AssetApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	duids, err := a.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		service.OrderByCreateTime, resource.SqlColumnName, resource.SqlColumnHwAddress, resource.SqlColumnAssetType,
		resource.SqlColumnManufacturer, resource.SqlColumnModel, resource.SqlColumnAccessNetworkTime))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return duids, nil
}

func (a *AssetApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	asset, err := a.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return asset, nil
}

func (a *AssetApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := a.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (a *AssetApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	asset := ctx.Resource.(*resource.Asset)
	if err := a.Service.Update(asset); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return asset, nil
}

func (a *AssetApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case excel.ActionNameImport:
		return a.actionImportExcel(ctx)
	case excel.ActionNameExport:
		return a.actionExportExcel(ctx)
	case excel.ActionNameExportTemplate:
		return a.actionExportExcelTemplate(ctx)
	case resource.ActionNameBatchDelete:
		return a.actionBatchDelete(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameAsset, ctx.Resource.GetAction().Name))
	}
}

func (a *AssetApi) actionImportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*excel.ImportFile)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameAsset, errorno.ErrNameImport))
	}

	if resp, err := a.Service.ImportExcel(file); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return resp, nil
	}
}

func (a *AssetApi) actionExportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if file, err := a.Service.ExportExcel(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}

func (a *AssetApi) actionExportExcelTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if file, err := a.Service.ExportExcelTemplate(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}

func (a *AssetApi) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	asset, ok := ctx.Resource.GetAction().Input.(*resource.Assets)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameAsset, errorno.ErrNameImport))
	}

	if err := a.Service.BatchDelete(asset.Ids); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return nil, nil
	}
}
