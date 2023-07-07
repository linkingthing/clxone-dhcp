package api

import (
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
)

type Agent4Api struct {
	Service *service.Agent4Service
}

func NewAgent4Api() *Agent4Api {
	return &Agent4Api{Service: service.NewAgent4Service()}
}

func (h *Agent4Api) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	agents, err := h.Service.List()
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return agents, nil
}

func (h *Agent4Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	agent4 := ctx.Resource.(*resource.Agent4)
	if err := h.Service.Get(agent4); err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}

	return agent4, nil
}
