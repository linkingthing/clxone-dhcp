package api

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/services"
	dhcp_agent "github.com/linkingthing/clxone-dhcp/pkg/pb/dhcp-agent"

	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type SubnetHandler struct {
}

func NewSubnetHandler() *SubnetHandler {
	return &SubnetHandler{}
}

func (s *SubnetHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet)
	if err := subnet.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create subnet params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var subnets []*resource.Subnet
		if err := tx.Fill(map[string]interface{}{"orderby": "subnet_id desc"},
			&subnets); err != nil {
			return fmt.Errorf("get max subnet id from db failed: %s\n", err.Error())
		}

		subnet.SubnetId = 1
		if len(subnets) > 0 {
			subnet.SubnetId = subnets[0].SubnetId + 1
		}

		subnet.SetID(strconv.Itoa(int(subnet.SubnetId)))
		if err := s.checkSubnetAvailable(subnet, subnets, tx); err != nil {
			return err
		}

		if _, err := tx.Insert(subnet); err != nil {
			return err
		}

		return sendCreateSubnetCmdToDDIAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create subnet %s failed: %s", subnet.Subnet, err.Error()))
	}

	return subnet, nil
}

func sendCreateSubnetCmdToDDIAgent(subnet *resource.Subnet) error {
	cmd := services.CreateSubnet4
	var req []byte
	var err error
	if subnet.Version == util.IPVersion4 {
		req, err = proto.Marshal(&dhcp_agent.CreateSubnet4Request{
			Id:                  subnet.SubnetId,
			Ipnet:               subnet.Subnet,
			ValidLifetime:       subnet.ValidLifetime,
			MaxValidLifetime:    subnet.MaxValidLifetime,
			MinValidLifetime:    subnet.MinValidLifetime,
			DomainServers:       subnet.DomainServers,
			Routers:             subnet.Routers,
			ClientClass:         subnet.ClientClass,
			IfaceName:           subnet.IfaceName,
			RelayAgentAddresses: subnet.RelayAgentAddresses,
		})
	} else {
		cmd = services.CreateSubnet6
		req, err = proto.Marshal(&dhcp_agent.CreateSubnet6Request{
			Id:                    subnet.SubnetId,
			Ipnet:                 subnet.Subnet,
			ValidLifetime:         subnet.ValidLifetime,
			MaxValidLifetime:      subnet.MaxValidLifetime,
			MinValidLifetime:      subnet.MinValidLifetime,
			DnsServers:            subnet.DomainServers,
			IfaceName:             subnet.IfaceName,
			RelayAgentAddresses:   subnet.RelayAgentAddresses,
			RelayAgentInterfaceId: subnet.RelayAgentInterfaceId,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal create subnet request failed: %s", err.Error())
	}

	// return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
	return services.NewDHCPAgentService().SendDHCPCmd(cmd, req)
}

func (s *SubnetHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	conditions := map[string]interface{}{"orderby": "subnet_id"}
	if version, ok := util.IPVersionFromFilter(ctx.GetFilters()); ok == false {
		return nil, nil
	} else {
		conditions[util.FilterNameVersion] = version
	}

	if subnet, ok := util.GetFilterValueWithEqModifierFromFilters(util.FileNameSubnet, ctx.GetFilters()); ok {
		conditions[util.FileNameSubnet] = subnet
	}

	var subnets resource.Subnets
	if err := db.GetResources(conditions, &subnets); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnets from db failed: %s", err.Error()))
	}

	subnetsLeasesCount, err := resource.GetSubnetsLeasesCount()
	if err != nil {
		log.Warnf("get subnets leases count failed: %s", err.Error())
	}

	var visible resource.Subnets
	for _, subnet := range subnets {
		resource.SetSubnetLeasesUsedRatioWithLeasesCount(subnet, subnetsLeasesCount)
		visible = append(visible, subnet)
	}

	sort.Sort(visible)
	return visible, nil
}

func (s *SubnetHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	var subnets []*resource.Subnet
	subnetInterface, err := restdb.GetResourceWithID(db.GetDB(), subnetID, &subnets)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet %s from db failed: %s", subnetID, err.Error()))
	}

	subnet := subnetInterface.(*resource.Subnet)
	if err := setSubnetLeasesUsedRatio(subnet); err != nil {
		log.Warnf("get subnet %s leases used ratio failed: %s", subnetID, err.Error())
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

	var resp *dhcp_agent.GetLeasesCountResponse
	var err error
	if subnet.Version == util.IPVersion4 {
		resp, err = grpcclient.GetDHCPGrpcClient().GetSubnet4LeasesCount(context.TODO(),
			&dhcp_agent.GetSubnet4LeasesCountRequest{Id: subnet.SubnetId})
	} else {
		resp, err = grpcclient.GetDHCPGrpcClient().GetSubnet6LeasesCount(context.TODO(),
			&dhcp_agent.GetSubnet6LeasesCountRequest{Id: subnet.SubnetId})
	}

	return resp.GetLeasesCount(), err
}

func (s *SubnetHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet)
	if err := subnet.ValidateParams(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update subnet params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet, map[string]interface{}{
			"valid_lifetime":           subnet.ValidLifetime,
			"max_valid_lifetime":       subnet.MaxValidLifetime,
			"min_valid_lifetime":       subnet.MinValidLifetime,
			"domain_servers":           subnet.DomainServers,
			"routers":                  subnet.Routers,
			"client_class":             subnet.ClientClass,
			"iface_name":               subnet.IfaceName,
			"relay_agent_addresses":    subnet.RelayAgentAddresses,
			"relay_agent_interface_id": subnet.RelayAgentInterfaceId,
			"tags":                     subnet.Tags,
			"network_type":             subnet.NetworkType,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendUpdateSubnetCmdToDDIAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return subnet, nil
}

func setSubnetFromDB(tx restdb.Transaction, subnet *resource.Subnet) error {
	var subnets []*resource.Subnet
	if err := tx.Fill(map[string]interface{}{restdb.IDField: subnet.GetID()}, &subnets); err != nil {
		return fmt.Errorf("get subnet %s from db failed: %s", subnet.GetID(), err.Error())
	}

	if len(subnets) == 0 {
		return fmt.Errorf("no found subnet %s", subnet.GetID())
	}

	subnet.Version = subnets[0].Version
	subnet.SubnetId = subnets[0].SubnetId
	subnet.Capacity = subnets[0].Capacity
	subnet.Subnet = subnets[0].Subnet
	subnet.Ipnet = subnets[0].Ipnet
	return nil
}

func sendUpdateSubnetCmdToDDIAgent(subnet *resource.Subnet) error {
	var req []byte
	var err error
	cmd := services.UpdateSubnet4
	if subnet.Version == util.IPVersion4 {
		req, err = proto.Marshal(&dhcp_agent.UpdateSubnet4Request{
			Id:                  subnet.SubnetId,
			ValidLifetime:       subnet.ValidLifetime,
			MaxValidLifetime:    subnet.MaxValidLifetime,
			MinValidLifetime:    subnet.MinValidLifetime,
			DomainServers:       subnet.DomainServers,
			Routers:             subnet.Routers,
			ClientClass:         subnet.ClientClass,
			IfaceName:           subnet.IfaceName,
			RelayAgentAddresses: subnet.RelayAgentAddresses,
		})
	} else {
		cmd = services.UpdateSubnet6
		req, err = proto.Marshal(&dhcp_agent.UpdateSubnet6Request{
			Id:                    subnet.SubnetId,
			ValidLifetime:         subnet.ValidLifetime,
			MaxValidLifetime:      subnet.MaxValidLifetime,
			MinValidLifetime:      subnet.MinValidLifetime,
			DnsServers:            subnet.DomainServers,
			IfaceName:             subnet.IfaceName,
			RelayAgentAddresses:   subnet.RelayAgentAddresses,
			RelayAgentInterfaceId: subnet.RelayAgentInterfaceId,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal update subnet request failed: %s", err.Error())
	}

	// return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
	return services.NewDHCPAgentService().SendDHCPCmd(cmd, req)
}

func (s *SubnetHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.(*resource.Subnet)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		leasesCount, err := getSubnetLeasesCount(subnet)
		if err != nil {
			return fmt.Errorf("get subnet %s leases count failed: %s", subnet.Subnet, err.Error())
		}

		if leasesCount != 0 {
			return fmt.Errorf("can not delete subnet with %d ips had been allocated", leasesCount)
		}

		if _, err := tx.Delete(resource.TableSubnet,
			map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendDeleteSubnetCmdToDDIAgent(subnet)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	//eventbus.PublishResourceDeleteEvent(subnet)
	return nil
}

func sendDeleteSubnetCmdToDDIAgent(subnet *resource.Subnet) error {
	var req []byte
	var err error
	cmd := services.DeleteSubnet4
	if subnet.Version == util.IPVersion4 {
		req, err = proto.Marshal(&dhcp_agent.DeleteSubnet4Request{Id: subnet.SubnetId})
	} else {
		cmd = services.DeleteSubnet6
		req, err = proto.Marshal(&dhcp_agent.DeleteSubnet6Request{Id: subnet.SubnetId})
	}

	if err != nil {
		return fmt.Errorf("marshal delete subnet %s request failed: %s", subnet.GetID(), err.Error())
	}

	// return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
	return services.NewDHCPAgentService().SendDHCPCmd(cmd, req)
}

func (s *SubnetHandler) checkSubnetAvailable(subnet *resource.Subnet, allSubnets []*resource.Subnet, tx restdb.Transaction) error {
	for _, allSubnet := range allSubnets {
		if subnet.Version == util.IPVersion6 {
			if util.CheckPrefixsContainEachOther(allSubnet.Subnet, &subnet.Ipnet) {
				return fmt.Errorf("the %s exists in subnets", allSubnet.Subnet)
			}
		} else if subnet.Version == util.IPVersion4 {
			if util.PrefixsEqual(allSubnet.Subnet, subnet.Subnet) {
				return fmt.Errorf("the %s exists in subnets", allSubnet.Subnet)
			}
		}
	}

	// return ipamresource.CheckSubNetAvailable(tx, subnet.Ipnet, subnet.Version)
	// TODO
	return nil
}
