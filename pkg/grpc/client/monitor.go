package client

import (
	"sync"

	"github.com/linkingthing/cement/log"

	"github.com/linkingthing/clxone-dhcp/config"
	pb "github.com/linkingthing/clxone-dhcp/pkg/proto"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
)

type MonitorGrpcClient struct {
	client pbmonitor.MonitorServiceClient
}

var gMonitorGrpcClient *MonitorGrpcClient
var monitorOnce sync.Once

func newMonitorGrpcClient() error {
	conn, err := pb.NewConn(config.GetConfig().CallServices.Monitor)
	if err != nil {
		return err
	}

	gMonitorGrpcClient = &MonitorGrpcClient{client: pbmonitor.NewMonitorServiceClient(conn)}
	return nil
}

func GetMonitorGrpcClient() pbmonitor.MonitorServiceClient {
	monitorOnce.Do(func() {
		if err := newMonitorGrpcClient(); err != nil {
			log.Fatalf("create monitor grpc client failed: %s", err.Error())
		}
	})
	return gMonitorGrpcClient.client
}
