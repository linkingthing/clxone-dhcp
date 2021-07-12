package api

import (
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
)

type DhcpServerHandler struct {
}

func NewDhcpServerHandler() *DhcpServerHandler {
	return &DhcpServerHandler{}
}

func (h *DhcpServerHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	server4 := &resource.DhcpServer{}
	server4.SetID(string(DHCPVersion4))
	server6 := &resource.DhcpServer{}
	server6.SetID(string(DHCPVersion6))
	return []*resource.DhcpServer{server4, server6}, nil
}

func (h *DhcpServerHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	switch serverID := ctx.Resource.GetID(); DHCPVersion(serverID) {
	case DHCPVersion4, DHCPVersion6:
		return ctx.Resource, nil
	default:
		return nil, resterror.NewAPIError(resterror.InvalidFormat, "no found dhcp server"+serverID)
	}
}
