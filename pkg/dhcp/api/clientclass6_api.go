package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

type ClientClass6Api struct {
	Service *service.ClientClass6Service
}

func NewClientClass6Api() *ClientClass6Api {
	return &ClientClass6Api{Service: service.NewClientClass6Service()}
}

func (c *ClientClass6Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclass := ctx.Resource.(*resource.ClientClass6)
	clientclass.SetID(clientclass.Name)
	if err := clientclass.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create clientclass6 %s failed: %s", clientclass.GetID(), err.Error()))
	}
	retClientClass, err := c.Service.Create(clientclass)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("add clientclass %s failed: %s", clientclass.Name, err.Error()))
	}
	return retClientClass, nil
}

func (c *ClientClass6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	clientclasses, err := c.Service.List()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list clientclasses from db failed: %s", err.Error()))
	}
	return clientclasses, nil
}

func (c *ClientClass6Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclassID := ctx.Resource.(*resource.ClientClass6).GetID()
	clientclasses, err := c.Service.Get(clientclassID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get clientclass %s from db failed: %s",
				clientclassID, err.Error()))
	}
	return clientclasses, nil
}

func (c *ClientClass6Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclass := ctx.Resource.(*resource.ClientClass6)
	if err := clientclass.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update clientclass6 %s failed: %s", clientclass.GetID(), err.Error()))
	}
	retClientClass, err := c.Service.Update(clientclass)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update clientclass %s failed: %s", clientclass.GetID(), err.Error()))
	}

	return retClientClass, nil
}

func (c *ClientClass6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	clientclassID := ctx.Resource.(*resource.ClientClass6).GetID()
	err := c.Service.Delete(clientclassID)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete clientclass %s failed: %s", clientclassID, err.Error()))
	}

	return nil
}
