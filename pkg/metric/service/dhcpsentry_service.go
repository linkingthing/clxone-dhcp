package service

import (
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type DhcpSentryService struct {
}

func NewDhcpSentryService() *DhcpSentryService {
	return &DhcpSentryService{}
}

func (h *DhcpSentryService) List() (interface{}, error) {
	tagMap, err := kafka.GetDHCPAgentService().GetDHCPNodeTags()
	if err != nil {
		return nil, err
	}

	sentries := make([]*resource.DhcpSentry, 0, len(tagMap))
	if _, ok := tagMap[string(kafka.AgentRoleSentry4)]; ok {
		sentry4 := &resource.DhcpSentry{}
		sentry4.SetID(string(DHCPVersion4))
		sentries = append(sentries, sentry4)
	}
	if _, ok := tagMap[string(kafka.AgentRoleSentry6)]; ok {
		sentry6 := &resource.DhcpSentry{}
		sentry6.SetID(string(DHCPVersion6))
		sentries = append(sentries, sentry6)
	}

	return sentries, nil
}

func (h *DhcpSentryService) Get(ctx *restresource.Context) (restresource.Resource, error) {
	switch sentryID := ctx.Resource.GetID(); DHCPVersion(sentryID) {
	case DHCPVersion4, DHCPVersion6:
		return ctx.Resource, nil
	default:
		return nil, errorno.ErrNotFound(errorno.ErrNameDhcpSentryNode, sentryID)
	}
}
