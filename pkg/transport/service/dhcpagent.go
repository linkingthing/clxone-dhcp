package service

import (
	"context"
	"time"

	consulutil "github.com/linkingthing/clxone-utils/consul"

	"github.com/linkingthing/clxone-dhcp/config"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

const (
	DHCPTagServer4 = "server4"
	DHCPTagServer6 = "server6"
)

func CallDhcpAgentGrpc4(f func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error) error {
	return CallDhcpAgentGrpc(f, DHCPTagServer4)
}

func CallDhcpAgentGrpc6(f func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error) error {
	return CallDhcpAgentGrpc(f, DHCPTagServer6)
}

func CallDhcpAgentGrpc(f func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error, serverTags ...string) error {
	conn, err := consulutil.NewGrpcConn(
		config.ConsulConfig, config.GetConfig().Consul.CallServices.DhcpAgent,
		serverTags...)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return f(ctx, pbdhcpagent.NewDHCPManagerClient(conn))
}
