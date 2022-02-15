package api

import (
	"fmt"

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

func (s *Subnet6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnets, err := s.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnet6s failed: %s", err.Error()))
	}
	return subnets, nil
}

func (s *Subnet6Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	subnet, err := s.Service.Get(subnetID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet %s failed: %s", subnetID, err.Error()))
	}

	return subnet, nil
}

func (s *Subnet6Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet6)
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

func (s *Subnet6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.(*resource.Subnet6)
	if err := s.Service.Delete(subnet); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete subnet %s failed: %s", subnet.GetID(), err.Error()))
	}
	return nil
}

func (s *Subnet6Api) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
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

func (s *Subnet6Api) ActionUpdateNodes(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	subnetNode, ok := ctx.Resource.GetAction().Input.(*resource.SubnetNode)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action update subnet6 %s nodes input invalid", subnetID))
	}
	_, err := s.Service.UpdateNodes(subnetID, subnetNode)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet6 %s nodes failed: %s", subnetID, err.Error()))
	}

	return nil, nil
}

func (s *Subnet6Api) ActionCouldBeCreated(ctx *restresource.Context) (interface{}, *resterror.APIError) {
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

func (s *Subnet6Api) ActionListWithSubnets(ctx *restresource.Context) (interface{}, *resterror.APIError) {
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
