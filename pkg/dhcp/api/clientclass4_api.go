package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type ClientClass4Api struct {
	Service *service.ClientClass4Service
}

func NewClientClass4Api() *ClientClass4Api {
	return &ClientClass4Api{Service: service.NewClientClass4Service()}
}

func (c *ClientClass4Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientClass := ctx.Resource.(*resource.ClientClass4)
	if err := c.Service.Create(clientClass); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return clientClass, nil
}

func (c *ClientClass4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	clientClasses, err := c.Service.List()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return clientClasses, nil
}

func (c *ClientClass4Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientClass, err := c.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return clientClass, nil
}

func (c *ClientClass4Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientClass := ctx.Resource.(*resource.ClientClass4)
	if err := c.Service.Update(clientClass); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return clientClass, nil
}

func (c *ClientClass4Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := c.Service.Delete(ctx.Resource.GetID()); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}
