package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/linkingthing/cement/slice"
	"google.golang.org/grpc"

	"github.com/linkingthing/clxone-dhcp/config"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

const (
	DHCPTagServer4 = "server4"
	DHCPTagServer6 = "server6"
)

func CallDhcpAgentGrpc4(f func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error) error {
	return CallDhcpAgentGrpc(DHCPTagServer4, f)
}

func CallDhcpAgentGrpc6(f func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error) error {
	return CallDhcpAgentGrpc(DHCPTagServer6, f)
}

func CallDhcpAgentGrpc(serverTag string, f func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error) error {
	addr, err := getDHCPServerNode(serverTag)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithBlock(), grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)))
	if err != nil {
		return err
	}

	defer conn.Close()

	return f(ctx, pbdhcpagent.NewDHCPManagerClient(conn))
}

func getDHCPServerNode(serverTag string) (string, error) {
	dhcpNodes, err := GetDHCPNodes()
	if err != nil {
		return "", err
	}

	var serverNodes []string
	for _, node := range dhcpNodes.GetNodes() {
		if !node.GetServiceAlive() {
			continue
		}

		if slice.SliceIndex(node.GetServiceTags(), serverTag) != -1 {
			if vip := node.GetVirtualIp(); vip != "" {
				if serverTag == DHCPTagServer4 {
					return genGrpcAddr4(vip), nil
				} else {
					return genGrpcAddr6(vip), nil
				}
			}

			if serverTag == DHCPTagServer4 {
				serverNodes = append(serverNodes, genGrpcAddr4(node.GetIpv4()))
			} else {
				serverNodes = append(serverNodes, genGrpcAddr6(node.GetIpv6()))
			}
		}
	}

	if len(serverNodes) == 0 {
		return "", fmt.Errorf("no server alived")
	}

	rand.Seed(time.Now().UnixNano())
	return serverNodes[rand.Intn(len(serverNodes))], nil
}

func genGrpcAddr4(ip string) string {
	return ip + ":" + strconv.FormatUint(uint64(config.GetConfig().Server.AgentGrpcPort), 10)
}

func genGrpcAddr6(ip string) string {
	return "[" + ip + "]:" + strconv.FormatUint(uint64(config.GetConfig().Server.AgentGrpcPort), 10)
}
