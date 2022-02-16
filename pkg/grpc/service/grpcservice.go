package service

import (
	"context"
	"fmt"

	"github.com/linkingthing/clxone-dhcp/pkg/grpc/parser"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
)

type GRPCService struct {
}

func NewGRPCService() *GRPCService {
	return &GRPCService{}
}

func (g *GRPCService) GetSubnet4WithIp(ctx context.Context, req *pbdhcp.GetSubnet4WithIpRequest) (*pbdhcp.GetSubnet4WithIpResponse, error) {
	if subnets, err := GetDHCPService().GetSubnet4WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet4WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet4WithIpResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GRPCService) GetSubnet6WithIp(ctx context.Context, req *pbdhcp.GetSubnet6WithIpRequest) (*pbdhcp.GetSubnet6WithIpResponse, error) {
	if subnets, err := GetDHCPService().GetSubnet6WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet6WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet6WithIpResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GRPCService) GetSubnets4WithIps(ctx context.Context, req *pbdhcp.GetSubnets4WithIpsRequest) (*pbdhcp.GetSubnets4WithIpsResponse, error) {
	if subnets, err := GetDHCPService().GetSubnets4WithIps(req.GetIps()); err != nil {
		return &pbdhcp.GetSubnets4WithIpsResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnets4WithIpsResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GRPCService) GetSubnets6WithIps(ctx context.Context, req *pbdhcp.GetSubnets6WithIpsRequest) (*pbdhcp.GetSubnets6WithIpsResponse, error) {
	if subnets, err := GetDHCPService().GetSubnets6WithIps(req.GetIps()); err != nil {
		return &pbdhcp.GetSubnets6WithIpsResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnets6WithIpsResponse{Succeed: true, Subnets: subnets}, nil
	}
}

func (g *GRPCService) GetSubnet4AndLease4WithIp(ctx context.Context, req *pbdhcp.GetSubnet4AndLease4WithIpRequest) (*pbdhcp.GetSubnet4AndLease4WithIpResponse, error) {
	if ipv4Infos, err := GetDHCPService().GetSubnet4AndLease4WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet4AndLease4WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet4AndLease4WithIpResponse{
			Succeed:          true,
			Ipv4Informations: ipv4Infos,
		}, nil
	}
}

func (g *GRPCService) GetSubnet6AndLease6WithIp(ctx context.Context, req *pbdhcp.GetSubnet6AndLease6WithIpRequest) (*pbdhcp.GetSubnet6AndLease6WithIpResponse, error) {
	if ipv6Infos, err := GetDHCPService().GetSubnet6AndLease6WithIp(req.GetIp()); err != nil {
		return &pbdhcp.GetSubnet6AndLease6WithIpResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnet6AndLease6WithIpResponse{
			Succeed:          true,
			Ipv6Informations: ipv6Infos,
		}, nil
	}
}

func (g *GRPCService) GetSubnets4AndLeases4WithIps(ctx context.Context, req *pbdhcp.GetSubnets4AndLeases4WithIpsRequest) (*pbdhcp.GetSubnets4AndLeases4WithIpsResponse, error) {
	if ipv4Infos, err := GetDHCPService().GetSubnets4AndLeases4WithIps(req.GetIps()); err != nil {
		return &pbdhcp.GetSubnets4AndLeases4WithIpsResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnets4AndLeases4WithIpsResponse{
			Succeed:          true,
			Ipv4Informations: ipv4Infos,
		}, nil
	}
}

func (g *GRPCService) GetSubnets6AndLeases6WithIps(ctx context.Context, req *pbdhcp.GetSubnets6AndLeases6WithIpsRequest) (*pbdhcp.GetSubnets6AndLeases6WithIpsResponse, error) {
	if ipv6Infos, err := GetDHCPService().GetSubnets6AndLeases6WithIps(req.GetIps()); err != nil {
		return &pbdhcp.GetSubnets6AndLeases6WithIpsResponse{Succeed: false}, err
	} else {
		return &pbdhcp.GetSubnets6AndLeases6WithIpsResponse{
			Succeed:          true,
			Ipv6Informations: ipv6Infos,
		}, nil
	}
}

////
func (g *GRPCService) GetListAllSubnet4(ctx context.Context, req *pbdhcp.GetSubnet4Request) (*pbdhcp.GetSubnet4Response, error) {
	if len(req.Prefixes) != 0 {
		return nil, fmt.Errorf("getlistallsubnet4 prefixes is not nil ")
	}
	if subnetList, err := GetDHCPService().GetAllSubnet4s(); err != nil {
		return nil, fmt.Errorf("get subnet4 all failed : %s", err.Error())
	} else {
		return &pbdhcp.GetSubnet4Response{Subnet4S: subnetList}, nil
	}
}

func (g *GRPCService) GetListSubnet4ByPrefixes(ctx context.Context, req *pbdhcp.GetSubnet4Request) (*pbdhcp.GetSubnet4Response, error) {
	if len(req.GetPrefixes()) == 0 {
		return nil, fmt.Errorf("getlistsubnet4byprefixes prefixes is nil ")
	}
	if subnetList, err := GetDHCPService().GetWithSubnet4sByPrefixes(req.GetPrefixes()); err != nil {
		return nil, err
	} else {
		return &pbdhcp.GetSubnet4Response{Subnet4S: subnetList}, nil
	}
}

func (g *GRPCService) GetListPool4BySubnet4(ctx context.Context,
	req *pbdhcp.GetPool4BySubnet4Request) (*pbdhcp.GetPool4BySubnet4Response, error) {
	subnet := parser.DecodePbSubnet4(req.GetSubnet())
	if subnetList, err := GetDHCPService().GetPool4List(subnet); err != nil {
		return nil, fmt.Errorf("get list pool4 by subnet4 failed :%s", err.Error())
	} else {
		return &pbdhcp.GetPool4BySubnet4Response{Pool4S: subnetList}, nil
	}
}

func (g *GRPCService) GetListReservedPool4BySubnet4Id(ctx context.Context,
	req *pbdhcp.GetReservedPool4BySubnet4IdRequest) (*pbdhcp.GetReservedPool4BySubnet4IdResponse, error) {
	if tmpList, err := GetDHCPService().GetReservedPool4List(req.GetSubnet4Id()); err != nil {
		return nil, fmt.Errorf("getreservedpool4list failed :%s", err.Error())
	} else {
		return &pbdhcp.GetReservedPool4BySubnet4IdResponse{ReservedPool4S: tmpList}, nil
	}
}

func (g *GRPCService) GetListReservation4BySubnet4Id(ctx context.Context,
	req *pbdhcp.GetReservation4BySubnet4IdRequest) (*pbdhcp.GetReservation4BySubnet4IdResponse, error) {
	if tmpList, err := GetDHCPService().GetReservation4List(req.GetSubnet4Id()); err != nil {
		return nil, fmt.Errorf("GetReservation4List failed :%s", err.Error())
	} else {
		return &pbdhcp.GetReservation4BySubnet4IdResponse{Reservation4S: tmpList}, nil
	}
}

func (g *GRPCService) GetListLease4BySubnet4Id(ctx context.Context,
	req *pbdhcp.GetLease4BySubnet4IdRequest) (*pbdhcp.GetLease4BySubnet4IdResponse, error) {
	if tmpList, err := GetDHCPService().GetSubnetLease4List(req.GetSubnet4Id()); err != nil {
		return nil, fmt.Errorf("getlistlease4bysubnet4id failed: %s", err.Error())
	} else {
		return &pbdhcp.GetLease4BySubnet4IdResponse{SubnetLease4S: tmpList}, nil
	}
}

func (g *GRPCService) GetListLease4ByIp(ctx context.Context,
	req *pbdhcp.GetLease4ByIpRequest) (*pbdhcp.GetLease4ByIpResponse, error) {
	if tmpList, err := GetDHCPService().GetSubnetLease4ByIp(req.GetSubnet4Id(), req.GetIp()); err != nil {
		return nil, fmt.Errorf("getlistlease4byip failed: %s", err.Error())
	} else {
		return &pbdhcp.GetLease4ByIpResponse{SubnetLease4S: tmpList}, nil
	}
}

func (g *GRPCService) GetListAllSubnet6(ctx context.Context, req *pbdhcp.GetSubnet6Request) (*pbdhcp.GetSubnet6Response, error) {
	if len(req.Prefixes) != 0 {
		return nil, fmt.Errorf("getlistallsubnet6 prefixes is not nil ")
	}
	if tmpList, err := GetDHCPService().GetAllSubnet6(); err != nil {
		return nil, fmt.Errorf("getlistallsubnet6 failed :%s", err.Error())
	} else {
		return &pbdhcp.GetSubnet6Response{Subnet6S: tmpList}, nil
	}
}

func (g *GRPCService) GetListSubnet6ByPrefixes(ctx context.Context, req *pbdhcp.GetSubnet6Request) (*pbdhcp.GetSubnet6Response, error) {
	if len(req.Prefixes) == 0 {
		return nil, fmt.Errorf("getlistsubnet6byprefixes prefixes is nil ")
	}
	if tmpList, err := GetDHCPService().GetWithSubnet6ByPrefixes(req.GetPrefixes()); err != nil {
		return nil, fmt.Errorf("getlistsubnet6byprefixes failed :%s", err.Error())
	} else {
		return &pbdhcp.GetSubnet6Response{Subnet6S: tmpList}, nil
	}
}

func (g *GRPCService) GetListPool6BySubnet6Id(ctx context.Context,
	req *pbdhcp.GetPool6BySubnet6IdRequest) (*pbdhcp.GetPool6BySubnet6IdResponse, error) {
	subnet := parser.DecodePbSubnet6(req.GetSubnet())
	if tmpList, err := GetDHCPService().GetPool6List(subnet); err != nil {
		return nil, fmt.Errorf("getlistpool6bysubnet6id failed :%s", err.Error())
	} else {
		return &pbdhcp.GetPool6BySubnet6IdResponse{Pool6S: tmpList}, nil
	}
}

func (g *GRPCService) GetListReservedPool6BySubnet6Id(ctx context.Context,
	req *pbdhcp.GetReservedPool6BySubnet6IdRequest) (*pbdhcp.GetReservedPool6BySubnet6IdResponse, error) {
	subnet := parser.DecodePbSubnet6(req.GetSubnet())
	if tmpList, err := GetDHCPService().GetReservedPool6List(subnet); err != nil {
		return nil, fmt.Errorf("getlistreservedpool6bysubnet6id failed :%s", err.Error())
	} else {
		return &pbdhcp.GetReservedPool6BySubnet6IdResponse{ReservedPool6S: tmpList}, nil
	}
}

func (g *GRPCService) GetListReservation6BySubnet6Id(ctx context.Context,
	req *pbdhcp.GetReservation6BySubnet6IdRequest) (*pbdhcp.GetReservation6BySubnet6IdResponse, error) {
	if tmpList, err := GetDHCPService().GetReservation6List(req.GetSubnet6Id()); err != nil {
		return nil, fmt.Errorf("getlistreservation6bysubnet6id failed :%s", err.Error())
	} else {
		return &pbdhcp.GetReservation6BySubnet6IdResponse{Reservation6S: tmpList}, nil
	}
}

func (g *GRPCService) GetListLease6BySubnet6Id(ctx context.Context,
	req *pbdhcp.GetLease6BySubnet6IdRequest) (*pbdhcp.GetLease6BySubnet6IdResponse, error) {
	if tmpList, err := GetDHCPService().GetSLease6ListBySubnetId(req.GetSubnet6Id()); err != nil {
		return nil, fmt.Errorf("getlistlease6bysubnet6id failed :%s", err.Error())
	} else {
		return &pbdhcp.GetLease6BySubnet6IdResponse{SubnetLease6: tmpList}, nil
	}
}

func (g *GRPCService) GetListLease6ByIp(ctx context.Context,
	req *pbdhcp.GetLease6ByIpRequest) (*pbdhcp.GetLease6ByIpResponse, error) {
	if tmpValue, err := GetDHCPService().GetSubnetLease6ByIp(req.GetSubnet6Id(), req.GetIp()); err != nil {
		return nil, fmt.Errorf("getlistlease6byip failed :%s", err.Error())
	} else {
		return &pbdhcp.GetLease6ByIpResponse{SubnetLease6: tmpValue}, nil
	}
}

////
func (g *GRPCService) CreateReservation4S(ctx context.Context,
	req *pbdhcp.CreateReservation4SRequest) (*pbdhcp.CreateReservation4SResponse, error) {

	reservation := parser.DecodePbReservation4(req.GetReservation())
	if err := reservation.Validate(); err != nil {
		return nil, fmt.Errorf("create reservation4 params invalid: %s", err.Error())
	}
	if tmpValue, err := GetDHCPService().CreateReservation4s(req.GetParentId(), reservation); err != nil {
		return nil, fmt.Errorf("create reservation4 failed: %s", err.Error())
	} else {
		return &pbdhcp.CreateReservation4SResponse{Succeed: tmpValue}, nil
	}
}

func (g *GRPCService) CreateReservedPool4(ctx context.Context,
	req *pbdhcp.CreateReservedPool4Request) (*pbdhcp.CreateReservedPool4Response, error) {
	subnet := parser.DecodePbSubnet4(req.GetParentSubnet())
	pool := parser.DecodePbReservedPool4(req.GetReservedPool())
	if err := pool.Validate(); err != nil {
		return nil, fmt.Errorf("create reserved4 pool params invalid: %s", err.Error())
	}
	if tmpValue, err := GetDHCPService().CreateReservedPool4(subnet, pool); err != nil {
		return nil, fmt.Errorf("create reserved4 pool failed: %s", err.Error())
	} else {
		return &pbdhcp.CreateReservedPool4Response{Succeed: tmpValue}, nil
	}
}

func (g *GRPCService) CreateReservation6S(ctx context.Context,
	req *pbdhcp.CreateReservation6SRequest) (*pbdhcp.CreateReservation6SResponse, error) {
	reservation := parser.DecodePbReservation6(req.GetReservation())
	if err := reservation.Validate(); err != nil {
		return nil, fmt.Errorf("create reservation6 params invalid: %s", err.Error())
	}
	if tmpValue, err := GetDHCPService().CreateReservation6s(req.GetParentId(), reservation); err != nil {
		return nil, fmt.Errorf("create reservation6 failed: %s", err.Error())
	} else {
		return &pbdhcp.CreateReservation6SResponse{Succeed: tmpValue}, nil
	}
}

func (g *GRPCService) CreateReservedPool6(ctx context.Context,
	req *pbdhcp.CreateReservedPool6Request) (*pbdhcp.CreateReservedPool6Response, error) {
	subnet := parser.DecodePbSubnet6(req.GetParentSubnet())
	pool := parser.DecodePbReservedPool6(req.GetReservedPool())
	if err := pool.Validate(); err != nil {
		return nil, fmt.Errorf("create reserved6 pool params invalid: %s", err.Error())
	}
	if tmpValue, err := GetDHCPService().CreateReservedPool6(subnet, pool); err != nil {
		return nil, fmt.Errorf("create reserved6 pool failed: %s", err.Error())
	} else {
		return &pbdhcp.CreateReservedPool6Response{Succeed: tmpValue}, nil
	}
}
