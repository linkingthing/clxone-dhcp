package grpcclient

import (
	"sync"

	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/clxone-dhcp/config"
	pb "github.com/linkingthing/clxone-dhcp/pkg/proto"
	dhcp_agent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type GrpcClient struct {
	DHCPClient dhcp_agent.DHCPManagerClient
}

var grpcClient *GrpcClient
var once sync.Once

func NewDhcpAgentClient() error {
	conn, err := pb.NewConn(config.GetConfig().CallServices.DhcpAgent)
	if err != nil {
		return err
	}
	grpcClient = &GrpcClient{DHCPClient: dhcp_agent.NewDHCPManagerClient(conn)}
	return nil
}

func GetDHCPAgentGrpcClient() dhcp_agent.DHCPManagerClient {
	once.Do(func() {
		if err := NewDhcpAgentClient(); err != nil {
			log.Fatalf("create dhcp agent grpc client failed: %s", err.Error())
		}
	})
	return grpcClient.DHCPClient
}
