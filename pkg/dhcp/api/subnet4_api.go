package api

import (
	"fmt"

	csvutil "github.com/linkingthing/clxone-utils/csv"
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
	if err := subnet.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create subnet params invalid: %s", err.Error()))
	}
	retSubnet, err := s.Service.Create(subnet)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create subnet %s failed: %s", subnet.Subnet, err.Error()))
	}

	return retSubnet, nil
}

func (s *Subnet4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnets, err := s.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnet4s from db failed: %s", err.Error()))
	}
	return subnets, nil
}

func (s *Subnet4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	subnet, err := s.Service.Get(subnetID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet %s from db failed: %s", subnetID, err.Error()))
	}
	return subnet, nil
}

func (s *Subnet4Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := subnet.ValidateParams(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update subnet params invalid: %s", err.Error()))
	}
	retSubnet, err := s.Service.Update(subnet)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return retSubnet, nil
}

func (s *Subnet4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := s.Service.Delete(subnet); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return nil
}

func (s *Subnet4Handler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case csvutil.ActionNameImportCSV:
		return s.ActionImportCSV(ctx)
	case csvutil.ActionNameExportCSV:
		return s.ActionExportCSV(ctx)
	case csvutil.ActionNameExportCSVTemplate:
		return s.ActionExportCSVTemplate(ctx)
	case resource.ActionNameUpdateNodes:
		return s.ActionUpdateNodes(ctx)
	case resource.ActionNameCouldBeCreated:
		return s.ActionCouldBeCreated(ctx)
	case resource.ActionNameListWithSubnets:
		return s.ActionListWithSubnets(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (s *Subnet4Handler) ActionImportCSV(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*csvutil.ImportFile)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.ServerError, "action importcsv input invalid")
	}
	if _, err := s.Service.ImportCSV(file); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("import subnet4s from file %s failed: %s",
				file.Name, err.Error()))
	}
	return nil, nil
}

func (s *Subnet4Handler) ActionExportCSV(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if exportFile, err := s.Service.ExportCSV(); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export device failed: %s", err.Error()))
	} else {
		return exportFile, nil
	}
}

func (s *Subnet4Handler) ActionExportCSVTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	if file, err := s.Service.ExportCSVTemplate(); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export subnet4 template failed: %s", err.Error()))
	} else {
		return file, nil
	}
}

func (s *Subnet4Handler) ActionUpdateNodes(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	subnetNode, ok := ctx.Resource.GetAction().Input.(*resource.SubnetNode)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action update subnet4 %s nodes input invalid", subnetID))
	}
	_, err := s.Service.UpdateNodes(subnetID, subnetNode)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet4 %s nodes failed: %s", subnetID, err.Error()))
	}

	return nil, nil
}

func (s *Subnet4Handler) ActionCouldBeCreated(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	couldBeCreatedSubnet, ok := ctx.Resource.GetAction().Input.(*resource.CouldBeCreatedSubnet)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action check subnet could be created input invalid"))
	}
	_, err := s.Service.CouldBeCreated(couldBeCreatedSubnet)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("action check subnet could be created failed: %s", err.Error()))
	}

	return nil, nil
}

func (s *Subnet4Handler) ActionListWithSubnets(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetListInput, ok := ctx.Resource.GetAction().Input.(*resource.SubnetListInput)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action list subnet input invalid"))
	}
	ret, err := s.Service.ListWithSubnets(subnetListInput)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("action list subnet failed: %s", err.Error()))
	}
	return ret, nil
}
