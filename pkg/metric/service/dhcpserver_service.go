package service

import (
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type DhcpServerService struct {
}

func NewDhcpServerService() *DhcpServerService {
	return &DhcpServerService{}
}

func (h *DhcpServerService) List() (interface{}, error) {
	tagMap, err := kafka.GetDHCPAgentService().GetDHCPNodeTags()
	if err != nil {
		return nil, err
	}

	servers := make([]*resource.DhcpServer, 0, len(tagMap))
	if _, ok := tagMap[string(kafka.AgentRoleServer4)]; ok {
		server4 := &resource.DhcpServer{}
		server4.SetID(string(DHCPVersion4))
		servers = append(servers, server4)
	}
	if _, ok := tagMap[string(kafka.AgentRoleServer6)]; ok {
		server6 := &resource.DhcpServer{}
		server6.SetID(string(DHCPVersion6))
		servers = append(servers, server6)
	}

	return servers, nil
}

func (h *DhcpServerService) Get(ctx *restresource.Context) (restresource.Resource, error) {
	switch serverID := ctx.Resource.GetID(); DHCPVersion(serverID) {
	case DHCPVersion4, DHCPVersion6:
		return ctx.Resource, nil
	default:
		return nil, errorno.ErrNotFound(errorno.ErrNameDhcpServerNode, serverID)
	}
}
