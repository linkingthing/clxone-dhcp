package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-utils/excel"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type Reservation6Api struct {
	Service *service.Reservation6Service
}

func NewReservation6Api() *Reservation6Api {
	return &Reservation6Api{Service: service.NewReservation6Service()}
}

func (r *Reservation6Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation := ctx.Resource.(*resource.Reservation6)
	if err := r.Service.Create(ctx.Resource.GetParent().(*resource.Subnet6), reservation); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return reservation, nil
}

func (r *Reservation6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	reservations, err := r.Service.List(ctx.Resource.GetParent().(*resource.Subnet6))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return reservations, nil
}

func (r *Reservation6Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation, err := r.Service.Get(ctx.Resource.GetParent().(*resource.Subnet6), ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return reservation, nil
}

func (r *Reservation6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := r.Service.Delete(
		ctx.Resource.GetParent().(*resource.Subnet6),
		ctx.Resource.(*resource.Reservation6)); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (r *Reservation6Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation := ctx.Resource.(*resource.Reservation6)
	if err := r.Service.Update(ctx.Resource.GetParent().GetID(), reservation); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return reservation, nil
}

func (s *Reservation6Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionBatchDelete:
		return s.actionBatchDelete(ctx)
	case excel.ActionNameImport:
		return s.actionImportExcel(ctx)
	case excel.ActionNameExport:
		return s.actionExportExcel(ctx)
	case excel.ActionNameExportTemplate:
		return s.actionExportExcelTemplate()
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameDhcpReservation, errorno.ErrName(ctx.Resource.GetAction().Name)))
	}
}

func (s *Reservation6Api) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	input, ok := ctx.Resource.GetAction().Input.(*resource.BatchDeleteInput)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameDhcpReservation, resource.ActionBatchDelete))
	}

	if err := s.Service.BatchDeleteReservation6s(ctx.Resource.GetParent().GetID(), input.Ids); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return nil, nil
	}
}

func (s *Reservation6Api) actionImportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*excel.ImportFile)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameDhcpReservation, excel.ActionNameImport))
	}

	if resp, err := s.Service.ImportExcel(file, ctx.Resource.GetParent().GetID()); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return resp, nil
	}
}

func (s *Reservation6Api) actionExportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if exportFile, err := s.Service.ExportExcel(ctx.Resource.GetParent().GetID()); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return exportFile, nil
	}
}

func (s *Reservation6Api) actionExportExcelTemplate() (interface{}, *resterror.APIError) {
	if file, err := s.Service.ExportExcelTemplate(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}
