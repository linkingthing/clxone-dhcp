package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
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
	if err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if len(ids) > 0 {
			subnetIndex, subnetAgrs := genSqlArgsAndIndex(ids)
			err = tx.FillEx(&subnets, fmt.Sprintf(GetSubnetsWithIdSql, subnetIndex), subnetAgrs...)
		} else {
			err = tx.Fill(nil, &subnets)
		}
		return err

	}); err != nil {
		logrus.Error(err)
	}
	return
}

func (a *DHCPService) GetSubnet6ByIDs(ids ...string) (subnets []*resource.Subnet6, err error) {
	if err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if len(ids) > 0 {
			subnetIndex, subnetAgrs := genSqlArgsAndIndex(ids)
			err = tx.FillEx(&subnets, fmt.Sprintf(GetSubnetsWithIdSql, subnetIndex), subnetAgrs...)
		} else {
			err = tx.Fill(nil, &subnets)
		}
		return err

	}); err != nil {
		logrus.Error(err)
	}
	return
}

func (a *DHCPService) GetNodeList() (nodes []*metricresource.Node, err error) {
	endpoints, err := pb.GetEndpoints(config.GetConfig().CallServices.DhcpAgent)
	if err != nil {
		logrus.Error(err)
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("found clxone-dhcp-agnet-grpc: %s", err.Error()))
	}
	for _, end := range endpoints {
		response, err := end(context.Background(), struct{}{})
		if err != nil {
			logrus.Error(err)
			return nil, err
		}

		if err != nil {
			log.Printf("did not connect: %v", err)
			return nil, err
		}

		ip := strings.Split(response.(string), ":")[0]
		node := &metricresource.Node{
			Ip:       ip,
			HostName: ip,
		}
		node.SetID(ip)

		nodes = append(nodes, node)
	}

	return
}

func (a *DHCPService) GetClosestSubnet4ByIDs(ids []string, ip string) (subnet *resource.Subnet4, err error) {
	subnets, err := a.GetSubnet4ByIDs(ids...)
	if err != nil {
		logrus.Error(err)
		return
	}

	subnet, err = getClosestSubnet4(subnets, net.ParseIP(ip))

	if err != nil {
		logrus.Error(err)
	}

	return
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

func (a *DHCPService) GetClosestSubnet6ByIDs(ids []string, ip string) (subnet *resource.Subnet6, err error) {
	subnets, err := a.GetSubnet6ByIDs(ids...)
	if err != nil {
		logrus.Error(err)
		return
	}

	subnet, err = getClosestSubnet6(subnets, net.ParseIP(ip))
	if err != nil {
		logrus.Error(err)
	}

	return
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
