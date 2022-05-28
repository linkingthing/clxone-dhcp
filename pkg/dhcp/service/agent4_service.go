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

	var agents []*resource.Agent4
	for _, node := range dhcpNodes.GetNodes() {
		if node.GetServiceAlive() && kafka.IsAgentService(node.GetServiceTags(), kafka.AgentRoleSentry4) {
			if vip := node.GetVirtualIp(); vip != "" {
				agent4 := &resource.Agent4{
					Name: vip,
					Ip:   vip,
				}
				agent4.SetID(node.GetIpv4())
				return []*resource.Agent4{agent4}, nil
			} else {
				agent4 := &resource.Agent4{
					Name: node.GetName(),
					Ip:   node.GetIpv4(),
				}
				agent4.SetID(node.GetIpv4())
				agents = append(agents, agent4)
			}
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
