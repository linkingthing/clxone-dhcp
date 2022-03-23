package service

import (
	"context"
	"fmt"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
)

type Agent6Service struct {
}

func NewAgent6Service() *Agent6Service {
	return &Agent6Service{}
}

func (h *Agent6Service) List() ([]*resource.Agent6, error) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, fmt.Errorf("get dhcpNodes failed: %s", err.Error())
	}

	var agents []*resource.Agent6
	for _, node := range dhcpNodes.GetNodes() {
		if node.GetServiceAlive() && kafka.IsAgentService(node.GetServiceTags(), kafka.AgentRoleSentry6) {
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

func (h *Agent6Service) Get(agent *resource.Agent6) error {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return fmt.Errorf("get dhcpNodes failed: %s", err.Error())
	}

	for _, node := range dhcpNodes.GetNodes() {
		if node.GetServiceAlive() && kafka.IsAgentService(node.GetServiceTags(), kafka.AgentRoleSentry6) &&
			node.Ipv4 == agent.GetID() {
			agent.Name = node.GetName()
			agent.Ip = node.GetIpv4()
			return nil
		}
	}

	return fmt.Errorf("no found dhcp agent %s", agent.GetID())
}
