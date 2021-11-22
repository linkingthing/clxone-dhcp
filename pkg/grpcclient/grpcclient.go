package grpcclient

import (
	"sync"

	"github.com/linkingthing/cement/log"

	"github.com/linkingthing/clxone-dhcp/config"
	pb "github.com/linkingthing/clxone-dhcp/pkg/proto"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type GrpcClient struct {
	DHCPClient pbdhcpagent.DHCPManagerClient
}

var grpcClient *GrpcClient
var once sync.Once

func NewDhcpAgentClient() error {
	conn, err := pb.NewConn(config.GetConfig().CallServices.DhcpAgent)
	if err != nil {
		return err
	}
	grpcClient = &GrpcClient{DHCPClient: pbdhcpagent.NewDHCPManagerClient(conn)}
	return nil
}

func GetDHCPAgentGrpcClient() pbdhcpagent.DHCPManagerClient {
	once.Do(func() {
		if err := NewDhcpAgentClient(); err != nil {
			log.Fatalf("create dhcp agent grpc client failed: %s", err.Error())
		}
	})
	return grpcClient.DHCPClient
}
