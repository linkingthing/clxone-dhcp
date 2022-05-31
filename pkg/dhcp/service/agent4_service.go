package service

import (
	"context"
	"fmt"
	"time"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
)

const (
	DeployModelSingleton string = "singleton"
	DeployModelCluster   string = "cluster"
	DeployModelHa        string = "ha"
	DeployModelAnycast   string = "anycast"
)

type Agent4Service struct {
}

func NewAgent4Service() *Agent4Service {
	return &Agent4Service{}
}

func (h *Agent4Service) List() ([]*resource.Agent4, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(ctx,
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list dhcp nodes failed: %s", err.Error())
	}

	nameMap := mergeNodeIpOnName(dhcpNodes.GetNodes(), kafka.AgentRoleSentry4)

	agents := make([]*resource.Agent4, 0, len(nameMap))
	for name, ips := range nameMap {
		if len(ips) > 0 {
			agent := &resource.Agent4{
				Name: name,
				Ips:  ips,
			}
			agent.SetID(name)
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

func (h *Agent4Service) Get(agent *resource.Agent4) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(ctx,
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return fmt.Errorf("get dhcp nodes failed: %s", err.Error())
	}

	nameMap := mergeNodeIpOnName(dhcpNodes.GetNodes(), kafka.AgentRoleSentry4)
	if ips, ok := nameMap[agent.GetID()]; ok {
		agent.Ips = ips
		return nil
	}

	return fmt.Errorf("no found dhcp node %s", agent.GetID())
}

func GetNodeNames(isv4 bool) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(ctx,
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, fmt.Errorf("get dhcp nodes failed: %s", err.Error())
	}

	sentryRole := kafka.AgentRoleSentry4
	if !isv4 {
		sentryRole = kafka.AgentRoleSentry6
	}

	nodeNames := make(map[string]string)
	for _, node := range dhcpNodes.GetNodes() {
		if kafka.IsAgentService(node.GetServiceTags(), sentryRole) {
			if vip := node.GetVirtualIp(); vip != "" {
				return map[string]string{vip: vip}, nil
			} else {
				nodeNames[node.GetIpv4()] = node.GetName()
			}
		}
	}

	return nodeNames, nil
}

func IsSentryHA(isv4 bool) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(ctx,
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return false, fmt.Errorf("get dhcp nodes failed: %s", err.Error())
	}

	sentryRole := kafka.AgentRoleSentry4
	if !isv4 {
		sentryRole = kafka.AgentRoleSentry6
	}

	for _, node := range dhcpNodes.GetNodes() {
		if kafka.IsAgentService(node.GetServiceTags(), sentryRole) &&
			node.GetVirtualIp() != "" {
			return true, nil
		}
	}

	return false, nil
}

func mergeNodeIpOnName(nodes []*pbmonitor.Node, role kafka.AgentRole) map[string][]string {
	nameMap := make(map[string][]string, len(nodes))
	for _, node := range nodes {
		if node.GetServiceAlive() && kafka.IsAgentService(node.GetServiceTags(), role) {
			if node.Deploy == DeployModelHa && node.VirtualIp != "" {
				return map[string][]string{node.Name: {node.VirtualIp}}
			}
			nameMap[node.Name] = append(nameMap[node.Name], node.Ipv4)
		}
	}
	return nameMap
}
