package transports

import (
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/services"
	"github.com/linkingthing/clxone-dhcp/pkg/pb/dhcp"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type DHCPServiceBinding struct {
	DHCPService *services.DHCPService
}

var _ dhcp.DhcpServiceServer = DHCPServiceBinding{}

func (b DHCPServiceBinding) SearchSubnet(ctx context.Context,
	req *dhcp.SearchSubnetRequest) (*dhcp.SearchSubnetResponse, error) {
	subnets, err := b.DHCPService.GetSubnetByIDs(req.Id...)
	if err != nil {
		return nil, err
	}
	return transSearchSubnetResponse(subnets), nil
}

func (b DHCPServiceBinding) SearchClosestSubnet(ctx context.Context,
	req *dhcp.SearchClosestSubnetRequest) (*dhcp.SearchClosestSubnetResponse, error) {
	subnet, err := b.DHCPService.GetClosestSubnetByIDs(req.Id, req.Ip)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return &dhcp.SearchClosestSubnetResponse{
		Subnet: &dhcp.Subnet{
			Id:          subnet.ID,
			Subnet:      subnet.Subnet,
			SubnetId:    subnet.SubnetId,
			Tags:        subnet.Tags,
			NetworkType: subnet.NetworkType,
			Capacity:    subnet.Capacity,
			UsedRatio:   subnet.UsedRatio,
			UsedCount:   subnet.UsedCount,
			Version:     uint32(subnet.Version),
		},
	}, nil
}

func transSearchSubnetResponse(subnets []*resource.Subnet) *dhcp.SearchSubnetResponse {
	var gsubnets []*dhcp.Subnet
	for _, subnet := range subnets {
		gsubnets = append(gsubnets, &dhcp.Subnet{
			Id:          subnet.ID,
			Subnet:      subnet.Subnet,
			SubnetId:    subnet.SubnetId,
			Tags:        subnet.Tags,
			NetworkType: subnet.NetworkType,
			Capacity:    subnet.Capacity,
			UsedRatio:   subnet.UsedRatio,
			UsedCount:   subnet.UsedCount,
			Version:     uint32(subnet.Version),
		})
	}

	return &dhcp.SearchSubnetResponse{
		Subnets: gsubnets,
	}
}
