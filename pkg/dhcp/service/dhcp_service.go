package service

import (
	"context"
	"errors"
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
)

const (
	GetSubnetsWithIdSql = "select * from gr_subnet where id in (%s)"
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
			err = tx.FillEx(&subnets, fmt.Sprintf(GetSubnetsWithIdSql, subnetIndex), subnetAgrs...)
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
			err = tx.FillEx(&subnets, fmt.Sprintf(GetSubnetsWithIdSql, subnetIndex), subnetAgrs...)
		} else {
			err = tx.Fill(nil, &subnets)
		}
		return err

	})
	return
}

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

func (a *DHCPService) GetClosestSubnet4ByIDs(ids []string, ip string) (*resource.Subnet4, error) {
	subnets, err := a.GetSubnet4ByIDs(ids...)
	if err != nil {
		return nil, err
	}

	return getClosestSubnet4(subnets, net.ParseIP(ip))
}

func getClosestSubnet4(subnets []*resource.Subnet4, ip net.IP) (subnet *resource.Subnet4, err error) {
	var maxPrefixLen int
	for _, subnet_ := range subnets {
		if subnet_.Ipnet.Contains(ip) {
			if ones, _ := subnet_.Ipnet.Mask.Size(); ones > maxPrefixLen {
				subnet = subnet_
				maxPrefixLen = ones
			}
		}
	}

	if subnet == nil {
		err = errors.New("can not find subnet")
	}

	return
}

func (a *DHCPService) GetClosestSubnet6ByIDs(ids []string, ip string) (*resource.Subnet6, error) {
	subnets, err := a.GetSubnet6ByIDs(ids...)
	if err != nil {
		return nil, err
	}

	return getClosestSubnet6(subnets, net.ParseIP(ip))
}

func getClosestSubnet6(subnets []*resource.Subnet6, ip net.IP) (subnet *resource.Subnet6, err error) {
	var maxPrefixLen int
	for _, subnet_ := range subnets {
		if subnet_.Ipnet.Contains(ip) {
			if ones, _ := subnet_.Ipnet.Mask.Size(); ones > maxPrefixLen {
				subnet = subnet_
				maxPrefixLen = ones
			}
		}
	}

	if subnet == nil {
		err = errors.New("can not find subnet")
	}

	return
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
