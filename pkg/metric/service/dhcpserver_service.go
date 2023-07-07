package service

import (
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type DhcpServerService struct {
}

func NewDhcpServerService() *DhcpServerService {
	return &DhcpServerService{}
}

func (h *DhcpServerService) List() (interface{}, error) {
	server4 := &resource.DhcpServer{}
	server4.SetID(string(DHCPVersion4))
	server6 := &resource.DhcpServer{}
	server6.SetID(string(DHCPVersion6))
	return []*resource.DhcpServer{server4, server6}, nil
}

func (h *DhcpServerService) Get(ctx *restresource.Context) (restresource.Resource, error) {
	switch serverID := ctx.Resource.GetID(); DHCPVersion(serverID) {
	case DHCPVersion4, DHCPVersion6:
		return ctx.Resource, nil
	default:
		return nil, errorno.ErrNotFound(errorno.ErrNameDhcpServerNode, serverID)
	}
}
