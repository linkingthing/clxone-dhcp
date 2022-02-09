package grpcservice

import (
	"fmt"
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

func (g *GRPCService) GetListAllSubnet4(ctx context.Context, req *pbdhcp.GetSubnet4Request) (*pbdhcp.GetSubnet4Response, error) {
	if subnetList, err := service.GetDHCPService().GetAllSubnet4s(); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetSubnet4Response{Subnet4S: subnetList}, nil
	}
}

func (g *GRPCService) GetListSubnet4ByPrefixes(ctx context.Context, req *pbdhcp.GetSubnet4Request) (*pbdhcp.GetSubnet4Response, error) {
	if len(req.GetPrefixes()) == 0 {
		return nil, fmt.Errorf("getlistsubnet4byprefixes prefixes is nil ")
	}
	if subnetList, err := service.GetDHCPService().GetWithSubnet4sByPrefixes(req.GetPrefixes()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetSubnet4Response{Subnet4S: subnetList}, nil
	}
}

func (g *GRPCService) GetListPool4BySubnet4Id(ctx context.Context,
	req *pbdhcp.GetPool4BySubnet4IdRequest) (*pbdhcp.GetPool4BySubnet4IdResponse, error) {
	if subnetList, err := service.GetDHCPService().GetWithPool4BySubnet4Id(req.GetSubnet4Id()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetPool4BySubnet4IdResponse{Pool4S: subnetList}, nil
	}
}

func (g *GRPCService) GetListReservedPool4BySubnet4Id(ctx context.Context,
	req *pbdhcp.GetReservedPool4BySubnet4IdRequest) (*pbdhcp.GetReservedPool4BySubnet4IdResponse, error) {
	if tmpList, err := service.GetDHCPService().GetReservedPool4BySubnet4Id(req.GetSubnet4Id()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetReservedPool4BySubnet4IdResponse{ReservedPool4S: tmpList}, nil
	}
}

func (g *GRPCService) GetListReservation4BySubnet4Id(ctx context.Context,
	req *pbdhcp.GetReservation4BySubnet4IdRequest) (*pbdhcp.GetReservation4BySubnet4IdResponse, error) {
	if tmpList, err := service.GetDHCPService().GetReservation4BySubnet4Id(req.GetSubnet4Id()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetReservation4BySubnet4IdResponse{Reservation4S: tmpList}, nil
	}
}

func (g *GRPCService) GetListLease4BySubnet4Id(ctx context.Context,
	req *pbdhcp.GetLease4BySubnet4IdRequest) (*pbdhcp.GetLease4BySubnet4IdResponse, error) {
	if tmpList, err := service.GetDHCPService().GetWithLease4BySubnet4Id(req.GetSubnet4Id()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetLease4BySubnet4IdResponse{SubnetLease4S: tmpList}, nil
	}
}

func (g *GRPCService) GetListLease4ByIp(ctx context.Context,
	req *pbdhcp.GetLease4ByIpRequest) (*pbdhcp.GetLease4ByIpResponse, error) {
	if tmpList, err := service.GetDHCPService().GetWithLease4ByIp(req.GetSubnet4Id(), req.GetIp()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetLease4ByIpResponse{SubnetLease4S: tmpList}, nil
	}
}

func (g *GRPCService) GetListAllSubnet6(ctx context.Context, req *pbdhcp.GetSubnet6Request) (*pbdhcp.GetSubnet6Response, error) {
	if len(req.Prefixes) == 0 {
		if tmpList, err := service.GetDHCPService().GetAllSubnet6(); err != nil {
			return nil, err
		} else {
			return &pbdhcp.GetSubnet6Response{Subnet6S: tmpList}, nil
		}
	}
	if tmpList, err := service.GetDHCPService().GetWithSubnet6ByPrefixes(req.GetPrefixes()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetSubnet6Response{Subnet6S: tmpList}, nil
	}
}

func (g *GRPCService) GetListSubnet6ByPrefixes(ctx context.Context, req *pbdhcp.GetSubnet6Request) (*pbdhcp.GetSubnet6Response, error) {
	if len(req.Prefixes) == 0 {
		return nil, fmt.Errorf("getlistsubnet6byprefixes prefixes is nil ")
	}
	if tmpList, err := service.GetDHCPService().GetWithSubnet6ByPrefixes(req.GetPrefixes()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetSubnet6Response{Subnet6S: tmpList}, nil
	}
}

func (g *GRPCService) GetListPool6BySubnet6Id(ctx context.Context,
	req *pbdhcp.GetPool6BySubnet6IdRequest) (*pbdhcp.GetPool6BySubnet6IdResponse, error) {
	if tmpList, err := service.GetDHCPService().GetWithPool6BySubnet6Id(req.GetSubnet6Id()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetPool6BySubnet6IdResponse{Pool6S: tmpList}, nil
	}
}

func (g *GRPCService) GetListReservedPool6BySubnet6Id(ctx context.Context,
	req *pbdhcp.GetReservedPool6BySubnet6IdRequest) (*pbdhcp.GetReservedPool6BySubnet6IdResponse, error) {
	if tmpList, err := service.GetDHCPService().GetWithReservedPool6BySubnet6Id(req.GetSubnet6Id()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetReservedPool6BySubnet6IdResponse{ReservedPool6S: tmpList}, nil
	}
}

func (g *GRPCService) GetListReservation6BySubnet6Id(ctx context.Context,
	req *pbdhcp.GetReservation6BySubnet6IdRequest) (*pbdhcp.GetReservation6BySubnet6IdResponse, error) {
	if tmpList, err := service.GetDHCPService().GetWithReservation6BySubnet6Id(req.GetSubnet6Id()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetReservation6BySubnet6IdResponse{Reservation6S: tmpList}, nil
	}
}

func (g *GRPCService) GetListLease6BySubnet6Id(ctx context.Context,
	req *pbdhcp.GetLease6BySubnet6IdRequest) (*pbdhcp.GetLease6BySubnet6IdResponse, error) {
	if tmpList, err := service.GetDHCPService().GetWithLease6BySubnet6Id(req.GetSubnet6Id()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetLease6BySubnet6IdResponse{SubnetLease6: tmpList}, nil
	}
}

func (g *GRPCService) GetListLease6ByIp(ctx context.Context,
	req *pbdhcp.GetLease6ByIpRequest) (*pbdhcp.GetLease6ByIpResponse, error) {
	if tmpValue, err := service.GetDHCPService().GetWithLease6ByIp(req.GetSubnet6Id(), req.GetIp()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetLease6ByIpResponse{SubnetLease6: tmpValue}, nil
	}
}
