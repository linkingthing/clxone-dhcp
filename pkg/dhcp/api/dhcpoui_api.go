package api

import (
	"github.com/linkingthing/clxone-utils/excel"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type DhcpOuiApi struct {
	Service *service.DhcpOuiService
}

func NewDhcpOuiApi() *DhcpOuiApi {
	return &DhcpOuiApi{Service: service.NewDhcpOuiService()}
}

func (o *DhcpOuiApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpOui := ctx.Resource.(*resource.DhcpOui)
	if err := o.Service.Create(dhcpOui); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return dhcpOui, nil
}

func (o *DhcpOuiApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ouis, err := o.Service.List(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return ouis, nil
}

func (o *DhcpOuiApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpOui, err := o.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return dhcpOui, nil
}

func (o *DhcpOuiApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	dhcpOui := ctx.Resource.(*resource.DhcpOui)
	if err := o.Service.Update(dhcpOui); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return dhcpOui, nil
}

func (o *DhcpOuiApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := o.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (o *DhcpOuiApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case excel.ActionNameImport:
		return o.actionImportExcel(ctx)
	case excel.ActionNameExport:
		return o.actionExportExcel()
	case excel.ActionNameExportTemplate:
		return o.actionExportExcelTemplate()
	case resource.ActionNameBatchDelete:
		return o.actionBatchDelete(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameOui, ctx.Resource.GetAction().Name))
	}
}

func (o *DhcpOuiApi) actionImportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*excel.ImportFile)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameOui, errorno.ErrNameImport))
	}

	if resp, err := o.Service.ImportExcel(file); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return resp, nil
	}
}

func (o *DhcpOuiApi) actionExportExcel() (interface{}, *resterror.APIError) {
	if exportFile, err := o.Service.ExportExcel(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return exportFile, nil
	}
}

func (o *DhcpOuiApi) actionExportExcelTemplate() (interface{}, *resterror.APIError) {
	if file, err := o.Service.ExportExcelTemplate(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}

func (o *DhcpOuiApi) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ouis, ok := ctx.Resource.GetAction().Input.(*resource.DhcpOuis)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameOui, errorno.ErrNameBatchDelete))
	}

	if err := o.Service.BatchDelete(ouis.Ids); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return nil, nil
	}
}
