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

type AddressCodeLayoutSegmentApi struct {
	Service *service.AddressCodeLayoutSegmentService
}

func NewAddressCodeLayoutSegmentApi() *AddressCodeLayoutSegmentApi {
	return &AddressCodeLayoutSegmentApi{Service: service.NewAddressCodeLayoutSegmentService()}
}

func (s *AddressCodeLayoutSegmentApi) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode := ctx.Resource.(*resource.AddressCodeLayoutSegment)
	if err := s.Service.Create(ctx.Resource.GetParent().GetParent().GetID(),
		ctx.Resource.GetParent().GetID(), addressCode); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}

func (s *AddressCodeLayoutSegmentApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	duids, err := s.Service.List(ctx.Resource.GetParent().GetID(),
		util.GenStrConditionsFromFilters(ctx.GetFilters(),
			resource.SqlColumnCode, resource.SqlColumnCode, resource.SqlColumnValue))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return duids, nil
}

func (s *AddressCodeLayoutSegmentApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode, err := s.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}

func (s *AddressCodeLayoutSegmentApi) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := s.Service.Delete(ctx.Resource.GetParent().GetParent().GetID(),
		ctx.Resource.GetParent().GetID(), ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (s *AddressCodeLayoutSegmentApi) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	addressCode := ctx.Resource.(*resource.AddressCodeLayoutSegment)
	if err := s.Service.Update(ctx.Resource.GetParent().GetParent().GetID(),
		ctx.Resource.GetParent().GetID(), addressCode); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return addressCode, nil
}

func (s *AddressCodeLayoutSegmentApi) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case excel.ActionNameImport:
		return s.actionImportExcel(ctx)
	case excel.ActionNameExport:
		return s.actionExportExcel(ctx)
	case excel.ActionNameExportTemplate:
		return s.actionExportExcelTemplate(ctx)
	case resource.ActionNameBatchDelete:
		return s.actionBatchDelete(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameAddressCodeLayoutSegment, ctx.Resource.GetAction().Name))
	}
}

func (s *AddressCodeLayoutSegmentApi) actionImportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*excel.ImportFile)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameAddressCodeLayoutSegment, errorno.ErrNameImport))
	}

	if resp, err := s.Service.ImportExcel(ctx.Resource.GetParent().GetParent().GetID(),
		ctx.Resource.GetParent().GetID(), file); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return resp, nil
	}
}

func (s *AddressCodeLayoutSegmentApi) actionExportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if file, err := s.Service.ExportExcel(ctx.Resource.GetParent().GetID()); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}

func (s *AddressCodeLayoutSegmentApi) actionExportExcelTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if file, err := s.Service.ExportExcelTemplate(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}

func (s *AddressCodeLayoutSegmentApi) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	segments, ok := ctx.Resource.GetAction().Input.(*resource.AddressCodeLayoutSegments)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameAddressCodeLayoutSegment, errorno.ErrNameBatchDelete))
	}

	if err := s.Service.BatchDelete(ctx.Resource.GetParent().GetParent().GetID(),
		ctx.Resource.GetParent().GetID(), segments.Codes); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return nil, nil
	}
}
