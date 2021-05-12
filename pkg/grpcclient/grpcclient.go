package grpcclient

import (
	"github.com/linkingthing/clxone-dhcp/pkg/pb"
	dhcp_agent "github.com/linkingthing/clxone-dhcp/pkg/pb/dhcp-agent"
	"google.golang.org/grpc"
)

type GrpcClient struct {
	DHCPClient dhcp_agent.DHCPManagerClient
}

var grpcClient *GrpcClient

func NewDhcpAgentClient() *grpc.ClientConn {
	conn, err := pb.NewClient("clxone-dhcp-agent-grpc")
	if err != nil {
		return nil
	}
	grpcClient = &GrpcClient{DHCPClient: dhcp_agent.NewDHCPManagerClient(conn)}
	return conn
}

func GetDHCPAgentGrpcClient() dhcp_agent.DHCPManagerClient {
	if grpcClient == nil {
		NewDhcpAgentClient()
	}
	return grpcClient.DHCPClient
}
