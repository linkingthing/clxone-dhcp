package api

import (
	"context"
	"fmt"
	"net"
	"sort"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

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

		if err := checkPrefixBelongsToIpnet(subnet.Ipnet, pdpool.Prefix, pdpool.PrefixLen); err != nil {
			return err
		}

		if err := checkPdPoolConflictWithSubnet6PdPools(tx, subnet.GetID(), pdpool); err != nil {
			return err
		}

		pdpool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pdpool); err != nil {
			return err
		}

		return sendCreatePdPoolCmdToDHCPAgent(subnet.SubnetId, pdpool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pdpool %s-%d with subnet %s failed: %s",
				pdpool.String(), pdpool.DelegatedLen, subnet.GetID(), err.Error()))
	}

	return pdpool, nil
}

func checkPrefixBelongsToIpnet(ipnet net.IPNet, prefix string, prefixLen uint32) error {
	if ones, _ := ipnet.Mask.Size(); uint32(ones) > prefixLen {
		return fmt.Errorf("pdpool %s prefix len %d should bigger than subnet mask len %d",
			prefix, prefixLen, ones)
	}

	if checkIPsBelongsToIpnet(ipnet, prefix) == false {
		return fmt.Errorf("pdpool %s not belongs to subnet %s", prefix, ipnet.String())
	}

	return nil
}

func checkPdPoolConflictWithSubnet6PdPools(tx restdb.Transaction, subnetID string, pdpool *resource.PdPool) error {
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetID}, &pdpools); err != nil {
		return fmt.Errorf("get pdpools with subnet %s from db failed: %s",
			subnetID, err.Error())
	}

	for _, pdpool_ := range pdpools {
		if pdpool_.CheckConflictWithAnother(pdpool) {
			return fmt.Errorf("pdpool %s conflict with pdpool %s",
				pdpool.String(), pdpool_.String())
		}
	}

	return nil
}

func sendCreatePdPoolCmdToDHCPAgent(subnetID uint64, pdpool *resource.PdPool) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreatePdPool,
		&dhcpagent.CreatePdPoolRequest{
			SubnetId:     subnetID,
			Prefix:       pdpool.Prefix,
			PrefixLen:    pdpool.PrefixLen,
			DelegatedLen: pdpool.DelegatedLen,
		})
}

func (p *PdPoolHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var pdpools resource.PdPools
	if err := db.GetResources(map[string]interface{}{"subnet6": subnetID}, &pdpools); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pdpools with subnet %s from db failed: %s", subnetID, err.Error()))
	}

	sort.Sort(pdpools)
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
			return fmt.Errorf("get pdpool %s leases count failed: %s", pdpool.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete pdpool with %d ips had been allocated", leasesCount)
		}

		if _, err := tx.Delete(resource.TablePdPool,
			map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return err
		}

		return sendDeletePdPoolCmdToDHCPAgent(subnet.SubnetId, pdpool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pdpool %s with subnet %s failed: %s",
				pdpool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func setPdPoolFromDB(tx restdb.Transaction, pdpool *resource.PdPool) error {
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pdpool.GetID()}, &pdpools); err != nil {
		return fmt.Errorf("get pdpool from db failed: %s", err.Error())
	}

	if len(pdpools) == 0 {
		return fmt.Errorf("no found pool %s", pdpool.GetID())
	}

	pdpool.Subnet6 = pdpools[0].Subnet6
	pdpool.Prefix = pdpools[0].Prefix
	pdpool.PrefixLen = pdpools[0].PrefixLen
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

func sendDeletePdPoolCmdToDHCPAgent(subnetID uint64, pdpool *resource.PdPool) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeletePdPool,
		&dhcpagent.DeletePdPoolRequest{
			SubnetId:     subnetID,
			Prefix:       pdpool.Prefix,
			PrefixLen:    pdpool.PrefixLen,
			DelegatedLen: pdpool.DelegatedLen,
		})
}
