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

func CallDhcpAgentGrpc(f func(ctx context.Context, client pbdhcpagent.DHCPManagerClient) error) error {
	conn, err := consulutil.NewGrpcConn(
		config.ConsulConfig, config.GetConfig().Consul.CallServices.DhcpAgent,
		DHCPTagServer4,
		DHCPTagServer6)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return f(ctx, pbdhcpagent.NewDHCPManagerClient(conn))
}
