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

type AgentRole string

const (
	AgentRoleSentry4 AgentRole = "sentry4"
	AgentRoleServer4 AgentRole = "server4"
	AgentRoleSentry6 AgentRole = "sentry6"
	AgentRoleServer6 AgentRole = "server6"
)

type Agent4Handler struct {
}

func NewAgent4Handler() *Agent4Handler {
	return &Agent4Handler{}
}

func (h *Agent4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp agent4s failed: %s", err.Error()))
	}

	var agents []*resource.Agent4
	for _, node := range dhcpNodes.GetNodes() {
		if node.GetServiceAlive() && IsAgentService(node.GetServiceTags(), AgentRoleSentry4) {
			agent4 := &resource.Agent4{
				Name: node.GetName(),
				Ip:   node.GetIpv4(),
			}
			agent4.SetID(node.GetIpv4())
			if node.GetVirtualIp() != "" {
				return []*resource.Agent4{agent4}, nil
			} else {
				agents = append(agents, agent4)
			}
		}
	}

	return agents, nil
}

func IsAgentService(tags []string, role AgentRole) bool {
	for _, tag := range tags {
		if tag == string(role) {
			return true
		}
	}

	return false
}

func (h *Agent4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	agent := ctx.Resource.(*resource.Agent4)
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list dhcp agent4s failed: %s", err.Error()))
	}

	for _, node := range dhcpNodes.GetNodes() {
		if node.GetServiceAlive() && IsAgentService(node.GetServiceTags(), AgentRoleSentry4) &&
			node.Ipv4 == agent.GetID() {
			agent.Name = node.GetName()
			agent.Ip = node.GetIpv4()
			return agent, nil
		}
	}

	return nil, resterror.NewAPIError(resterror.NotFound,
		fmt.Sprintf("no found dhcp agent %s", agent.GetID()))
}
