package api

import (
	"context"
	"fmt"
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

type ReservedPdPoolHandler struct {
}

func NewReservedPdPoolHandler() *ReservedPdPoolHandler {
	return &ReservedPdPoolHandler{}
}

func (p *ReservedPdPoolHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pdpool := ctx.Resource.(*resource.ReservedPdPool)
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

		if err := checkReservedPdPoolConflictWithSubnet6Pools(tx, subnet.GetID(), pdpool); err != nil {
			return err
		}

		pdpool.Subnet6 = subnet.GetID()
		if _, err := tx.Insert(pdpool); err != nil {
			return err
		}

		return sendCreateReservedPdPoolCmdToDHCPAgent(subnet.SubnetId, pdpool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pdpool %s-%d with subnet %s failed: %s",
				pdpool.String(), pdpool.DelegatedLen, subnet.GetID(), err.Error()))
	}

	return pdpool, nil
}

func checkReservedPdPoolConflictWithSubnet6Pools(tx restdb.Transaction, subnetID string, pdpool *resource.ReservedPdPool) error {
	if err := checkReservedPdPoolConflictWithSubnet6ReservedPdPools(tx, subnetID, pdpool); err != nil {
		return err
	}

	return checkReservedPdPoolConflictWithSubnet6Reservation6s(tx, subnetID, pdpool)
}

func checkReservedPdPoolConflictWithSubnet6ReservedPdPools(tx restdb.Transaction, subnetID string, pdpool *resource.ReservedPdPool) error {
	var pdpools []*resource.ReservedPdPool
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

func checkReservedPdPoolConflictWithSubnet6Reservation6s(tx restdb.Transaction, subnetID string, pdpool *resource.ReservedPdPool) error {
	var reservations []*resource.Reservation6
	if err := tx.Fill(map[string]interface{}{"subnet6": subnetID}, &reservations); err != nil {
		return fmt.Errorf("get reservation6s with subnet %s from db failed: %s",
			subnetID, err.Error())
	}

	for _, reservation := range reservations {
		for _, prefix := range reservation.Prefixes {
			if pdpool.Contains(prefix) {
				return fmt.Errorf("reserved pdpool %s conflict with reservation6 %s prefix %s",
					pdpool.String(), reservation.String(), prefix)
			}
		}
	}

	return nil
}

func sendCreateReservedPdPoolCmdToDHCPAgent(subnetID uint64, pdpool *resource.ReservedPdPool) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.CreateReservedPdPool,
		&dhcpagent.CreateReservedPdPoolRequest{
			SubnetId:     subnetID,
			Prefix:       pdpool.Prefix,
			PrefixLen:    pdpool.PrefixLen,
			DelegatedLen: pdpool.DelegatedLen,
		})
}

func (p *ReservedPdPoolHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var pdpools resource.ReservedPdPools
	if err := db.GetResources(map[string]interface{}{"subnet6": subnetID}, &pdpools); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pdpools with subnet %s from db failed: %s", subnetID, err.Error()))
	}

	sort.Sort(pdpools)
	return pdpools, nil
}

func (p *ReservedPdPoolHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pdpoolID := ctx.Resource.GetID()
	var pdpools []*resource.ReservedPdPool
	pdpool, err := restdb.GetResourceWithID(db.GetDB(), pdpoolID, &pdpools)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pdpool %s with subnet %s from db failed: %s",
				pdpoolID, subnetID, err.Error()))
	}

	return pdpool.(*resource.ReservedPdPool), nil
}

func (p *ReservedPdPoolHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet6)
	pdpool := ctx.Resource.(*resource.ReservedPdPool)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservedPdPoolFromDB(tx, pdpool); err != nil {
			return err
		}

		pdpool.Subnet6 = subnet.GetID()
		if leasesCount, err := getReservedPdPoolLeasesCount(pdpool); err != nil {
			return fmt.Errorf("get pdpool %s leases count failed: %s", pdpool.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete pdpool with %d ips had been allocated", leasesCount)
		}

		if _, err := tx.Delete(resource.TableReservedPdPool,
			map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservedPdPoolCmdToDHCPAgent(subnet.SubnetId, pdpool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pdpool %s with subnet %s failed: %s",
				pdpool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func setReservedPdPoolFromDB(tx restdb.Transaction, pdpool *resource.ReservedPdPool) error {
	var pdpools []*resource.ReservedPdPool
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

func getReservedPdPoolLeasesCount(pdpool *resource.ReservedPdPool) (uint64, error) {
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

func sendDeleteReservedPdPoolCmdToDHCPAgent(subnetID uint64, pdpool *resource.ReservedPdPool) error {
	return dhcpservice.GetDHCPAgentService().SendDHCPCmd(dhcpservice.DeleteReservedPdPool,
		&dhcpagent.DeleteReservedPdPoolRequest{
			SubnetId:     subnetID,
			Prefix:       pdpool.Prefix,
			PrefixLen:    pdpool.PrefixLen,
			DelegatedLen: pdpool.DelegatedLen,
		})
}
