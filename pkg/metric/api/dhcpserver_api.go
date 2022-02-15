package api

import (
	"fmt"

	"github.com/linkingthing/clxone-dhcp/pkg/metric/service"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"
)

type DhcpServerApi struct {
	Service *service.DhcpServerService
}

func NewDhcpServerApi() *DhcpServerApi {
	return &DhcpServerApi{Service: service.NewDhcpServerService()}
}

func (h *DhcpServerApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	res, err := h.Service.List()
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list dhcp server failed :%s", err.Error()))
	}
	return res, nil
}

func (h *DhcpServerApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	res, err := h.Service.Get(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("get dhcp server failed :%s", err.Error()))
	}
	return res, nil
}
