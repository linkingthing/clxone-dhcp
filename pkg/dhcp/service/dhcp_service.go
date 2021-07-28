package service

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"

	"github.com/linkingthing/clxone-dhcp/config"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	metricresource "github.com/linkingthing/clxone-dhcp/pkg/metric/resource"
	pb "github.com/linkingthing/clxone-dhcp/pkg/proto"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
)

const (
	GetSubnet4sWithIdSql = "select * from gr_subnet4 where id in (%s)"
	GetSubnet6sWithIdSql = "select * from gr_subnet6 where id in (%s)"
)

var globalDHCPService *DHCPService
var onceDHCPService sync.Once

type DHCPService struct {
}

func GetDHCPService() *DHCPService {
	onceDHCPService.Do(func() {
		globalDHCPService = &DHCPService{}
	})
	return globalDHCPService
}

func (a *DHCPService) GetSubnet4ByIDs(ids ...string) (subnets []*resource.Subnet4, err error) {
	err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if len(ids) > 0 {
			subnetIndex, subnetAgrs := genSqlArgsAndIndex(ids)
			err = tx.FillEx(&subnets, fmt.Sprintf(GetSubnet4sWithIdSql, subnetIndex), subnetAgrs...)
		} else {
			err = tx.Fill(nil, &subnets)
		}
		return err
	})
	return
}

func (a *DHCPService) GetSubnet6ByIDs(ids ...string) (subnets []*resource.Subnet6, err error) {
	err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if len(ids) > 0 {
			subnetIndex, subnetAgrs := genSqlArgsAndIndex(ids)
			err = tx.FillEx(&subnets, fmt.Sprintf(GetSubnet6sWithIdSql, subnetIndex), subnetAgrs...)
		} else {
			err = tx.Fill(nil, &subnets)
		}
		return err

	})
	return
}

//TODO remove it
func (a *DHCPService) GetNodeList() (nodes []*metricresource.Node, err error) {
	endpoints, err := pb.GetEndpoints(config.GetConfig().CallServices.DhcpAgent)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("found clxone-dhcp-agnet-grpc: %s", err.Error()))
	}

	for _, end := range endpoints {
		response, err := end(context.Background(), struct{}{})
		if err != nil {
			return nil, err
		}

		ip, _, err := net.SplitHostPort(response.(string))
		if err != nil {
			return nil, err
		}

		node := &metricresource.Node{
			Ip:       ip,
			Hostname: ip,
		}
		node.SetID(ip)
		nodes = append(nodes, node)
	}

	return
}

func (a *DHCPService) GetClosestSubnet4ByIDs(ids []string, ip string) (*pbdhcp.Subnet, error) {
	subnets, err := a.GetSubnet4ByIDs(ids...)
	if err != nil {
		return nil, err
	}

	return getClosestSubnet4(subnets, net.ParseIP(ip))
}

func getClosestSubnet4(subnets []*resource.Subnet4, ip net.IP) (*pbdhcp.Subnet, error) {
	var maxPrefixLen int
	var subnet4 *resource.Subnet4
	for _, subnet := range subnets {
		if subnet.Ipnet.Contains(ip) {
			if ones, _ := subnet.Ipnet.Mask.Size(); ones > maxPrefixLen {
				subnet4 = subnet
				maxPrefixLen = ones
			}
		}
	}

	if subnet4 == nil {
		return nil, fmt.Errorf("no find subnet with ip %s", ip.String())
	}

	return &pbdhcp.Subnet{
		Id:          subnet4.ID,
		Subnet:      subnet4.Subnet,
		SubnetId:    uint32(subnet4.SubnetId),
		Tags:        subnet4.Tags,
		NetworkType: subnet4.NetworkType,
		Capacity:    subnet4.Capacity,
		UsedRatio:   subnet4.UsedRatio,
		UsedCount:   subnet4.UsedCount,
	}, nil
}

func (a *DHCPService) GetClosestSubnet6ByIDs(ids []string, ip string) (*pbdhcp.Subnet, error) {
	subnets, err := a.GetSubnet6ByIDs(ids...)
	if err != nil {
		return nil, err
	}

	return getClosestSubnet6(subnets, net.ParseIP(ip))
}

func getClosestSubnet6(subnets []*resource.Subnet6, ip net.IP) (*pbdhcp.Subnet, error) {
	var maxPrefixLen int
	var subnet6 *resource.Subnet6
	for _, subnet := range subnets {
		if subnet.Ipnet.Contains(ip) {
			if ones, _ := subnet.Ipnet.Mask.Size(); ones > maxPrefixLen {
				subnet6 = subnet
				maxPrefixLen = ones
			}
		}
	}

	if subnet6 == nil {
		return nil, fmt.Errorf("no find subnet with ip %s", ip.String())
	}

	return &pbdhcp.Subnet{
		Id:          subnet6.ID,
		Subnet:      subnet6.Subnet,
		SubnetId:    uint32(subnet6.SubnetId),
		Tags:        subnet6.Tags,
		NetworkType: subnet6.NetworkType,
		Capacity:    subnet6.Capacity,
		UsedRatio:   subnet6.UsedRatio,
		UsedCount:   subnet6.UsedCount,
	}, nil
}

func genSqlArgsAndIndex(args []string) (string, []interface{}) {
	var indexes []string
	var sqlAgrs []interface{}
	for i, arg := range args {
		indexes = append(indexes, "$"+strconv.Itoa(i+1))
		sqlAgrs = append(sqlAgrs, arg)
	}

	return strings.Join(indexes, ","), sqlAgrs
}
