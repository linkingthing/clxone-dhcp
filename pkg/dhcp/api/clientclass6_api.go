package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type ClientClass6Api struct {
	Service *service.ClientClass6Service
}

func NewClientClass6Api() *ClientClass6Api {
	return &ClientClass6Api{Service: service.NewClientClass6Service()}
}

func (c *ClientClass6Api) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientClass := ctx.Resource.(*resource.ClientClass6)
	if err := c.Service.Create(clientClass); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return clientClass, nil
}

func (c *ClientClass6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	clientClasses, err := c.Service.List(util.GenStrConditionsFromFilters(ctx.GetFilters(),
		resource.SqlColumnName, resource.SqlColumnName, resource.SqlColumnCode))
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return clientClasses, nil
}

func (c *ClientClass6Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientClass, err := c.Service.Get(ctx.Resource.GetID())
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return clientClass, nil
}

func (c *ClientClass6Api) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	clientClass := ctx.Resource.(*resource.ClientClass6)
	if err := c.Service.Update(clientClass); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return clientClass, nil
}

func (c *ClientClass6Api) Delete(ctx *restresource.Context) *resterror.APIError {
	if err := c.Service.Delete(ctx.Resource.GetID()); err != nil {
		return resterror.NewAPIError(resterror.ServerError, err.Error())
	}

	return nil
}
