package api

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/linkingthing/cement/slice"

	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
)

func sendDHCPCmdWithNodes(isv4 bool, sentryNodes []string, cmd dhcpservice.DHCPCmd, req proto.Message) ([]string, error) {
	if len(sentryNodes) == 0 {
		return nil, nil
	}

	nodes, err := getDHCPNodes(sentryNodes, isv4)
	if err != nil {
		return nil, err
	}

	return dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(nodes, cmd, req)
}

func getDHCPNodes(sentryNodes []string, isv4 bool) ([]string, error) {
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
			if IsAgentService(node.GetServiceTags(), sentryRole) {
				sentryNodeMap[node.GetIpv4()] = struct{}{}
			}

			if IsAgentService(node.GetServiceTags(), serverRole) {
				hasServer = true
				if IsAgentService(node.GetServiceTags(), sentryRole) == false ||
					slice.SliceIndex(sentryNodes, node.GetIpv4()) == -1 {
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
