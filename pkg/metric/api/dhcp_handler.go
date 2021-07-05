package api

import (
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

const (
	ResourceIDSentry4 = "dhcpsentry4"
	ResourceIDServer4 = "dhcpserver4"
	ResourceIDSentry6 = "dhcpsentry6"
	ResourceIDServer6 = "dhcpserver6"
)

type DhcpHandler struct {
	prometheusAddr string
}

func NewDhcpHandler(conf *config.DHCPConfig) *DhcpHandler {
	return &DhcpHandler{
		prometheusAddr: conf.Prometheus.Addr,
	}
}

func (h *DhcpHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	sentry4 := &resource.Dhcp{}
	sentry4.SetID(ResourceIDSentry4)
	server4 := &resource.Dhcp{}
	server4.SetID(ResourceIDServer4)
	sentry6 := &resource.Dhcp{}
	sentry6.SetID(ResourceIDSentry6)
	server6 := &resource.Dhcp{}
	server6.SetID(ResourceIDServer6)
	return []*resource.Dhcp{sentry4, server4, sentry6, server6}, nil
}

func (h *DhcpHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	switch dhcpID := ctx.Resource.GetID(); dhcpID {
	case ResourceIDSentry4, ResourceIDServer4, ResourceIDSentry6, ResourceIDServer6:
		return ctx.Resource, nil
	default:
		return nil, resterror.NewAPIError(resterror.InvalidFormat, "no found dhcp "+dhcpID)
	}
}
