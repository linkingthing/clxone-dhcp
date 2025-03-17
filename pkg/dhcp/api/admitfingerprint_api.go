package api

import (
	"github.com/linkingthing/clxone-utils/excel"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type AdmitFingerprintApi struct {
	Service *service.AdmitFingerprintService
}

func NewAdmitFingerprintApi() *AdmitFingerprintApi {
	return &AdmitFingerprintApi{Service: service.NewAdmitFingerprintService()}
}

func (f *AdmitFingerprintApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitFingerprint := ctx.Resource.(*resource.AdmitFingerprint)
	if err := f.Service.Create(admitFingerprint); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitFingerprint, nil
}

func (f *AdmitFingerprintApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	fingerprints, err := f.Service.List()
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return fingerprints, nil
}

func (f *AdmitFingerprintApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitFingerprint, err := f.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitFingerprint, nil
}

func (f *AdmitFingerprintApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := f.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (f *AdmitFingerprintApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	admitFingerprint := ctx.Resource.(*resource.AdmitFingerprint)
	if err := f.Service.Update(admitFingerprint); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return admitFingerprint, nil
}

func (f *AdmitFingerprintApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case excel.ActionNameImport:
		return f.actionImportExcel(ctx)
	case excel.ActionNameExport:
		return f.actionExportExcel()
	case excel.ActionNameExportTemplate:
		return f.actionExportExcelTemplate()
	case resource.ActionNameBatchDelete:
		return f.actionBatchDelete(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameFingerprint, ctx.Resource.GetAction().Name))
	}
}

func (f *AdmitFingerprintApi) actionImportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
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

func (f *AdmitFingerprintApi) actionExportExcel() (interface{}, *resterror.APIError) {
	if exportFile, err := f.Service.ExportExcel(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return exportFile, nil
	}
}

func (f *AdmitFingerprintApi) actionExportExcelTemplate() (interface{}, *resterror.APIError) {
	if file, err := f.Service.ExportExcelTemplate(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}

func (f *AdmitFingerprintApi) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	fingerprints, ok := ctx.Resource.GetAction().Input.(*resource.AdmitFingerprints)
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
