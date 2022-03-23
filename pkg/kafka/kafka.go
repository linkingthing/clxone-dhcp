package kafka

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/linkingthing/cement/slice"

	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
)

type AgentRole string

const (
	AgentRoleSentry4 AgentRole = "sentry4"
	AgentRoleServer4 AgentRole = "server4"
	AgentRoleSentry6 AgentRole = "sentry6"
	AgentRoleServer6 AgentRole = "server6"
)

func SendDHCPCmdWithNodes(isv4 bool, sentryNodes []string, cmd DHCPCmd, req proto.Message) ([]string, error) {
	if len(sentryNodes) == 0 {
		return nil, nil
	}

	nodes, err := GetDHCPNodes(sentryNodes, isv4)
	if err != nil {
		return nil, err
	}

	return GetDHCPAgentService().SendDHCPCmdWithNodes(nodes, cmd, req)
}

func GetDHCPNodes(sentryNodes []string, isv4 bool) ([]string, error) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, err
	}

	sentryRole := AgentRoleSentry4
	serverRole := AgentRoleServer4
	if isv4 == false {
		sentryRole = AgentRoleSentry6
		serverRole = AgentRoleServer6
	}

	var serverNodes []string
	sentryNodeMap := make(map[string]struct{})
	hasServer := false
	for _, node := range dhcpNodes.GetNodes() {
		if node.GetServiceAlive() {
			hasSentry := IsAgentService(node.GetServiceTags(), sentryRole)
			if hasSentry {
				sentryNodeMap[node.GetIpv4()] = struct{}{}
			}

			if IsAgentService(node.GetServiceTags(), serverRole) {
				hasServer = true
				if hasSentry == false || slice.SliceIndex(sentryNodes, node.GetIpv4()) == -1 {
					serverNodes = append(serverNodes, node.GetIpv4())
				}
			}
		}
	}

	if hasServer == false {
		return nil, fmt.Errorf("no found valid dhcp server nodes")
	}

	for _, sentryNode := range sentryNodes {
		if _, ok := sentryNodeMap[sentryNode]; ok == false {
			return nil, fmt.Errorf("invalid sentry node %s", sentryNode)
		}
	}

	return append(sentryNodes, serverNodes...), nil
}

func IsAgentService(tags []string, role AgentRole) bool {
	for _, tag := range tags {
		if tag == string(role) {
			return true
		}
	}

	return false
}
