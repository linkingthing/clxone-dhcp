package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-utils/excel"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type RateLimitDuidApi struct {
	Service *service.RateLimitDuidService
}

func NewRateLimitDuidApi() *RateLimitDuidApi {
	return &RateLimitDuidApi{Service: service.NewRateLimitDuidService()}
}

func (d *RateLimitDuidApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rateLimitDuid := ctx.Resource.(*resource.RateLimitDuid)
	if err := d.Service.Create(rateLimitDuid); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimitDuid, nil
}

func (d *RateLimitDuidApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	rateLimitDuids, err := d.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		resource.SqlColumnDuid, resource.SqlColumnDuid))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimitDuids, nil
}

func (d *RateLimitDuidApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rateLimitDuid, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimitDuid, nil
}

func (d *RateLimitDuidApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := d.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (d *RateLimitDuidApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	rateLimitDuid := ctx.Resource.(*resource.RateLimitDuid)
	if err := d.Service.Update(rateLimitDuid); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return rateLimitDuid, nil
}

func (d *RateLimitDuidApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case excel.ActionNameImport:
		return d.actionImportExcel(ctx)
	case excel.ActionNameExport:
		return d.actionExportExcel()
	case excel.ActionNameExportTemplate:
		return d.actionExportExcelTemplate()
	case resource.ActionNameBatchDelete:
		return d.actionBatchDelete(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameDuid, ctx.Resource.GetAction().Name))
	}
}

func (d *RateLimitDuidApi) actionImportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*excel.ImportFile)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameDuid, errorno.ErrNameImport))
	}

	if resp, err := d.Service.ImportExcel(file); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return resp, nil
	}
}

func (d *RateLimitDuidApi) actionExportExcel() (interface{}, *resterror.APIError) {
	if exportFile, err := d.Service.ExportExcel(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return exportFile, nil
	}
}

func (d *RateLimitDuidApi) actionExportExcelTemplate() (interface{}, *resterror.APIError) {
	if file, err := d.Service.ExportExcelTemplate(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}

func (d *RateLimitDuidApi) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	duids, ok := ctx.Resource.GetAction().Input.(*resource.RateLimitDuids)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameDuid, errorno.ErrNameBatchDelete))
	}

	if err := d.Service.BatchDelete(duids.Ids); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return nil, nil
	}
}
