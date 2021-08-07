package api

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	dhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type Subnet4Handler struct {
}

func NewSubnet4Handler() *Subnet4Handler {
	return &Subnet4Handler{}
}

func (s *Subnet4Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := subnet.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create subnet params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var subnets []*resource.Subnet4
		if err := tx.Fill(map[string]interface{}{"orderby": "subnet_id desc"}, &subnets); err != nil {
			return fmt.Errorf("get subnets from db failed: %s\n", err.Error())
		}

		subnet.SubnetId = 1
		if len(subnets) > 0 {
			subnet.SubnetId = subnets[0].SubnetId + 1
		}

		subnet.SetID(strconv.Itoa(int(subnet.SubnetId)))
		if err := checkSubnet4ConflictWithSubnet4s(subnet, subnets); err != nil {
			return err
		}

		if _, err := tx.Insert(subnet); err != nil {
			return err
		}

		return sendCreateSubnet4CmdToDHCPAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create subnet %s failed: %s", subnet.Subnet, err.Error()))
	}

	return subnet, nil
}

func checkSubnet4ConflictWithSubnet4s(subnet4 *resource.Subnet4, subnets []*resource.Subnet4) error {
	for _, subnet := range subnets {
		if subnet.CheckConflictWithAnother(subnet4) {
			return fmt.Errorf("subnet4 %s conflict with subnet4 %s", subnet4.Subnet, subnet.Subnet)
		}
	}

	return nil
}

func sendCreateSubnet4CmdToDHCPAgent(subnet *resource.Subnet4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateSubnet4,
		&dhcpagent.CreateSubnet4Request{
			Id:                  subnet.SubnetId,
			Subnet:              subnet.Subnet,
			ValidLifetime:       subnet.ValidLifetime,
			MaxValidLifetime:    subnet.MaxValidLifetime,
			MinValidLifetime:    subnet.MinValidLifetime,
			RenewTime:           subnet.ValidLifetime / 2,
			RebindTime:          subnet.ValidLifetime * 3 / 4,
			SubnetMask:          subnet.SubnetMask,
			DomainServers:       subnet.DomainServers,
			Routers:             subnet.Routers,
			ClientClass:         subnet.ClientClass,
			IfaceName:           subnet.IfaceName,
			RelayAgentAddresses: subnet.RelayAgentAddresses,
			NextServer:          subnet.NextServer,
			TftpServer:          subnet.TftpServer,
			Bootfile:            subnet.Bootfile,
		})
}

func (s *Subnet4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	conditions := map[string]interface{}{"orderby": "subnet_id"}
	if subnet, ok := util.GetFilterValueWithEqModifierFromFilters(util.FileNameSubnet, ctx.GetFilters()); ok {
		conditions[util.FileNameSubnet] = subnet
	}

	var subnets resource.Subnet4s
	if err := db.GetResources(conditions, &subnets); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list subnets from db failed: %s", err.Error()))
	}

	subnetsLeasesCount, err := getSubnet4sLeasesCount()
	if err != nil {
		log.Warnf("get subnets leases count failed: %s", err.Error())
	}

	for _, subnet := range subnets {
		if subnet.Capacity != 0 {
			if leasesCount, ok := subnetsLeasesCount[subnet.SubnetId]; ok {
				subnet.UsedCount = leasesCount
				subnet.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(subnet.Capacity))
			}
		}
	}

	sort.Sort(subnets)
	return subnets, nil
}

func getSubnet4sLeasesCount() (map[uint64]uint64, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnets4LeasesCount(context.TODO(),
		&dhcpagent.GetSubnetsLeasesCountRequest{})
	return resp.GetSubnetsLeasesCount(), err
}

func (s *Subnet4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetID()
	var subnets []*resource.Subnet4
	subnetInterface, err := restdb.GetResourceWithID(db.GetDB(), subnetID, &subnets)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get subnet %s from db failed: %s", subnetID, err.Error()))
	}

	subnet := subnetInterface.(*resource.Subnet4)
	if err := setSubnet4LeasesUsedRatio(subnet); err != nil {
		log.Warnf("get subnet %s leases used ratio failed: %s", subnetID, err.Error())
	}

	return subnet, nil
}

func setSubnet4LeasesUsedRatio(subnet *resource.Subnet4) error {
	leasesCount, err := getSubnet4LeasesCount(subnet)
	if err != nil {
		return err
	}

	if leasesCount != 0 {
		subnet.UsedCount = leasesCount
		subnet.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(subnet.Capacity))
	}
	return nil
}

func getSubnet4LeasesCount(subnet *resource.Subnet4) (uint64, error) {
	if subnet.Capacity == 0 {
		return 0, nil
	}

	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4LeasesCount(context.TODO(),
		&dhcpagent.GetSubnet4LeasesCountRequest{Id: subnet.SubnetId})
	return resp.GetLeasesCount(), err
}

func (s *Subnet4Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := subnet.ValidateParams(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update subnet params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
			"valid_lifetime":        subnet.ValidLifetime,
			"max_valid_lifetime":    subnet.MaxValidLifetime,
			"min_valid_lifetime":    subnet.MinValidLifetime,
			"subnet_mask":           subnet.SubnetMask,
			"domain_servers":        subnet.DomainServers,
			"routers":               subnet.Routers,
			"client_class":          subnet.ClientClass,
			"iface_name":            subnet.IfaceName,
			"relay_agent_addresses": subnet.RelayAgentAddresses,
			"next_server":           subnet.NextServer,
			"tftp_server":           subnet.TftpServer,
			"bootfile":              subnet.Bootfile,
			"tags":                  subnet.Tags,
			"network_type":          subnet.NetworkType,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendUpdateSubnet4CmdToDHCPAgent(subnet)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return subnet, nil
}

func setSubnet4FromDB(tx restdb.Transaction, subnet *resource.Subnet4) error {
	var subnets []*resource.Subnet4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: subnet.GetID()}, &subnets); err != nil {
		return fmt.Errorf("get subnet %s from db failed: %s", subnet.GetID(), err.Error())
	}

	if len(subnets) == 0 {
		return fmt.Errorf("no found subnet %s", subnet.GetID())
	}

	subnet.SubnetId = subnets[0].SubnetId
	subnet.Capacity = subnets[0].Capacity
	subnet.Subnet = subnets[0].Subnet
	subnet.Ipnet = subnets[0].Ipnet
	return nil
}

func sendUpdateSubnet4CmdToDHCPAgent(subnet *resource.Subnet4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.UpdateSubnet4,
		&dhcpagent.UpdateSubnet4Request{
			Id:                  subnet.SubnetId,
			Subnet:              subnet.Subnet,
			ValidLifetime:       subnet.ValidLifetime,
			MaxValidLifetime:    subnet.MaxValidLifetime,
			MinValidLifetime:    subnet.MinValidLifetime,
			RenewTime:           subnet.ValidLifetime / 2,
			RebindTime:          subnet.ValidLifetime * 3 / 4,
			SubnetMask:          subnet.SubnetMask,
			DomainServers:       subnet.DomainServers,
			Routers:             subnet.Routers,
			ClientClass:         subnet.ClientClass,
			IfaceName:           subnet.IfaceName,
			NextServer:          subnet.NextServer,
			TftpServer:          subnet.TftpServer,
			Bootfile:            subnet.Bootfile,
			RelayAgentAddresses: subnet.RelayAgentAddresses,
		})
}

func (s *Subnet4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.(*resource.Subnet4)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if leasesCount, err := getSubnet4LeasesCount(subnet); err != nil {
			return fmt.Errorf("get subnet %s leases count failed: %s", subnet.Subnet, err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete subnet with %d ips had been allocated", leasesCount)
		}

		if _, err := tx.Delete(resource.TableSubnet4,
			map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return err
		}

		return sendDeleteSubnet4CmdToDHCPAgent(subnet)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete subnet %s failed: %s", subnet.GetID(), err.Error()))
	}

	return nil
}

func sendDeleteSubnet4CmdToDHCPAgent(subnet *resource.Subnet4) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteSubnet4,
		&dhcpagent.DeleteSubnet4Request{Id: subnet.SubnetId})
}
