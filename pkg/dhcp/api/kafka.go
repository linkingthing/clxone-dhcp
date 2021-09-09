package api

import (
	"fmt"

	"github.com/golang/protobuf/proto"

	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

func sendDHCPCmdWithNodes(sentryNodes []string, cmd dhcpservice.DHCPCmd, req proto.Message) ([]string, error) {
	if len(sentryNodes) == 0 {
		return nil, nil
	}

	nodes, err := getDHCPNodes(sentryNodes, true)
	if err != nil {
		return nil, err
	}

	return dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(nodes, cmd, req)
}

func getDHCPNodes(sentryNodes []string, isv4 bool) ([]string, error) {
	checks, services, err := GetConsulHandler().GetDHCPAgentChecksAndServices()
	if err != nil {
		return nil, err
	}

	serviceRoles := []AgentRole{AgentRoleSentry4, AgentRoleServer4}
	sentryRole := string(AgentRoleSentry4)
	serverRole := string(AgentRoleServer4)
	if isv4 == false {
		serviceRoles = []AgentRole{AgentRoleSentry6, AgentRoleServer6}
		sentryRole = string(AgentRoleSentry6)
		serverRole = string(AgentRoleServer6)
	}

	nodeRoles := make(map[string][]string)
	for _, check := range checks {
		if check.Validate() {
			if service := getSentryServiceWithServiceID(check.ServiceID, services,
				isAgentServiceMatchRoles, serviceRoles...); service != nil {
				nodeRoles[service.Address] = service.ServiceTags
			}
		}
	}

	for _, node := range sentryNodes {
		if roles, ok := nodeRoles[node]; ok == false ||
			util.SliceIndex(roles, sentryRole) == -1 {
			return nil, fmt.Errorf("node %s is not a dhcp sentry node", node)
		}
	}

	var serverNodes []string
	hasServer := false
	for node, roles := range nodeRoles {
		if util.SliceIndex(roles, serverRole) != -1 {
			hasServer = true
			if util.SliceIndex(roles, sentryRole) == -1 {
				serverNodes = append(serverNodes, node)
			}
		}
	}

	if hasServer == false {
		return nil, fmt.Errorf("no found valid dhcp server nodes")
	}

	return append(sentryNodes, serverNodes...), nil
}
