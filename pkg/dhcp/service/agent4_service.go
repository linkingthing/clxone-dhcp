package service

import (
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	transport "github.com/linkingthing/clxone-dhcp/pkg/transport/service"
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
	nodeMap, err := GetAgentInfo(true, kafka.AgentRoleSentry4)
	if err != nil {
		return nil, err
	}

	agents := make([]*resource.Agent4, 0, len(nodeMap))
	for id, node := range nodeMap {
		agent := &resource.Agent4{
			Name: node.Name,
			Ips:  node.Ips,
		}
		agent.SetID(id)
		agents = append(agents, agent)
	}

	return agents, nil
}

func (h *Agent4Service) Get(agent *resource.Agent4) error {
	nodeMap, err := GetAgentInfo(true, kafka.AgentRoleSentry4)
	if err != nil {
		return err
	}
	if node, ok := nodeMap[agent.GetID()]; ok {
		agent.Name = node.Name
		agent.Ips = node.Ips
		return nil
	}

	return errorno.ErrNotFound(errorno.ErrNameDhcpNode, agent.GetID())
}

func GetNodeNames(isv4 bool) (map[string]string, error) {
	sentryRole := kafka.AgentRoleSentry4
	if !isv4 {
		sentryRole = kafka.AgentRoleSentry6
	}
	nodeMap, err := GetAgentInfo(false, sentryRole)
	if err != nil {
		return nil, err
	}
	nodeNames := make(map[string]string, len(nodeMap))
	for _, agent := range nodeMap {
		for _, ip := range agent.Ips {
			nodeNames[ip] = agent.Name
		}
	}
	return nodeNames, nil
}

func IsSentryHA(isv4 bool) (bool, error) {
	dhcpNodes, err := transport.GetDHCPNodes()
	if err != nil {
		return false, err
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

type Agent struct {
	Id   string
	Name string
	Ips  []string
}

func (a Agent) HasNode(node string) bool {
	for _, ip := range a.Ips {
		if ip == node {
			return true
		}
	}
	return false
}

func GetAgentInfo(alive bool, role ...kafka.AgentRole) (map[string]Agent, error) {
	dhcpNodes, err := transport.GetDHCPNodes()
	if err != nil {
		return nil, err
	}

	nodes := dhcpNodes.GetNodes()
	nodeMap := make(map[string]Agent, len(nodes))
	for _, node := range nodes {
		if alive && !node.GetServiceAlive() || !kafka.IsAgentService(node.GetServiceTags(), role...) {
			continue
		}
		if vip := node.VirtualIp; vip != "" {
			agent := Agent{
				Id:   vip,
				Name: vip,
				Ips:  []string{vip},
			}
			return map[string]Agent{agent.Id: agent}, nil
		}

		if node.Deploy == DeployModelAnycast {
			id := node.Name
			agent := nodeMap[id]
			agent.Id = id
			agent.Name = node.Name
			agent.Ips = append(agent.Ips, node.Ipv4)
			nodeMap[id] = agent
		} else {
			nodeMap[node.Ipv4] = Agent{
				Id:   node.Ipv4,
				Name: node.Name,
				Ips:  []string{node.Ipv4},
			}
		}
	}

	return nodeMap, nil
}
