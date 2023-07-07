package service

import (
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type DhcpSentryService struct {
}

func NewDhcpSentryService() *DhcpSentryService {
	return &DhcpSentryService{}
}

func (h *DhcpSentryService) List() (interface{}, error) {
	sentry4 := &resource.DhcpSentry{}
	sentry4.SetID(string(DHCPVersion4))
	sentry6 := &resource.DhcpSentry{}
	sentry6.SetID(string(DHCPVersion6))
	return []*resource.DhcpSentry{sentry4, sentry6}, nil
}

func (h *DhcpSentryService) Get(ctx *restresource.Context) (restresource.Resource, error) {
	switch sentryID := ctx.Resource.GetID(); DHCPVersion(sentryID) {
	case DHCPVersion4, DHCPVersion6:
		return ctx.Resource, nil
	default:
		return nil, errorno.ErrNotFound(errorno.ErrNameDhcpSentryNode, sentryID)
	}
}
