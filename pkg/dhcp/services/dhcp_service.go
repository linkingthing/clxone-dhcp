package services

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	restdb "github.com/zdnscloud/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	dhcp_agent "github.com/linkingthing/clxone-dhcp/pkg/pb/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

const (
	GetSubnetsWithIdSql = "select * from gr_subnet where id in (%s)"
)

var globalDHCPService *DHCPService
var onceDHCPService sync.Once

type DHCPService struct {
}

func NewDHCPService() *DHCPService {
	onceDHCPService.Do(func() {
		globalDHCPService = &DHCPService{}
	})
	return globalDHCPService
}

func (a *DHCPService) GetSubnetByIDs(ids ...string) (subnets []*resource.Subnet, err error) {
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

func (a *DHCPService) GetClosestSubnetByIDs(ids []string, ip string) (subnet *resource.Subnet, err error) {
	subnets, err := a.GetSubnetByIDs(ids...)
	if err != nil {
		logrus.Error(err)
		return
	}

	subnet, err = getClosestSubnet(subnets, net.ParseIP(ip))

	if err != nil {
		logrus.Error(err)
		return
	}
	return
}

func getClosestSubnet(subnets []*resource.Subnet, ip net.IP) (subnet *resource.Subnet, err error) {
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

func setSubnetLeasesUsedRatio(subnet *resource.Subnet) error {
	leasesCount, err := getSubnetLeasesCount(subnet)
	if err != nil {
		return err
	}

	if leasesCount != 0 {
		subnet.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(subnet.Capacity))
	}
	return nil
}

func getSubnetLeasesCount(subnet *resource.Subnet) (uint64, error) {
	if subnet.Capacity == 0 {
		return 0, nil
	}

	var resp *dhcp_agent.GetLeasesCountResponse
	var err error
	if subnet.Version == util.IPVersion4 {
		resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnet4LeasesCount(context.TODO(),
			&dhcp_agent.GetSubnet4LeasesCountRequest{Id: subnet.SubnetId})
	} else {
		resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnet6LeasesCount(context.TODO(),
			&dhcp_agent.GetSubnet6LeasesCountRequest{Id: subnet.SubnetId})
	}

	return resp.GetLeasesCount(), err
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
