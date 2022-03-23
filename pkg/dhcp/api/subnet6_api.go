package api

import (
	"fmt"
	csvutil "github.com/linkingthing/clxone-utils/csv"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type Subnet6Api struct {
	Service *service.Subnet6Service
}

func NewSubnet6Api() *Subnet6Api {
	return &Subnet6Api{Service: service.NewSubnet6Service()}
}

func (s *Subnet6Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet6)
	if err := s.Service.Create(subnet); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return subnet, nil
}

func (s *Subnet6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnets, err := s.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return subnets, nil
}

func (s *Subnet6Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet, err := s.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return subnet, nil
}

func (s *Subnet6Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet6)
	if err := s.Service.Update(subnet); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return subnet, nil
}

func (s *Subnet6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := s.Service.Delete(ctx.Resource.(*resource.Subnet6)); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}

func (s *Subnet6Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case csvutil.ActionNameImportCSV:
		return s.importCSV(ctx)
	case csvutil.ActionNameExportCSV:
		return s.actionExportCSV()
	case csvutil.ActionNameExportCSVTemplate:
		return s.exportCSVTemplate()
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

func (s *Subnet6Api) actionUpdateNodes(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	subnetNode, ok := ctx.Resource.GetAction().Input.(*resource.SubnetNode)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action update subnet6 %s nodes input invalid", subnetID))
	}

	if err := s.Service.UpdateNodes(subnetID, subnetNode); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet6 %s nodes failed: %s", subnetID, err.Error()))
	}

	return nil, nil
}

func (s *Subnet6Api) actionCouldBeCreated(ctx *restresource.Context) (interface{}, *resterror.APIError) {
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

func (s *Subnet6Api) actionListWithSubnets(ctx *restresource.Context) (interface{}, *resterror.APIError) {
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

func (s *Subnet6Api) importCSV(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*csvutil.ImportFile)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action importcsv input invalid"))
	}

	if err := s.Service.ImportCSV(file); err !=nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("import subnet6 csv failed:%s",err.Error()))
	}

	return nil, nil
}

func (s *Subnet6Api) actionExportCSV() (interface{}, *resterror.APIError) {
	if exportFile, err := s.Service.ExportCSV(); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export subnet6 csv failed: %s", err.Error()))
	} else {
		return exportFile, nil
	}
}

func (s *Subnet6Api) exportCSVTemplate() (interface{}, *resterror.APIError) {
	if file, err := s.Service.ExportCSVTemplate(); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("export subnet6 template failed: %s", err.Error()))
	} else {
		return file, nil
	}
}
