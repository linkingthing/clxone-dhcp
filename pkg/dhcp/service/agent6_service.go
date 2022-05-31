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

type Agent6Service struct {
}

func NewAgent6Service() *Agent6Service {
	return &Agent6Service{}
}

func (h *Agent6Service) List() ([]*resource.Agent6, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(ctx,
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, fmt.Errorf("get dhcp nodes failed: %s", err.Error())
	}

	nameMap := mergeNodeIpOnName(dhcpNodes.GetNodes(), kafka.AgentRoleSentry6)

	agents := make([]*resource.Agent6, 0, len(nameMap))
	for name, ips := range nameMap {
		if len(ips) > 0 {
			agent := &resource.Agent6{
				Name: name,
				Ips:  ips,
			}
			agent.SetID(name)
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

func (h *Agent6Service) Get(agent *resource.Agent6) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(ctx,
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return fmt.Errorf("get dhcp nodes failed: %s", err.Error())
	}

	nameMap := mergeNodeIpOnName(dhcpNodes.GetNodes(), kafka.AgentRoleSentry6)
	if ips, ok := nameMap[agent.GetID()]; ok {
		agent.Ips = ips
		return nil
	}

	return fmt.Errorf("no found dhcp node %s", agent.GetID())
}
