package service

import (
	"context"
	"time"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/errorno"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
)

func GetDHCPNodes() (response *pbmonitor.GetNodesResponse, err error) {
	if err = callWardenGrpc(func(client pbmonitor.MonitorServiceClient) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		response, err = client.GetNodes(ctx, &pbmonitor.GetNodesRequest{
			QueryType:   pbmonitor.NodeQueryType_service_role,
			ServiceRole: "dhcp",
		})
		return err
	}); err != nil {
		err = errorno.ErrNetworkError(errorno.ErrNameDhcpNode, err.Error())
	}
	return
}

func IsNodeMaster(hostname string) (response *pbmonitor.IsNodeMasterResponse, err error) {
	if err = callWardenGrpc(func(client pbmonitor.MonitorServiceClient) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		response, err = client.IsNodeMaster(ctx, &pbmonitor.IsNodeMasterRequest{
			Hostname: hostname,
		})
		return err
	}); err != nil {
		return
	}
	return
}

func callWardenGrpc(f func(client pbmonitor.MonitorServiceClient) error) error {
	conn, err := getServiceConnect(config.GetConfig().Server.Hostname, config.GetConfig().Consul.CallServices.Warden)
	if err != nil {
		return err
	}
	defer conn.Close()

	return f(pbmonitor.NewMonitorServiceClient(conn))
}
