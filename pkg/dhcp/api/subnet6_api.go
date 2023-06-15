package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-utils/excel"
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
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return subnet, nil
}

func (s *Subnet6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnets, err := s.Service.List(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return subnets, nil
}

func (s *Subnet6Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet, err := s.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return subnet, nil
}

func (s *Subnet6Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet6)
	if err := s.Service.Update(subnet); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return subnet, nil
}

func (s *Subnet6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := s.Service.Delete(ctx.Resource.(*resource.Subnet6)); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}

func (s *Subnet6Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case excel.ActionNameImport:
		return s.importExcel(ctx)
	case excel.ActionNameExport:
		return s.actionExportExcel()
	case excel.ActionNameExportTemplate:
		return s.exportExcelTemplate()
	case resource.ActionNameUpdateNodes:
		return s.actionUpdateNodes(ctx)
	case resource.ActionNameCouldBeCreated:
		return s.actionCouldBeCreated(ctx)
	case resource.ActionNameListWithSubnets:
		return s.actionListWithSubnets(ctx)
	default:
		return nil, errorno.HandleAPIError(resterror.InvalidAction,
			errorno.ErrUnknownOpt(errorno.ErrNameNetworkV6, ctx.Resource.GetAction().Name))
	}
}

func (s *Subnet6Api) actionUpdateNodes(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	subnetNode, ok := ctx.Resource.GetAction().Input.(*resource.SubnetNode)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameNetworkV6, resource.ActionNameUpdateNodes))
	}

	if err := s.Service.UpdateNodes(subnetID, subnetNode); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil, nil
}

func (s *Subnet6Api) actionCouldBeCreated(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	couldBeCreatedSubnet, ok := ctx.Resource.GetAction().Input.(*resource.CouldBeCreatedSubnet)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameNetworkV6, resource.ActionNameCouldBeCreated))
	}

	if err := s.Service.CouldBeCreated(couldBeCreatedSubnet); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil, nil
}

func (s *Subnet6Api) actionListWithSubnets(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetListInput, ok := ctx.Resource.GetAction().Input.(*resource.SubnetListInput)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameNetworkV6, resource.ActionNameListWithSubnets))
	}

	ret, err := s.Service.ListWithSubnets(subnetListInput)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return ret, nil
}

func (s *Subnet6Api) importExcel(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	file, ok := ctx.Resource.GetAction().Input.(*excel.ImportFile)
	if !ok {
		return nil, errorno.HandleAPIError(resterror.InvalidFormat,
			errorno.ErrInvalidFormat(errorno.ErrNameNetworkV6, errorno.ErrNameImport))
	}

	if resp, err := s.Service.ImportExcel(file); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return resp, nil
	}
}

func (s *Subnet6Api) actionExportExcel() (interface{}, *resterror.APIError) {
	if exportFile, err := s.Service.ExportExcel(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return exportFile, nil
	}
}

func (s *Subnet6Api) exportExcelTemplate() (interface{}, *resterror.APIError) {
	if file, err := s.Service.ExportExcelTemplate(); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	} else {
		return file, nil
	}
}
