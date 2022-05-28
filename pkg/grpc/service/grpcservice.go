package service

import (
	"context"

	dhcppb "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
)

type GrpcService struct{}

func NewGrpcService() *GrpcService {
	return &GrpcService{}
}

func (g *GrpcService) GetSubnet4WithIp(ctx context.Context, req *dhcppb.GetSubnet4WithIpRequest) (*dhcppb.GetSubnet4WithIpResponse, error) {
	if subnets, err := GetDHCPService().GetSubnet4WithIp(req.GetIp()); err != nil {
		return &dhcppb.GetSubnet4WithIpResponse{Succeed: false}, err
	} else {
		return &dhcppb.GetSubnet4WithIpResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GrpcService) GetSubnet6WithIp(ctx context.Context, req *dhcppb.GetSubnet6WithIpRequest) (*dhcppb.GetSubnet6WithIpResponse, error) {
	if subnets, err := GetDHCPService().GetSubnet6WithIp(req.GetIp()); err != nil {
		return &dhcppb.GetSubnet6WithIpResponse{Succeed: false}, err
	} else {
		return &dhcppb.GetSubnet6WithIpResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GrpcService) GetSubnets4WithIps(ctx context.Context, req *dhcppb.GetSubnets4WithIpsRequest) (*dhcppb.GetSubnets4WithIpsResponse, error) {
	if subnets, err := GetDHCPService().GetSubnets4WithIps(req.GetIps()); err != nil {
		return &dhcppb.GetSubnets4WithIpsResponse{Succeed: false}, err
	} else {
		return &dhcppb.GetSubnets4WithIpsResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GrpcService) GetSubnets6WithIps(ctx context.Context, req *dhcppb.GetSubnets6WithIpsRequest) (*dhcppb.GetSubnets6WithIpsResponse, error) {
	if subnets, err := GetDHCPService().GetSubnets6WithIps(req.GetIps()); err != nil {
		return &dhcppb.GetSubnets6WithIpsResponse{Succeed: false}, err
	} else {
		return &dhcppb.GetSubnets6WithIpsResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GrpcService) GetSubnet4AndLease4WithIp(ctx context.Context, req *dhcppb.GetSubnet4AndLease4WithIpRequest) (*dhcppb.GetSubnet4AndLease4WithIpResponse, error) {
	if ipv4Infos, err := GetDHCPService().GetSubnet4AndLease4WithIp(req.GetIp()); err != nil {
		return &dhcppb.GetSubnet4AndLease4WithIpResponse{Succeed: false}, err
	} else {
		return &dhcppb.GetSubnet4AndLease4WithIpResponse{
			Succeed:          true,
			Ipv4Informations: ipv4Infos,
		}, nil
	}
}

func (g *GrpcService) GetSubnet6AndLease6WithIp(ctx context.Context, req *dhcppb.GetSubnet6AndLease6WithIpRequest) (*dhcppb.GetSubnet6AndLease6WithIpResponse, error) {
	if ipv6Infos, err := GetDHCPService().GetSubnet6AndLease6WithIp(req.GetIp()); err != nil {
		return &dhcppb.GetSubnet6AndLease6WithIpResponse{Succeed: false}, err
	} else {
		return &dhcppb.GetSubnet6AndLease6WithIpResponse{
			Succeed:          true,
			Ipv6Informations: ipv6Infos,
		}, nil
	}
}

func (g *GrpcService) GetSubnets4AndLeases4WithIps(ctx context.Context, req *dhcppb.GetSubnets4AndLeases4WithIpsRequest) (*dhcppb.GetSubnets4AndLeases4WithIpsResponse, error) {
	if ipv4Infos, err := GetDHCPService().GetSubnets4AndLeases4WithIps(req.GetIps()); err != nil {
		return &dhcppb.GetSubnets4AndLeases4WithIpsResponse{Succeed: false}, err
	} else {
		return &dhcppb.GetSubnets4AndLeases4WithIpsResponse{
			Succeed:          true,
			Ipv4Informations: ipv4Infos,
		}, nil
	}
}

func (g *GrpcService) GetSubnets6AndLeases6WithIps(ctx context.Context, req *dhcppb.GetSubnets6AndLeases6WithIpsRequest) (*dhcppb.GetSubnets6AndLeases6WithIpsResponse, error) {
	if ipv6Infos, err := GetDHCPService().GetSubnets6AndLeases6WithIps(req.GetIps()); err != nil {
		return &dhcppb.GetSubnets6AndLeases6WithIpsResponse{Succeed: false}, err
	} else {
		return &dhcppb.GetSubnets6AndLeases6WithIpsResponse{
			Succeed:          true,
			Ipv6Informations: ipv6Infos,
		}, nil
	}
}

func (g *GrpcService) GetAllSubnet4S(ctx context.Context, request *dhcppb.GetSubnetsRequest) (*dhcppb.GetSubnet4SResponse, error) {
	if subnets, err := GetDHCPService().GetAllSubnet4s(); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetSubnet4SResponse{
			Subnet4S: subnets,
		}, nil
	}
}

func (g *GrpcService) GetAllSubnet6S(ctx context.Context, request *dhcppb.GetSubnetsRequest) (*dhcppb.GetSubnet6SResponse, error) {
	if subnets, err := GetDHCPService().GetAllSubnet6s(); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetSubnet6SResponse{
			Subnet6S: subnets,
		}, nil
	}
}

func (g *GrpcService) GetSubnet4SByPrefixes(ctx context.Context, request *dhcppb.GetSubnetsRequest) (*dhcppb.GetSubnet4SResponse, error) {
	if subnets, err := GetDHCPService().GetSubnet4sByPrefixes(request.GetPrefixes()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetSubnet4SResponse{
			Subnet4S: subnets,
		}, nil
	}
}

func (g *GrpcService) GetSubnet6SByPrefixes(ctx context.Context, request *dhcppb.GetSubnetsRequest) (*dhcppb.GetSubnet6SResponse, error) {
	if subnets, err := GetDHCPService().GetSubnet6sByPrefixes(request.GetPrefixes()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetSubnet6SResponse{
			Subnet6S: subnets,
		}, nil
	}
}

func (g *GrpcService) GetPool4SBySubnet(ctx context.Context, request *dhcppb.GetSubnetPoolsRequest) (*dhcppb.GetPool4SResponse, error) {
	if pools, err := GetDHCPService().GetPool4sBySubnet(request.GetSubnet()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetPool4SResponse{
			Pools: pools,
		}, nil
	}
}

func (g *GrpcService) GetPool6SBySubnet(ctx context.Context, request *dhcppb.GetSubnetPoolsRequest) (*dhcppb.GetPool6SResponse, error) {
	if pools, err := GetDHCPService().GetPool6sBySubnet(request.GetSubnet()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetPool6SResponse{
			Pools: pools,
		}, nil
	}
}

func (g *GrpcService) GetReservedPool4SBySubnet(ctx context.Context, request *dhcppb.GetSubnetPoolsRequest) (*dhcppb.GetReservedPool4SResponse, error) {
	if pools, err := GetDHCPService().GetReservedPool4sBySubnet(request.GetSubnet()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetReservedPool4SResponse{
			Pools: pools,
		}, nil
	}
}

func (g *GrpcService) GetReservedPool6SBySubnet(ctx context.Context, request *dhcppb.GetSubnetPoolsRequest) (*dhcppb.GetReservedPool6SResponse, error) {
	if pools, err := GetDHCPService().GetReservedPool6sBySubnet(request.GetSubnet()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetReservedPool6SResponse{
			Pools: pools,
		}, nil
	}
}

func (g *GrpcService) GetReservation4SBySubnet(ctx context.Context, request *dhcppb.GetSubnetPoolsRequest) (*dhcppb.GetReservationPool4SResponse, error) {
	if pools, err := GetDHCPService().GetReservation4sBySubnet(request.GetSubnet()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetReservationPool4SResponse{
			Pools: pools,
		}, nil
	}
}

func (g *GrpcService) GetReservation6SBySubnet(ctx context.Context, request *dhcppb.GetSubnetPoolsRequest) (*dhcppb.GetReservationPool6SResponse, error) {
	if pools, err := GetDHCPService().GetReservation6sBySubnet(request.GetSubnet()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetReservationPool6SResponse{
			Pools: pools,
		}, nil
	}
}

func (g *GrpcService) GetPdPools6SBySubnet(ctx context.Context, request *dhcppb.GetSubnetPoolsRequest) (*dhcppb.GetPdPoolsBySubnetResponse, error) {
	if pools, err := GetDHCPService().GetPdPool6sBySubnet(request.GetSubnet()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetPdPoolsBySubnetResponse{
			PdPool6S: pools,
		}, nil
	}
}

func (g *GrpcService) GetLease4ByIp(ctx context.Context, request *dhcppb.GetLeaseByIpRequest) (*dhcppb.GetLease4ByIpResponse, error) {
	if lease4, err := GetDHCPService().GetLease4ByIp(request.GetIp()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetLease4ByIpResponse{
			Lease4: lease4,
		}, nil
	}
}

func (g *GrpcService) GetLease6ByIp(ctx context.Context, request *dhcppb.GetLeaseByIpRequest) (*dhcppb.GetLease6ByIpResponse, error) {
	if lease6, err := GetDHCPService().GetLease6ByIp(request.GetIp()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetLease6ByIpResponse{
			Lease6: lease6,
		}, nil
	}
}

func (g *GrpcService) GetLease4SBySubnet(ctx context.Context, request *dhcppb.GetLeasesBySubnetRequest) (*dhcppb.GetLease4SBySubnetResponse, error) {
	if lease4s, err := GetDHCPService().GetLease4ByPrefix(request.GetSubnet()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetLease4SBySubnetResponse{
			Lease4S: lease4s,
		}, nil
	}
}

func (g *GrpcService) GetLease6SBySubnet(ctx context.Context, request *dhcppb.GetLeasesBySubnetRequest) (*dhcppb.GetLease6SBySubnetResponse, error) {
	if lease6s, err := GetDHCPService().GetLease6sBySubnet(request.GetSubnet()); err != nil {
		return nil, err
	} else {
		return &dhcppb.GetLease6SBySubnetResponse{
			Lease6S: lease6s,
		}, nil
	}
}

func (g *GrpcService) CreateReservation4S(ctx context.Context, request *dhcppb.CreateReservation4SRequest) (*dhcppb.CreateReservation4SResponse, error) {
	if err := GetDHCPService().CreateReservation4s(request.GetSubnet(),
		request.GetReservation4S()); err != nil {
		return nil, err
	} else {
		return &dhcppb.CreateReservation4SResponse{Succeed: true}, nil
	}
}

func (g *GrpcService) CreateReservedPool4S(ctx context.Context, request *dhcppb.CreateReservedPool4SRequest) (*dhcppb.CreateReservedPool4SResponse, error) {
	if err := GetDHCPService().CreateReservedPool4s(request.GetSubnet(),
		request.GetReservedPool4S()); err != nil {
		return nil, err
	} else {
		return &dhcppb.CreateReservedPool4SResponse{Succeed: true}, nil
	}
}

func (g *GrpcService) CreateReservation6S(ctx context.Context, request *dhcppb.CreateReservation6SRequest) (*dhcppb.CreateReservation6SResponse, error) {
	if err := GetDHCPService().CreateReservation6s(request.GetSubnet(),
		request.GetReservation6S()); err != nil {
		return nil, err
	} else {
		return &dhcppb.CreateReservation6SResponse{Succeed: true}, nil
	}
}

func (g *GrpcService) CreateReservedPool6S(ctx context.Context, request *dhcppb.CreateReservedPool6SRequest) (*dhcppb.CreateReservedPool6SResponse, error) {
	if err := GetDHCPService().CreateReservedPool6s(request.GetSubnet(),
		request.GetReservedPool6S()); err != nil {
		return nil, err
	} else {
		return &dhcppb.CreateReservedPool6SResponse{Succeed: true}, nil
	}
}
