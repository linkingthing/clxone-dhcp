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

type AdmitDuidApi struct {
	Service *service.AdmitDuidService
}

func NewAdmitDuidApi() *AdmitDuidApi {
	return &AdmitDuidApi{Service: service.NewAdmitDuidService()}
}

func (d *AdmitDuidApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuid := ctx.Resource.(*resource.AdmitDuid)
	if err := d.Service.Create(admitDuid); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitDuid, nil
}

func (d *AdmitDuidApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	duids, err := d.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		resource.SqlColumnDuid, resource.SqlColumnDuid))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return duids, nil
}

func (d *AdmitDuidApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuid, err := d.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitDuid, nil
}

func (d *AdmitDuidApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := d.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (d *AdmitDuidApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitDuid := ctx.Resource.(*resource.AdmitDuid)
	if err := d.Service.Update(admitDuid); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitDuid, nil
}

func (d *AdmitDuidApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
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

func (d *AdmitDuidApi) actionImportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
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

func (d *AdmitDuidApi) actionExportExcel() (interface{}, *resterror.APIError) {
	if exportFile, err := d.Service.ExportExcel(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return exportFile, nil
	}
}

func (d *AdmitDuidApi) actionExportExcelTemplate() (interface{}, *resterror.APIError) {
	if file, err := d.Service.ExportExcelTemplate(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}

func (d *AdmitDuidApi) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	duids, ok := ctx.Resource.GetAction().Input.(*resource.AdmitDuids)
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
