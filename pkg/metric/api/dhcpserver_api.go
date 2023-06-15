package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/metric/service"
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
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return res, nil
}

func (h *DhcpServerApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	res, err := h.Service.Get(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return res, nil
}
