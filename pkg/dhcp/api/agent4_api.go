package api

import (
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
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
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp agent4s failed: %s", err.Error()))
	}
	return agents, nil
}

func (h *Agent4Api) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	agent := ctx.Resource.(*resource.Agent4)
	retAgent, err := h.Service.Get(agent)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.NotFound,
			fmt.Sprintf("get dhcp agent4s %s failed %s", agent.GetID(), err.Error()))
	}
	return retAgent, nil
}
