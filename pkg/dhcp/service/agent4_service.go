package service

import (
	"context"
	"fmt"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
)

type Agent4Service struct {
}

func NewAgent4Service() *Agent4Service {
	return &Agent4Service{}
}

func (h *Agent4Service) List() ([]*resource.Agent4, error) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list dhcp nodes failed: %s", err.Error())
	}

	var agents []*resource.Agent4
	for _, node := range dhcpNodes.GetNodes() {
		if node.GetServiceAlive() && kafka.IsAgentService(node.GetServiceTags(), kafka.AgentRoleSentry4) {
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

func (h *Agent4Service) Get(agent *resource.Agent4) error {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return fmt.Errorf("get dhcp nodes failed: %s", err.Error())
	}

	for _, node := range dhcpNodes.GetNodes() {
		if node.GetServiceAlive() && kafka.IsAgentService(node.GetServiceTags(), kafka.AgentRoleSentry4) &&
			node.Ipv4 == agent.GetID() {
			agent.Name = node.GetName()
			agent.Ip = node.GetIpv4()
			return nil
		}
	}

	return fmt.Errorf("no found dhcp node %s", agent.GetID())
}

func GetNodeNames(isv4 bool) (map[string]string, error) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, fmt.Errorf("get dhcp nodes failed: %s", err.Error())
	}

	sentryRole := kafka.AgentRoleSentry4
	if isv4 == false {
		sentryRole = kafka.AgentRoleSentry6
	}

	nodeNames := make(map[string]string)
	for _, node := range dhcpNodes.GetNodes() {
		if kafka.IsAgentService(node.GetServiceTags(), sentryRole) {
			nodeNames[node.GetIpv4()] = node.GetName()
		}
	}

	return nodeNames, nil
}

func GetSentryVirtualIpNode(isv4 bool) (string, error) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return "", fmt.Errorf("get dhcp nodes failed: %s", err.Error())
	}

	sentryRole := kafka.AgentRoleSentry4
	if isv4 == false {
		sentryRole = kafka.AgentRoleSentry6
	}

	for _, node := range dhcpNodes.GetNodes() {
		if kafka.IsAgentService(node.GetServiceTags(), sentryRole) &&
			node.GetVirtualIp() != "" {
			return node.GetIpv4(), nil
		}
	}

	return "", nil
}
