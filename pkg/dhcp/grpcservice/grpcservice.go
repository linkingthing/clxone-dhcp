package grpcservice

import (
	"golang.org/x/net/context"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
)

type GRPCService struct {
}

func NewGRPCService() *GRPCService {
	return &GRPCService{}
}

func (g *GRPCService) GetSubnet4WithIp(ctx context.Context, req *pbdhcp.GetSubnet4WithIpRequest) (*pbdhcp.GetSubnet4WithIpResponse, error) {
	if subnet, err := service.GetDHCPService().GetSubnet4WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet4WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet4WithIpResponse{Succeed: true, Subnet: subnet}, nil
	}
}

func (g *GRPCService) GetSubnet6WithIp(ctx context.Context, req *pbdhcp.GetSubnet6WithIpRequest) (*pbdhcp.GetSubnet6WithIpResponse, error) {
	if subnet, err := service.GetDHCPService().GetSubnet6WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet6WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet6WithIpResponse{Succeed: true, Subnet: subnet}, nil
	}
}

func (g *GRPCService) GetSubnet4AndLease4WithIp(ctx context.Context, req *pbdhcp.GetSubnet4AndLease4WithIpRequest) (*pbdhcp.GetSubnet4AndLease4WithIpResponse, error) {
	if subnet, lease, err := service.GetDHCPService().GetSubnet4AndLease4WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet4AndLease4WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet4AndLease4WithIpResponse{
			Succeed: true,
			Subnet:  subnet,
			Lease:   lease,
		}, nil
	}
}

func (g *GRPCService) GetSubnet6AndLease6WithIp(ctx context.Context, req *pbdhcp.GetSubnet6AndLease6WithIpRequest) (*pbdhcp.GetSubnet6AndLease6WithIpResponse, error) {
	if subnet, lease, err := service.GetDHCPService().GetSubnet6AndLease6WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet6AndLease6WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet6AndLease6WithIpResponse{
			Succeed: true,
			Subnet:  subnet,
			Lease:   lease,
		}, nil
	}
}
