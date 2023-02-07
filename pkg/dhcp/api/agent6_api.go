package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type Agent6Api struct {
	Service *service.Agent6Service
}

func NewAgent6Api() *Agent6Api {
	return &Agent6Api{Service: service.NewAgent6Service()}
}

func (h *Agent6Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	agents, err := h.Service.List()
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return agents, nil
}

func (h *Agent6Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	agent := ctx.Resource.(*resource.Agent6)
	if err := h.Service.Get(agent); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return agent, nil
}
