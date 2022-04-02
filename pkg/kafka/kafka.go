package kafka

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"

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

type RollBackFunc func([]string)

func SendDHCPCmdWithNodes(isv4 bool, sentryNodes []string, cmd DHCPCmd, req proto.Message, rollback RollBackFunc) error {
	if len(sentryNodes) == 0 {
		return nil
	}

	nodes, err := GetDHCPNodesWithSentryNodes(sentryNodes, isv4)
	if err != nil {
		return err
	}

	nodesForSucceed, err := GetDHCPAgentService().SendDHCPCmdWithNodes(nodes, cmd, req)
	if err != nil && rollback != nil {
		rollback(nodesForSucceed)
	}

	return err
}

func GetDHCPNodesWithSentryNodes(sentryNodes []string, isv4 bool) ([]string, error) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, fmt.Errorf("get dhcp nodes failed: %s", err.Error())
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
	hasVirtualIp := false
	for _, node := range dhcpNodes.GetNodes() {
		hasSentry := IsAgentService(node.GetServiceTags(), sentryRole)
		if hasSentry {
			if node.GetVirtualIp() != "" {
				sentryNodes = []string{node.GetIpv4()}
			}

			sentryNodeMap[node.GetIpv4()] = struct{}{}
		}

		if IsAgentService(node.GetServiceTags(), serverRole) {
			hasServer = true
			if hasSentry == false {
				if node.GetVirtualIp() != "" {
					hasVirtualIp = true
					serverNodes = []string{node.GetIpv4()}
				}

				if hasVirtualIp == false {
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

func IsAgentService(tags []string, roles ...AgentRole) bool {
	for _, tag := range tags {
		for _, role := range roles {
			if tag == string(role) {
				return true
			}
		}
	}

	return false
}

func SendDHCPCmd(cmd DHCPCmd, req proto.Message, rollback RollBackFunc) error {
	nodes, err := getDHCPNodes()
	if err != nil {
		return err
	}

	nodesForSucceed, err := GetDHCPAgentService().SendDHCPCmdWithNodes(nodes, cmd, req)
	if err != nil && rollback != nil {
		rollback(nodesForSucceed)
	}

	return err
}

func getDHCPNodes() ([]string, error) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, fmt.Errorf("get dhcp nodes failed: %s", err.Error())
	}

	var serverNodes []string
	var sentryNodes []string
	hasServer := false
	hasSentryVirtualIp := false
	hasServerVirtualIp := false
	for _, node := range dhcpNodes.GetNodes() {
		hasSentry := IsAgentService(node.GetServiceTags(), AgentRoleSentry4, AgentRoleSentry6)
		if hasSentry {
			if node.GetVirtualIp() != "" {
				hasSentryVirtualIp = true
				sentryNodes = []string{node.GetIpv4()}
			}

			if hasSentryVirtualIp == false {
				sentryNodes = append(sentryNodes, node.GetIpv4())
			}
		}

		if IsAgentService(node.GetServiceTags(), AgentRoleServer4, AgentRoleServer6) {
			hasServer = true
			if hasSentry == false {
				if node.GetVirtualIp() != "" {
					hasServerVirtualIp = true
					serverNodes = []string{node.GetIpv4()}
				}

				if hasServerVirtualIp == false {
					serverNodes = append(serverNodes, node.GetIpv4())
				}
			}
		}
	}

	if len(sentryNodes) == 0 || hasServer == false {
		return nil, fmt.Errorf("no found valid dhcp sentry or server nodes")
	}

	return append(sentryNodes, serverNodes...), nil
}
