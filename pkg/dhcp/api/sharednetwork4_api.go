package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type SharedNetwork4Api struct {
	Service *service.SharedNetwork4Service
}

func NewSharedNetwork4Api() *SharedNetwork4Api {
	return &SharedNetwork4Api{Service: service.NewSharedNetwork4Service()}
}

func (s *SharedNetwork4Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	sharedNetwork4 := ctx.Resource.(*resource.SharedNetwork4)
	if err := s.Service.Create(sharedNetwork4); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return sharedNetwork4, nil
}

func (s *SharedNetwork4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	sharedNetwork4s, err := s.Service.List(
		util.GenStrConditionsFromFilters(ctx.GetFilters(),
			util.FilterNameName, util.FilterNameName))
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return sharedNetwork4s, nil
}

func (s *SharedNetwork4Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	sharedNetwork4, err := s.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return sharedNetwork4, nil
}

func (s *SharedNetwork4Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	sharedNetwork4 := ctx.Resource.(*resource.SharedNetwork4)
	if err := s.Service.Update(sharedNetwork4); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return sharedNetwork4, nil
}

func (s *SharedNetwork4Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := s.Service.Delete(ctx.Resource.GetID()); err != nil {
		return errorno.HandleAPIError(resterror.ServerError, err)
	}

	return nil
}
