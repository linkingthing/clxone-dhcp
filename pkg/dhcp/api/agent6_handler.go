package api

import (
	"context"
	"fmt"

	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
)

type Agent6Handler struct {
}

func NewAgent6Handler() *Agent6Handler {
	return &Agent6Handler{}
}

func (h *Agent6Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp agent6s failed: %s", err.Error()))
	}

	var agents []*resource.Agent6
	for _, node := range dhcpNodes.GetNodes() {
		if node.GetServiceAlive() && IsAgentService(node.GetServiceTags(), AgentRoleSentry6) {
			agent6 := &resource.Agent6{
				Name: node.GetName(),
				Ip:   node.GetIpv4(),
			}
			agent6.SetID(node.GetIpv4())
			if node.GetVirtualIp() != "" {
				return []*resource.Agent6{agent6}, nil
			} else {
				agents = append(agents, agent6)
			}
		}
	}

	return agents, nil
}

func (h *Agent6Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	agent := ctx.Resource.(*resource.Agent6)
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp agent6s failed: %s", err.Error()))
	}

	for _, node := range dhcpNodes.GetNodes() {
		if node.GetServiceAlive() && IsAgentService(node.GetServiceTags(), AgentRoleSentry6) &&
			node.Ipv4 == agent.GetID() {
			agent.Name = node.GetName()
			agent.Ip = node.GetIpv4()
			return agent, nil
		}
	}

	return nil, resterror.NewAPIError(resterror.NotFound,
		fmt.Sprintf("no found dhcp agent %s", agent.GetID()))
}
