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

type AdmitMacApi struct {
	Service *service.AdmitMacService
}

func NewAdmitMacApi() *AdmitMacApi {
	return &AdmitMacApi{Service: service.NewAdmitMacService()}
}

func (m *AdmitMacApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMac := ctx.Resource.(*resource.AdmitMac)
	if err := m.Service.Create(admitMac); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitMac, nil
}

func (m *AdmitMacApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	macs, err := m.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		resource.SqlColumnHwAddress, resource.SqlColumnHwAddress))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return macs, nil
}

func (m *AdmitMacApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMac, err := m.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitMac, nil
}

func (m *AdmitMacApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := m.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (m *AdmitMacApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitMac := ctx.Resource.(*resource.AdmitMac)
	if err := m.Service.Update(admitMac); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitMac, nil
}

func (m *AdmitMacApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case excel.ActionNameImport:
		return m.actionImportExcel(ctx)
	case excel.ActionNameExport:
		return m.actionExportExcel()
	case excel.ActionNameExportTemplate:
		return m.actionExportExcelTemplate()
	case resource.ActionNameBatchDelete:
		return m.actionBatchDelete(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameMac, ctx.Resource.GetAction().Name))
	}
}

func (m *AdmitMacApi) actionImportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*excel.ImportFile)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameMac, errorno.ErrNameImport))
	}

	if resp, err := m.Service.ImportExcel(file); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return resp, nil
	}
}

func (m *AdmitMacApi) actionExportExcel() (interface{}, *resterror.APIError) {
	if exportFile, err := m.Service.ExportExcel(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return exportFile, nil
	}
}

func (m *AdmitMacApi) actionExportExcelTemplate() (interface{}, *resterror.APIError) {
	if file, err := m.Service.ExportExcelTemplate(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}

func (m *AdmitMacApi) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	macs, ok := ctx.Resource.GetAction().Input.(*resource.AdmitMacs)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameMac, errorno.ErrNameBatchDelete))
	}

	if err := m.Service.BatchDelete(macs.Ids); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return nil, nil
	}
}
