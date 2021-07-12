package api

import (
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type DhcpSentryHandler struct {
}

func NewDhcpSentryHandler() *DhcpSentryHandler {
	return &DhcpSentryHandler{}
}

func (h *DhcpSentryHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	sentry4 := &resource.DhcpSentry{}
	sentry4.SetID(string(DHCPVersion4))
	sentry6 := &resource.DhcpSentry{}
	sentry6.SetID(string(DHCPVersion6))
	return []*resource.DhcpSentry{sentry4, sentry6}, nil
}

func (h *DhcpSentryHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	switch sentryID := ctx.Resource.GetID(); DHCPVersion(sentryID) {
	case DHCPVersion4, DHCPVersion6:
		return ctx.Resource, nil
	default:
		return nil, resterror.NewAPIError(resterror.InvalidFormat, "no found dhcp sentry "+sentryID)
	}
}
