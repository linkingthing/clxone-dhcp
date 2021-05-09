package transports

import (
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/services"
	"github.com/linkingthing/clxone-dhcp/pkg/pb/dhcp"
	"golang.org/x/net/context"
)

type DHCPServiceBinding struct {
	DHCPService *services.DHCPService
}

func (b DHCPServiceBinding) GetSubnetByID(ctx context.Context, req *dhcp.GetSubnetByIDReq) (*dhcp.Subnet, error) {
	subnet, err := b.DHCPService.GetSubnetByID(req.Id)
	if err != nil {
		return nil, err
	}
	return &dhcp.Subnet{
		Id:     subnet.ID,
		Subnet: subnet.Subnet,
		Ipnet: &dhcp.IPNet{
			Ip:     string(subnet.Ipnet.IP),
			IpMask: subnet.Ipnet.Mask.String(),
		},
		SubnetId:              subnet.SubnetId,
		ValidLifetime:         subnet.ValidLifetime,
		MaxValidLifetime:      subnet.MaxValidLifetime,
		MinValidLifetime:      subnet.MinValidLifetime,
		DomainServers:         subnet.DomainServers,
		Routers:               subnet.Routers,
		ClientClass:           subnet.ClientClass,
		IfaceName:             subnet.IfaceName,
		RelayAgentAddresses:   subnet.RelayAgentAddresses,
		RelayAgentInterfaceId: subnet.RelayAgentInterfaceId,
		Tags:                  subnet.Tags,
		NetworkType:           subnet.NetworkType,
		Capacity:              subnet.Capacity,
		UsedRatio:             subnet.UsedRatio,
		UsedCount:             subnet.UsedCount,
		Version:               uint32(subnet.Version),
	}, nil
}
