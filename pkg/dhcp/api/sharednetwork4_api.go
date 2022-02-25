package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type SharedNetwork4Api struct {
	Service *service.SharedNetwork4Service
}

func NewSharedNetwork4Api() *SharedNetwork4Api {
	return &SharedNetwork4Api{Service: service.NewSharedNetwork4Service()}
}

func (s *SharedNetwork4Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	sharedNetwork4 := ctx.Resource.(*resource.SharedNetwork4)
	if err := sharedNetwork4.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create shared network4 params invalid: %s", err.Error()))
	}
	retsharedNetwork4, err := s.Service.Create(sharedNetwork4)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create shared network4 %s failed: %s",
				sharedNetwork4.Name, err.Error()))
	}

	return retsharedNetwork4, nil
}

func (s *SharedNetwork4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	sharedNetwork4s, err := s.Service.List(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list shared network4s from db failed: %s", err.Error()))
	}

	return sharedNetwork4s, nil
}

func (s *SharedNetwork4Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	sharedNetwork4 := ctx.Resource.(*resource.SharedNetwork4)
	retSharedNetwork4, err := s.Service.Get(sharedNetwork4)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get shared network4 %s failed: %s",
				sharedNetwork4.GetID(), err.Error()))
	}

	return retSharedNetwork4, nil
}

func (s *SharedNetwork4Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	sharedNetwork4 := ctx.Resource.(*resource.SharedNetwork4)
	if err := sharedNetwork4.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update shared network4 params invalid: %s", err.Error()))
	}
	retSharedNetwork4, err := s.Service.Update(sharedNetwork4)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update shared network4 %s failed: %s",
				sharedNetwork4.Name, err.Error()))
	}

	return retSharedNetwork4, nil
}

func (s *SharedNetwork4Api) Delete(ctx *restresource.Context) *resterror.APIError {
	sharedNetwork4Id := ctx.Resource.GetID()
	if err := s.Service.Delete(sharedNetwork4Id); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete shared network4 %s failed: %s",
				sharedNetwork4Id, err.Error()))
	}

	return nil
}
