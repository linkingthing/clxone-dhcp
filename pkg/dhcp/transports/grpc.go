package transports

import (
	"fmt"
	"net"

	"golang.org/x/net/context"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
)

type DHCPServiceBinding struct {
	DHCPService *service.DHCPService
}

var _ dhcp.DhcpServiceServer = DHCPServiceBinding{}

func (b DHCPServiceBinding) SearchSubnet(ctx context.Context,
	req *dhcp.SearchSubnetRequest) (*dhcp.SearchSubnetResponse, error) {
	subnets, err := b.DHCPService.GetSubnet4ByIDs(req.Id...)
	if err != nil {
		return nil, err
	}
	return transSearchSubnetResponse(subnets), nil
}

func (b DHCPServiceBinding) SearchClosestSubnet(ctx context.Context,
	req *dhcp.SearchClosestSubnetRequest) (*dhcp.SearchClosestSubnetResponse, error) {
	ip := net.ParseIP(req.GetIp())
	if ip == nil {
		return nil, fmt.Errorf("invalid ip %s", req.GetIp())
	}

	var subnet *dhcp.Subnet
	var err error
	if ip.To4() != nil {
		subnet, err = b.DHCPService.GetClosestSubnet4ByIDs(req.GetId(), req.GetIp())
	} else {
		subnet, err = b.DHCPService.GetClosestSubnet6ByIDs(req.GetId(), req.GetIp())
	}

	if err != nil {
		return nil, err
	}

	return &dhcp.SearchClosestSubnetResponse{Subnet: subnet}, nil
}

func transSearchSubnetResponse(subnets []*resource.Subnet4) *dhcp.SearchSubnetResponse {
	var gsubnets []*dhcp.Subnet
	for _, subnet := range subnets {
		gsubnets = append(gsubnets, &dhcp.Subnet{
			Id:        subnet.ID,
			Subnet:    subnet.Subnet,
			SubnetId:  uint32(subnet.SubnetId),
			Tags:      subnet.Tags,
			Capacity:  subnet.Capacity,
			UsedRatio: subnet.UsedRatio,
			UsedCount: subnet.UsedCount,
		})
	}

	return &dhcp.SearchSubnetResponse{
		Subnets: gsubnets,
	}
}
