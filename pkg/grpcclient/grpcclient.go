package grpcclient

import (
	"sync"

	"github.com/linkingthing/clxone-dhcp/pkg/pb"
	dhcp_agent "github.com/linkingthing/clxone-dhcp/pkg/pb/dhcp-agent"
	"google.golang.org/grpc"
)

type GrpcClient struct {
	DHCPClient dhcp_agent.DHCPManagerClient
}

var grpcClient *GrpcClient
var once sync.Once

func NewDhcpAgentClient() *grpc.ClientConn {
	conn, err := pb.NewClient("clxone-dhcp-agent-grpc")
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
