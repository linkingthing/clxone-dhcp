package api

import (
	"fmt"

	"github.com/linkingthing/clxone-utils/excel"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type Reservation4Api struct {
	Service *service.Reservation4Service
}

func NewReservation4Api() *Reservation4Api {
	return &Reservation4Api{Service: service.NewReservation4Service()}
}

func (r *Reservation4Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation := ctx.Resource.(*resource.Reservation4)
	if err := r.Service.Create(ctx.Resource.GetParent().(*resource.Subnet4), reservation); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return reservation, nil
}

func (r *Reservation4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	reservations, err := r.Service.List(ctx.Resource.GetParent().(*resource.Subnet4))
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return reservations, nil
}

func (r *Reservation4Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation, err := r.Service.Get(ctx.Resource.GetParent().(*resource.Subnet4), ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return reservation, nil
}

func (r *Reservation4Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := r.Service.Delete(
		ctx.Resource.GetParent().(*resource.Subnet4),
		ctx.Resource.(*resource.Reservation4)); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}

func (r *Reservation4Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	reservation := ctx.Resource.(*resource.Reservation4)
	if err := r.Service.Update(ctx.Resource.GetParent().GetID(), reservation); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return reservation, nil
}

func (s *Reservation4Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionBatchDelete:
		return s.actionBatchDelete(ctx)
	case excel.ActionNameImport:
		return s.actionImportExcel(ctx)
	case excel.ActionNameExport:
		return s.actionExportExcel()
	case excel.ActionNameExportTemplate:
		return s.actionExportExcelTemplate()
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (s *Reservation4Api) actionBatchDelete(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	input, ok := ctx.Resource.GetAction().Input.(*resource.BatchDeleteInput)
	if !ok {
		return nil, resterror.NewAPIError(resterror.ServerError, "action batch delete input invalid")
	}

	if err := s.Service.BatchDeleteReservation4s(ctx.Resource.GetParent().GetID(), input.Ids); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	} else {
		return nil, nil
	}
}

func (s *Reservation4Api) actionImportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*excel.ImportFile)
	if !ok {
		return nil, resterror.NewAPIError(resterror.ServerError, "action import reservation4s input invalid")
	}

	if resp, err := s.Service.ImportExcel(file); err != nil {
		return resp, resterror.NewAPIError(resterror.ServerError, err.Error())
	} else {
		return resp, nil
	}
}

func (s *Reservation4Api) actionExportExcel() (interface{}, *resterror.APIError) {
	if exportFile, err := s.Service.ExportExcel(); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	} else {
		return exportFile, nil
	}
}

func (s *Reservation4Api) actionExportExcelTemplate() (interface{}, *resterror.APIError) {
	if file, err := s.Service.ExportExcelTemplate(); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	} else {
		return file, nil
	}
}
