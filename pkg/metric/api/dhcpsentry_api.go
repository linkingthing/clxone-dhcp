package api

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/service"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"
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
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list dhcp sentry failed :%s", err.Error()))
	}
	return ret, nil
}

func (h *DhcpSentryApi) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	res, err := h.Service.Get(ctx)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, fmt.Sprintf("get dhcp sentry failed :%s", err.Error()))
	}
	return res, nil
}
