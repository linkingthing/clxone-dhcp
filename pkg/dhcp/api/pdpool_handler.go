package api

import (
	"context"
	"fmt"
	"net"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	dhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type PdPoolHandler struct {
}

func NewPdPoolHandler() *PdPoolHandler {
	return &PdPoolHandler{}
}

func (p *PdPoolHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := pdpool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pdpool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := checkPrefixBelongsToIpnet(subnet.Ipnet, pdpool.PrefixIpnet,
			pdpool.PrefixLen); err != nil {
			return err
		}

		if err := checkPdPoolConflictWithSubnet6PdPools(tx, subnet.GetID(),
			pdpool); err != nil {
			return err
		}

		pdpool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pdpool); err != nil {
			return err
		}

		return sendCreatePdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdpool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pdpool %s with subnet %s failed: %s",
				pdpool.String(), subnet.GetID(), err.Error()))
	}

	return pdpool, nil
}

func checkPrefixBelongsToIpnet(ipnet, prefixIpnet net.IPNet, prefixLen uint32) error {
	if ones, _ := ipnet.Mask.Size(); uint32(ones) > prefixLen {
		return fmt.Errorf("pdpool %s prefix len %d should bigger than subnet mask len %d",
			prefixIpnet.String(), prefixLen, ones)
	}

	if checkIPsBelongsToIpnet(ipnet, prefixIpnet.IP) == false {
		return fmt.Errorf("pdpool %s not belongs to subnet %s",
			prefixIpnet.String(), ipnet.String())
	}

	return nil
}

func checkPdPoolConflictWithSubnet6PdPools(tx restdb.Transaction, subnetID string, pdpool *resource.PdPool) error {
	var pdpools []*resource.PdPool
	if err := tx.FillEx(&pdpools,
		"select * from gr_pd_pool where subnet6 = $1 and prefix_ipnet && $2",
		subnetID, pdpool.PrefixIpnet); err != nil {
		return fmt.Errorf("get pdpools with subnet %s from db failed: %s",
			subnetID, err.Error())
	}

	if len(pdpools) != 0 {
		return fmt.Errorf("pdpool %s conflict with pdpool %s",
			pdpool.String(), pdpools[0].String())
	}

	return nil
}

func sendCreatePdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.PdPool) error {
	nodesForSucceed, err := sendDHCPCmdWithNodes(nodes, dhcpservice.CreatePdPool,
		pdpoolToCreatePdPoolRequest(subnetID, pdpool))
	if err != nil {
		if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, dhcpservice.DeletePdPool,
			pdpoolToDeletePdPoolRequest(subnetID, pdpool)); err != nil {
			log.Errorf("create subnet %d pdpool %s failed, and rollback it failed: %s",
				subnetID, pdpool.String(), err.Error())
		}
	}

	return err
}

func pdpoolToCreatePdPoolRequest(subnetID uint64, pdpool *resource.PdPool) *dhcpagent.CreatePdPoolRequest {
	return &dhcpagent.CreatePdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}

func (p *PdPoolHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var pdpools []*resource.PdPool
	if err := db.GetResources(map[string]interface{}{
		"subnet6": subnetID, "orderby": "prefix_ipnet"}, &pdpools); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pdpools with subnet %s from db failed: %s",
				subnetID, err.Error()))
	}

	return pdpools, nil
}

func (p *PdPoolHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pdpoolID := ctx.Resource.GetID()
	var pdpools []*resource.PdPool
	pdpool, err := restdb.GetResourceWithID(db.GetDB(), pdpoolID, &pdpools)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pdpool %s with subnet %s from db failed: %s",
				pdpoolID, subnetID, err.Error()))
	}

	return pdpool.(*resource.PdPool), nil
}

func (p *PdPoolHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setPdPoolFromDB(tx, pdpool); err != nil {
			return err
		}

		pdpool.Subnet6 = subnet.GetID()
		if leasesCount, err := getPdPoolLeasesCount(pdpool); err != nil {
			return fmt.Errorf("get pdpool %s leases count failed: %s",
				pdpool.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete pdpool with %d ips had been allocated",
				leasesCount)
		}

		if _, err := tx.Delete(resource.TablePdPool,
			map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return err
		}

		return sendDeletePdPoolCmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pdpool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pdpool %s with subnet %s failed: %s",
				pdpool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func setPdPoolFromDB(tx restdb.Transaction, pdpool *resource.PdPool) error {
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pdpool.GetID()},
		&pdpools); err != nil {
		return fmt.Errorf("get pdpool from db failed: %s", err.Error())
	}

	if len(pdpools) == 0 {
		return fmt.Errorf("no found pool %s", pdpool.GetID())
	}

	pdpool.Subnet6 = pdpools[0].Subnet6
	pdpool.Prefix = pdpools[0].Prefix
	pdpool.PrefixLen = pdpools[0].PrefixLen
	pdpool.PrefixIpnet = pdpools[0].PrefixIpnet
	pdpool.DelegatedLen = pdpools[0].DelegatedLen
	pdpool.Capacity = pdpools[0].Capacity
	return nil
}

func getPdPoolLeasesCount(pdpool *resource.PdPool) (uint64, error) {
	if pdpool.Capacity == 0 {
		return 0, nil
	}

	beginAddr, endAddr := pdpool.GetRange()
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetPool6LeasesCount(context.TODO(),
		&dhcpagent.GetPool6LeasesCountRequest{
			SubnetId:     subnetIDStrToUint64(pdpool.Subnet6),
			BeginAddress: beginAddr,
			EndAddress:   endAddr,
		})
	return resp.GetLeasesCount(), err
}

func sendDeletePdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.PdPool) error {
	_, err := sendDHCPCmdWithNodes(nodes, dhcpservice.DeletePdPool,
		pdpoolToDeletePdPoolRequest(subnetID, pdpool))
	return err
}

func pdpoolToDeletePdPoolRequest(subnetID uint64, pdpool *resource.PdPool) *dhcpagent.DeletePdPoolRequest {
	return &dhcpagent.DeletePdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}
