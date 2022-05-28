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

type AgentStack string

const (
	AgentStack4    AgentStack = "4"
	AgentStack6    AgentStack = "6"
	AgentStackDual AgentStack = "dual"
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

func GetDHCPNodesWithSentryNodes(selectedSentryNodes []string, isv4 bool) ([]string, error) {
	if len(selectedSentryNodes) == 0 {
		return nil, nil
	}

	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, fmt.Errorf("get dhcp nodes failed: %s", err.Error())
	}

	sentryRole := AgentRoleSentry4
	serverRole := AgentRoleServer4
	if !isv4 {
		sentryRole = AgentRoleSentry6
		serverRole = AgentRoleServer6
	}

	var serverNodes []string
	var sentryNodes []string
	sentryNodeMap := make(map[string]struct{})
	hasServer := false
	hasVirtualIp := false
	for _, node := range dhcpNodes.GetNodes() {
		hasSentry := IsAgentService(node.GetServiceTags(), sentryRole)
		if hasSentry {
			if vip := node.GetVirtualIp(); vip != "" {
				if vip != selectedSentryNodes[0] {
					return nil, fmt.Errorf("only node with virtual ip could be selected for ha model")
				}

				sentryNodes = []string{node.GetIpv4()}
			}

			sentryNodeMap[node.GetIpv4()] = struct{}{}
		}

		if IsAgentService(node.GetServiceTags(), serverRole) {
			hasServer = true
			if !hasSentry {
				if node.GetVirtualIp() != "" {
					hasVirtualIp = true
					serverNodes = []string{node.GetIpv4()}
				}

				if !hasVirtualIp {
					serverNodes = append(serverNodes, node.GetIpv4())
				}
			}
		}
	}

	if !hasServer {
		return nil, fmt.Errorf("no found valid dhcp server nodes")
	}

	if len(sentryNodes) != 0 {
		return append(sentryNodes, serverNodes...), nil
	} else {
		for _, sentryNode := range selectedSentryNodes {
			if _, ok := sentryNodeMap[sentryNode]; !ok {
				return nil, fmt.Errorf("invalid sentry node %s", sentryNode)
			}
		}

		return append(selectedSentryNodes, serverNodes...), nil
	}
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
	sentryNodes, serverNodes, _, err := GetDHCPNodes(AgentStackDual)
	if err != nil {
		return err
	}

	nodesForSucceed, err := GetDHCPAgentService().SendDHCPCmdWithNodes(append(sentryNodes, serverNodes...), cmd, req)
	if err != nil && rollback != nil {
		rollback(nodesForSucceed)
	}

	return err
}

func GetDHCPNodes(stack AgentStack) ([]string, []string, string, error) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, nil, "", fmt.Errorf("get dhcp nodes failed: %s", err.Error())
	}

	sentryRoles := []AgentRole{AgentRoleSentry4, AgentRoleSentry6}
	serverRoles := []AgentRole{AgentRoleServer4, AgentRoleServer6}
	if stack == AgentStack4 {
		sentryRoles = []AgentRole{AgentRoleSentry4}
		serverRoles = []AgentRole{AgentRoleServer4}
	} else if stack == AgentStack6 {
		sentryRoles = []AgentRole{AgentRoleSentry6}
		serverRoles = []AgentRole{AgentRoleServer6}
	}

	var serverNodes []string
	var sentryNodes []string
	hasServer := false
	hasSentryVirtualIp := false
	hasServerVirtualIp := false
	var virtualIp string
	for _, node := range dhcpNodes.GetNodes() {
		hasSentry := IsAgentService(node.GetServiceTags(), sentryRoles...)
		if hasSentry {
			if vip := node.GetVirtualIp(); vip != "" {
				hasSentryVirtualIp = true
				virtualIp = vip
				sentryNodes = []string{node.GetIpv4()}
			}

			if !hasSentryVirtualIp {
				sentryNodes = append(sentryNodes, node.GetIpv4())
			}
		}

		if IsAgentService(node.GetServiceTags(), serverRoles...) {
			hasServer = true
			if !hasSentry {
				if node.GetVirtualIp() != "" {
					hasServerVirtualIp = true
					serverNodes = []string{node.GetIpv4()}
				}

				if !hasServerVirtualIp {
					serverNodes = append(serverNodes, node.GetIpv4())
				}
			}
		}
	}

	if len(sentryNodes) == 0 || !hasServer {
		return nil, nil, "", fmt.Errorf("no found valid dhcp sentry or server nodes")
	}

	return sentryNodes, serverNodes, virtualIp, nil
}
