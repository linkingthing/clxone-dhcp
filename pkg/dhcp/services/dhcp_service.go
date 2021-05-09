package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	restdb "github.com/zdnscloud/gorest/db"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
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

func (a *DHCPService) GetSubnetByID(subnetID string) (subnet *resource.Subnet, err error) {
	var subnets []*resource.Subnet
	subnetInterface, err := restdb.GetResourceWithID(db.GetDB(), subnetID, &subnets)
	if err != nil {
		logrus.Error(err)
		return
	}

	subnet = subnetInterface.(*resource.Subnet)
	if err = setSubnetLeasesUsedRatio(subnet); err != nil {
		logrus.Error(err)
		return
	}

	return subnet, nil
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

	var resp *pb.GetLeasesCountResponse
	var err error
	if subnet.Version == util.IPVersion4 {
		resp, err = grpcclient.GetDHCPGrpcClient().GetSubnet4LeasesCount(context.TODO(),
			&pb.GetSubnet4LeasesCountRequest{Id: subnet.SubnetId})
	} else {
		resp, err = grpcclient.GetDHCPGrpcClient().GetSubnet6LeasesCount(context.TODO(),
			&pb.GetSubnet6LeasesCountRequest{Id: subnet.SubnetId})
	}

	return resp.GetLeasesCount(), err
}
