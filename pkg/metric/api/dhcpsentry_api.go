package api

import (
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/metric/service"
)

type DhcpSentryApi struct {
	Service *service.DhcpSentryService
}

func NewDhcpSentryApi() *DhcpSentryApi {
	return &DhcpSentryApi{Service: service.NewDhcpSentryService()}
}

func (h *DhcpSentryApi) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	ret, err := h.Service.List()
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return ret, nil
}

func (h *DhcpSentryApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	res, err := h.Service.Get(ctx)
	if err != nil {
		return nil, errorno.HandleAPIError(resterror.ServerError, err)
	}
	return res, nil
}
