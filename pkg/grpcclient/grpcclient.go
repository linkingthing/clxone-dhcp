package grpcclient

import (
	dhcp_agent "github.com/linkingthing/clxone-dhcp/pkg/pb/dhcp-agent"
	"google.golang.org/grpc"
)

type GrpcClient struct {
	DHCPClient dhcp_agent.DHCPManagerClient
}

var grpcClient *GrpcClient

func NewDhcpClient(conn *grpc.ClientConn) {
	grpcClient = &GrpcClient{DHCPClient: dhcp_agent.NewDHCPManagerClient(conn)}
}

func GetDHCPGrpcClient() dhcp_agent.DHCPManagerClient {
	return grpcClient.DHCPClient
}
