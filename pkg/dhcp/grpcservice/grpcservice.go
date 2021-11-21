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
	if subnets, err := service.GetDHCPService().GetSubnet4WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet4WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet4WithIpResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GRPCService) GetSubnet6WithIp(ctx context.Context, req *pbdhcp.GetSubnet6WithIpRequest) (*pbdhcp.GetSubnet6WithIpResponse, error) {
	if subnets, err := service.GetDHCPService().GetSubnet6WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet6WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet6WithIpResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GRPCService) GetSubnets4WithIps(ctx context.Context, req *pbdhcp.GetSubnets4WithIpsRequest) (*pbdhcp.GetSubnets4WithIpsResponse, error) {
	if subnets, err := service.GetDHCPService().GetSubnets4WithIps(req.GetIps()); err != nil {
		return &pbdhcp.GetSubnets4WithIpsResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnets4WithIpsResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GRPCService) GetSubnets6WithIps(ctx context.Context, req *pbdhcp.GetSubnets6WithIpsRequest) (*pbdhcp.GetSubnets6WithIpsResponse, error) {
	if subnets, err := service.GetDHCPService().GetSubnets6WithIps(req.GetIps()); err != nil {
		return &pbdhcp.GetSubnets6WithIpsResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnets6WithIpsResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GRPCService) GetSubnet4AndLease4WithIp(ctx context.Context, req *pbdhcp.GetSubnet4AndLease4WithIpRequest) (*pbdhcp.GetSubnet4AndLease4WithIpResponse, error) {
	if ipv4Infos, err := service.GetDHCPService().GetSubnet4AndLease4WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet4AndLease4WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet4AndLease4WithIpResponse{
			Succeed:          true,
			Ipv4Informations: ipv4Infos,
		}, nil
	}
}

func (g *GRPCService) GetSubnet6AndLease6WithIp(ctx context.Context, req *pbdhcp.GetSubnet6AndLease6WithIpRequest) (*pbdhcp.GetSubnet6AndLease6WithIpResponse, error) {
	if ipv6Infos, err := service.GetDHCPService().GetSubnet6AndLease6WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet6AndLease6WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet6AndLease6WithIpResponse{
			Succeed:          true,
			Ipv6Informations: ipv6Infos,
		}, nil
	}
}

func (g *GRPCService) GetSubnets4AndLeases4WithIps(ctx context.Context, req *pbdhcp.GetSubnets4AndLeases4WithIpsRequest) (*pbdhcp.GetSubnets4AndLeases4WithIpsResponse, error) {
	if ipv4Infos, err := service.GetDHCPService().GetSubnets4AndLeases4WithIps(req.GetIps()); err != nil {
		return &pbdhcp.GetSubnets4AndLeases4WithIpsResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnets4AndLeases4WithIpsResponse{
			Succeed:          true,
			Ipv4Informations: ipv4Infos,
		}, nil
	}
}

func (g *GRPCService) GetSubnets6AndLeases6WithIps(ctx context.Context, req *pbdhcp.GetSubnets6AndLeases6WithIpsRequest) (*pbdhcp.GetSubnets6AndLeases6WithIpsResponse, error) {
	if ipv6Infos, err := service.GetDHCPService().GetSubnets6AndLeases6WithIps(req.GetIps()); err != nil {
		return &pbdhcp.GetSubnets6AndLeases6WithIpsResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnets6AndLeases6WithIpsResponse{
			Succeed:          true,
			Ipv6Informations: ipv6Infos,
		}, nil
	}
}
