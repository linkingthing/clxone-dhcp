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

type DhcpFingerprintApi struct {
	Service *service.DhcpFingerprintService
}

func NewDhcpFingerprintApi() *DhcpFingerprintApi {
	return &DhcpFingerprintApi{Service: service.NewDhcpFingerprintService()}
}

func (f *DhcpFingerprintApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprint := ctx.Resource.(*resource.DhcpFingerprint)
	if err := f.Service.Create(fingerprint); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return fingerprint, nil
}

func (f *DhcpFingerprintApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	fingerprints, err := f.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		service.OrderByCreateTime, service.FingerprintFilterNames...))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return fingerprints, nil
}

func (f *DhcpFingerprintApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprint, err := f.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return fingerprint, nil
}

func (f *DhcpFingerprintApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	fingerprint := ctx.Resource.(*resource.DhcpFingerprint)
	if err := f.Service.Update(fingerprint); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return fingerprint, nil
}

func (f *DhcpFingerprintApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := f.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (f *DhcpFingerprintApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case excel.ActionNameImport:
		return f.actionImportExcel(ctx)
	case excel.ActionNameExport:
		return f.actionExportExcel()
	case excel.ActionNameExportTemplate:
		return f.actionExportExcelTemplate()
	case resource.ActionNameListClientTypes:
		return f.actionListClientTypes()
	case resource.ActionNameBatchDelete:
		return f.actionBatchDelete(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameFingerprint, ctx.Resource.GetAction().Name))
	}
}

func (f *DhcpFingerprintApi) actionImportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*excel.ImportFile)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameFingerprint, errorno.ErrNameImport))
	}

	if resp, err := f.Service.ImportExcel(file); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return resp, nil
	}
}

func (f *DhcpFingerprintApi) actionExportExcel() (interface{}, *resterror.APIError) {
	if exportFile, err := f.Service.ExportExcel(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return exportFile, nil
	}
}

func (f *DhcpFingerprintApi) actionExportExcelTemplate() (interface{}, *resterror.APIError) {
	if file, err := f.Service.ExportExcelTemplate(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}

func (f *DhcpFingerprintApi) actionListClientTypes() (interface{}, *resterror.APIError) {
	if clientTypes, err := f.Service.ListClientTypes(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return clientTypes, nil
	}
}

func (f *DhcpFingerprintApi) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	fingerprints, ok := ctx.Resource.GetAction().Input.(*resource.DhcpFingerprints)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameFingerprint, errorno.ErrNameBatchDelete))
	}

	if err := f.Service.BatchDelete(fingerprints.Ids); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return nil, nil
	}
}
