package service

import (
	"context"
	"fmt"
	"net"

	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/kafka"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type PdPoolService struct {
}

func NewPdPoolService() *PdPoolService {
	return &PdPoolService{}
}

func (p *PdPoolService) Create(subnet *resource.Subnet6, pdpool *resource.PdPool) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet6FromDB(tx, subnet); err != nil {
			return err
		} else if subnet.UseEui64 {
			return fmt.Errorf("subnet use EUI64, can not create pdpool")
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
		return nil, err
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
	nodesForSucceed, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.CreatePdPool,
		pdpoolToCreatePdPoolRequest(subnetID, pdpool))
	if err != nil {
		if _, err := kafka.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, kafka.DeletePdPool,
			pdpoolToDeletePdPoolRequest(subnetID, pdpool)); err != nil {
			log.Errorf("create subnet %d pdpool %s failed, and rollback it failed: %s",
				subnetID, pdpool.String(), err.Error())
		}
	}

	return err
}

func pdpoolToCreatePdPoolRequest(subnetID uint64, pdpool *resource.PdPool) *pbdhcpagent.CreatePdPoolRequest {
	return &pbdhcpagent.CreatePdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}

func (p *PdPoolService) List(subnetID string) (interface{}, error) {
	var pdpools []*resource.PdPool
	if err := db.GetResources(map[string]interface{}{
		resource.SqlColumnSubnet6: subnetID,
		util.SqlOrderBy:           resource.SqlColumnPrefixIpNet}, &pdpools); err != nil {
		return nil, err
	}

	return pdpools, nil
}

func (p *PdPoolService) Get(subnetID, pdpoolID string) (restresource.Resource, error) {
	var pdpools []*resource.PdPool
	pdpool, err := restdb.GetResourceWithID(db.GetDB(), pdpoolID, &pdpools)
	if err != nil {
		return nil, err
	}

	return pdpool.(*resource.PdPool), nil
}

func (p *PdPoolService) Delete(subnet *resource.Subnet6, pdpool *resource.PdPool) error {
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
		return err
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
		&pbdhcpagent.GetPool6LeasesCountRequest{
			SubnetId:     subnetIDStrToUint64(pdpool.Subnet6),
			BeginAddress: beginAddr,
			EndAddress:   endAddr,
		})
	return resp.GetLeasesCount(), err
}

func sendDeletePdPoolCmdToDHCPAgent(subnetID uint64, nodes []string, pdpool *resource.PdPool) error {
	_, err := kafka.SendDHCPCmdWithNodes(false, nodes, kafka.DeletePdPool,
		pdpoolToDeletePdPoolRequest(subnetID, pdpool))
	return err
}

func pdpoolToDeletePdPoolRequest(subnetID uint64, pdpool *resource.PdPool) *pbdhcpagent.DeletePdPoolRequest {
	return &pbdhcpagent.DeletePdPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
	}
}

func (p *PdPoolService) Update(pdpool *resource.PdPool) (restresource.Resource, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TablePdPool,
			map[string]interface{}{util.SqlColumnsComment: pdpool.Comment},
			map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found pdpool %s", pdpool.GetID())
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return pdpool, nil
}
