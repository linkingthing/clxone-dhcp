package api

import (
	"fmt"

	"github.com/linkingthing/clxone-utils/excel"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type Subnet4Handler struct {
	Service *service.Subnet4Service
}

func NewSubnet4Api() *Subnet4Handler {
	return &Subnet4Handler{Service: service.NewSubnet4Service()}
}

func (s *Subnet4Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := s.Service.Create(subnet); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return subnet, nil
}

func (s *Subnet4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnets, err := s.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return subnets, nil
}

func (s *Subnet4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet, err := s.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return subnet, nil
}

func (s *Subnet4Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := s.Service.Update(subnet); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return subnet, nil
}

func (s *Subnet4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := s.Service.Delete(ctx.Resource.(*resource.Subnet4)); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}

func (s *Subnet4Handler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case excel.ActionNameImport:
		return s.actionImportExcel(ctx)
	case excel.ActionNameExport:
		return s.actionExportExcel()
	case excel.ActionNameExportTemplate:
		return s.actionExportExcelTemplate()
	case resource.ActionNameUpdateNodes:
		return s.actionUpdateNodes(ctx)
	case resource.ActionNameCouldBeCreated:
		return s.actionCouldBeCreated(ctx)
	case resource.ActionNameListWithSubnets:
		return s.actionListWithSubnets(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (s *Subnet4Handler) actionImportExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*excel.ImportFile)
	if !ok {
		return nil, resterror.NewAPIError(resterror.ServerError, "action import subnet4s input invalid")
	}

	if resp, err := s.Service.ImportExcel(file); err != nil {
		return resp, resterror.NewAPIError(resterror.ServerError, err.Error())
	} else {
		return resp, nil
	}
}

func (s *Subnet4Handler) actionExportExcel() (interface{}, *resterror.APIError) {
	if exportFile, err := s.Service.ExportExcel(); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	} else {
		return exportFile, nil
	}
}

func (s *Subnet4Handler) actionExportExcelTemplate() (interface{}, *resterror.APIError) {
	if file, err := s.Service.ExportExcelTemplate(); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	} else {
		return file, nil
	}
}

func (s *Subnet4Handler) actionUpdateNodes(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	subnetNode, ok := ctx.Resource.GetAction().Input.(*resource.SubnetNode)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action update subnet4 %s nodes input invalid", subnetID))
	}

	if err := s.Service.UpdateNodes(subnetID, subnetNode); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil, nil
}

func (s *Subnet4Handler) actionCouldBeCreated(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	couldBeCreatedSubnet, ok := ctx.Resource.GetAction().Input.(*resource.CouldBeCreatedSubnet)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			"action check subnet4 could be created input invalid")
	}

	if err := s.Service.CouldBeCreated(couldBeCreatedSubnet); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil, nil
}

func (s *Subnet4Handler) actionListWithSubnets(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetListInput, ok := ctx.Resource.GetAction().Input.(*resource.SubnetListInput)
	if !ok {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			"action list subnet4s input invalid")
	}

	ret, err := s.Service.ListWithSubnets(subnetListInput)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return ret, nil
}
