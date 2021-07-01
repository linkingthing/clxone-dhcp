package grpcclient

import (
	"sync"

	"github.com/linkingthing/clxone-dhcp/config"
	pb "github.com/linkingthing/clxone-dhcp/pkg/proto"
	dhcp_agent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"google.golang.org/grpc"
)

type GrpcClient struct {
	DHCPClient dhcp_agent.DHCPManagerClient
}

var grpcClient *GrpcClient
var once sync.Once

func NewDhcpAgentClient() *grpc.ClientConn {
	conn, err := pb.NewConn(config.GetConfig().CallServices.DhcpAgent)
	if err != nil {
		return nil
	}
	grpcClient = &GrpcClient{DHCPClient: dhcp_agent.NewDHCPManagerClient(conn)}
	return conn
}

func GetDHCPAgentGrpcClient() dhcp_agent.DHCPManagerClient {
	once.Do(func() {
		NewDhcpAgentClient()
	})
	return grpcClient.DHCPClient
}
