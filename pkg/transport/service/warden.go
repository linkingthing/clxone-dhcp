package service

import (
	"context"
	"time"

	consulutil "github.com/linkingthing/clxone-utils/consul"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
)

func GetDHCPNodes() (response *pbmonitor.GetDHCPNodesResponse, err error) {
	if err = callWardenGrpc(func(client pbmonitor.MonitorServiceClient) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		response, err = client.GetDHCPNodes(ctx, &pbmonitor.GetDHCPNodesRequest{})
		return err
	}); err != nil {
		err = errorno.ErrNetworkError(errorno.ErrNameDhcpNode, err.Error())
	}
	return
}

func IsNodeMaster(node string) (response *pbmonitor.IsNodeMasterResponse, err error) {
	if err = callWardenGrpc(func(client pbmonitor.MonitorServiceClient) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		response, err = client.IsNodeMaster(ctx, &pbmonitor.IsNodeMasterRequest{
			Ip: node,
		})
		return err
	}); err != nil {
		return
	}
	return
}

func callWardenGrpc(f func(client pbmonitor.MonitorServiceClient) error) error {
	conn, err := consulutil.NewGrpcConn(config.ConsulConfig, config.GetConfig().Consul.CallServices.Warden)
	if err != nil {
		return err
	}
	defer conn.Close()

	return f(pbmonitor.NewMonitorServiceClient(conn))
}
