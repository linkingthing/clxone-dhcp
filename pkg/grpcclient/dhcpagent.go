package grpcclient

import (
	"sync"

	"github.com/linkingthing/cement/log"

	"github.com/linkingthing/clxone-dhcp/config"
	pb "github.com/linkingthing/clxone-dhcp/pkg/proto"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type DHCPAgentGrpcClient struct {
	client pbdhcpagent.DHCPManagerClient
}

var gDHCPAgentGrpcClient *DHCPAgentGrpcClient
var dhcpagentOnce sync.Once

func newDHCPAgentGrpcClient() error {
	conn, err := pb.NewConn(config.GetConfig().CallServices.DhcpAgent)
	if err != nil {
		return err
	}

	gDHCPAgentGrpcClient = &DHCPAgentGrpcClient{client: pbdhcpagent.NewDHCPManagerClient(conn)}
	return nil
}

func GetDHCPAgentGrpcClient() pbdhcpagent.DHCPManagerClient {
	dhcpagentOnce.Do(func() {
		if err := newDHCPAgentGrpcClient(); err != nil {
			log.Fatalf("create dhcp agent grpc client failed: %s", err.Error())
		}
	})
	return gDHCPAgentGrpcClient.client
}
