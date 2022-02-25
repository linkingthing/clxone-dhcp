package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
)

const (
	ClientClass4Option60 = "option vendor-class-identifier == '%s'"
)

type ClientClass4Api struct {
	Service *service.ClientClass4Service
}

func NewClientClass4Api() *ClientClass4Api {
	return &ClientClass4Api{Service: service.NewClientClass4Service()}
}

func (c *ClientClass4Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientClass := ctx.Resource.(*resource.ClientClass4)
	clientClass.SetID(clientClass.Name)
	if err := clientClass.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create clientclass4 %s failed: %s", clientClass.GetID(), err.Error()))
	}
	retClientClass, err := c.Service.Create(clientClass)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("add clientclass4 %s failed: %s", clientClass.Name, err.Error()))
	}
	return retClientClass, nil
}

func (c *ClientClass4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	clientClasses, err := c.Service.List()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list clientclass4s from db failed: %s", err.Error()))
	}
	return clientClasses, nil
}

func (c *ClientClass4Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclassID := ctx.Resource.(*resource.ClientClass4).GetID()
	clientclass, err := c.Service.Get(clientclassID)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get clientclass4 %s failed: %s", clientclassID, err.Error()))
	}
	return clientclass.(*resource.ClientClass4), nil
}

func (c *ClientClass4Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientclass := ctx.Resource.(*resource.ClientClass4)
	if err := clientclass.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update clientclass4 %s failed: %s", clientclass.GetID(), err.Error()))
	}
	retClientClass, err := c.Service.Update(clientclass)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update clientclass4 %s failed: %s", clientclass.GetID(), err.Error()))
	}
	return retClientClass, nil
}

func (c *ClientClass4Api) Delete(ctx *restresource.Context) *resterror.APIError {
	clientclassID := ctx.Resource.(*resource.ClientClass4).GetID()
	err := c.Service.Delete(clientclassID)
	if err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete clientclass4 %s failed: %s", clientclassID, err.Error()))
	}

	return nil
}
