package service

import (
	"fmt"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
)

type Agent6Service struct {
}

func NewAgent6Service() *Agent6Service {
	return &Agent6Service{}
}

func (h *Agent6Service) List() ([]*resource.Agent6, error) {
	nodeMap, err := GetAgentInfo(true, kafka.AgentRoleSentry6)
	if err != nil {
		return nil, err
	}

	agents := make([]*resource.Agent6, 0, len(nodeMap))
	for id, node := range nodeMap {
		agent := &resource.Agent6{
			Name: node.Name,
			Ips:  node.Ips,
		}
		agent.SetID(id)
		agents = append(agents, agent)
	}

	return agents, nil
}

func (h *Agent6Service) Get(agent *resource.Agent6) error {
	nodeMap, err := GetAgentInfo(true, kafka.AgentRoleSentry6)
	if err != nil {
		return err
	}

	if node, ok := nodeMap[agent.GetID()]; ok {
		agent.Name = node.Name
		agent.Ips = node.Ips
		return nil
	}

	return fmt.Errorf("no found dhcp node %s", agent.GetID())
}
