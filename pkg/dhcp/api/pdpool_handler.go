package api

import (
	"context"
	"fmt"
	"sort"

	"github.com/golang/protobuf/proto"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/services"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	dhcp_agent "github.com/linkingthing/clxone-dhcp/pkg/pb/dhcp-agent"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
)

type PdPoolHandler struct {
}

func NewPdPoolHandler() *PdPoolHandler {
	return &PdPoolHandler{}
}

func (p *PdPoolHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := pdpool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pdpool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if ones, _ := subnet.Ipnet.Mask.Size(); uint32(ones) > pdpool.PrefixLen {
			return fmt.Errorf("pdpool %s prefix len %d should bigger than subnet mask len %d",
				pdpool.Prefix, pdpool.PrefixLen, ones)
		}

		if checkIPsBelongsToIpnet(subnet.Ipnet, pdpool.Prefix) == false {
			return fmt.Errorf("pdpool %s not belongs to subnet %s", pdpool.Prefix, subnet.Subnet)
		}

		if conflictPool, conflict, err := checkPdPoolConflictWithSubnetPools(tx,
			subnet.GetID(), pdpool); err != nil {
			return err
		} else if conflict {
			return fmt.Errorf("pdpool %s conflict with pool %s in subnet %s",
				pdpool.String(), conflictPool, subnet.GetID())
		}

		pdpool.Subnet = subnet.GetID()
		if _, err := tx.Insert(pdpool); err != nil {
			return err
		}

		return sendCreatePDPoolCmdToDDIAgent(subnet.SubnetId, pdpool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pdpool %s-%d with subnet %s failed: %s",
				pdpool.String(), pdpool.DelegatedLen, subnet.GetID(), err.Error()))
	}

	return pdpool, nil
}

func checkPdPoolConflictWithSubnetPools(tx restdb.Transaction, subnetID string, pdpool *resource.PdPool) (string, bool, error) {
	subnet := pdpool.String()
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &pdpools); err != nil {
		return "", false, fmt.Errorf("get pdpools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, pdpool_ := range pdpools {
		subnet_ := pdpool_.String()
		if checkIPsBelongsToSubnet(subnet_, pdpool.Prefix) || checkIPsBelongsToSubnet(subnet, pdpool_.Prefix) {
			return subnet_, true, nil
		}
	}

	var pools []*resource.Pool
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &pools); err != nil {
		return "", false, fmt.Errorf("get pools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, pool := range pools {
		if pool.Version == util.IPVersion6 {
			if checkIPsBelongsToSubnet(subnet, pool.BeginAddress) {
				return pool.String(), true, nil
			}
		}
	}

	var reservations []*resource.Reservation
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &reservations); err != nil {
		return "", false, fmt.Errorf("get reservations with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, reservation := range reservations {
		if reservation.Version == util.IPVersion6 {
			if checkIPsBelongsToSubnet(subnet, reservation.IpAddress) {
				return reservation.String(), true, nil
			}
		}
	}

	var staticAddresses []*resource.StaticAddress
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &staticAddresses); err != nil {
		return "", false, fmt.Errorf("get static addresses with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, staticAddress := range staticAddresses {
		if staticAddress.Version == util.IPVersion6 {
			if checkIPsBelongsToSubnet(subnet, staticAddress.IpAddress) {
				return staticAddress.String(), true, nil
			}
		}
	}

	return "", false, nil
}

func sendCreatePDPoolCmdToDDIAgent(subnetID uint32, pdpool *resource.PdPool) error {
	req, err := proto.Marshal(&dhcp_agent.CreatePDPoolRequest{
		SubnetId:     subnetID,
		Prefix:       pdpool.Prefix,
		PrefixLen:    pdpool.PrefixLen,
		DelegatedLen: pdpool.DelegatedLen,
		DnsServers:   pdpool.DomainServers,
	})

	if err != nil {
		return fmt.Errorf("marshal create pdpool request failed: %s", err.Error())
	}

	// return kafkaproducer.GetKafkaProducer().SendDHCPCmd(kafkaconsumer.CreatePDPool, req)
	return services.NewDHCPAgentService().SendDHCPCmd(services.CreatePDPool, req)
}

func (p *PdPoolHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var pdpools resource.PdPools
	if err := db.GetResources(map[string]interface{}{"subnet": subnetID}, &pdpools); err != nil {
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
			fmt.Sprintf("get pdpool %s with subnet %s from db failed: %s", pdpoolID, subnetID, err.Error()))
	}

	return pdpool.(*resource.PdPool), nil
}

func (p *PdPoolHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := pdpool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update pdpool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setPdPoolFromDB(tx, pdpool); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TablePdPool, map[string]interface{}{
			"domain_servers": pdpool.DomainServers,
		}, map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return err
		}

		return sendUpdatePdPoolCmdToDDIAgent(subnetID, pdpool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update pdpool %s with subnet %s failed: %s", pdpool.String(), subnetID, err.Error()))
	}

	return pdpool, nil
}

func setPdPoolFromDB(tx restdb.Transaction, pdpool *resource.PdPool) error {
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pdpool.GetID()}, &pdpools); err != nil {
		return fmt.Errorf("get pdpool from db failed: %s", err.Error())
	}

	if len(pdpools) == 0 {
		return fmt.Errorf("no found pool %s", pdpool.GetID())
	}

	pdpool.Subnet = pdpools[0].Subnet
	pdpool.Prefix = pdpools[0].Prefix
	pdpool.Capacity = pdpools[0].Capacity
	return nil
}

func sendUpdatePdPoolCmdToDDIAgent(subnetID string, pdpool *resource.PdPool) error {
	req, err := proto.Marshal(&dhcp_agent.UpdatePDPoolRequest{
		SubnetId:   subnetIDStrToUint32(subnetID),
		Prefix:     pdpool.Prefix,
		DnsServers: pdpool.DomainServers,
	})

	if err != nil {
		return fmt.Errorf("marshal update pdpool request failed: %s", err.Error())
	}

	// return kafkaproducer.GetKafkaProducer().SendDHCPCmd(kafkaconsumer.UpdatePDPool, req)
	return services.NewDHCPAgentService().SendDHCPCmd(services.UpdatePDPool, req)
}

func (p *PdPoolHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	pdpool := ctx.Resource.(*resource.PdPool)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := setPdPoolFromDB(tx, pdpool); err != nil {
			return err
		}

		pdpool.Subnet = subnet.GetID()
		if leasesCount, err := getPdPoolLeasesCount(pdpool); err != nil {
			return fmt.Errorf("get pdpool %s leases count failed: %s", pdpool.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete pdpool with %d ips had been allocated", leasesCount)
		}

		if _, err := tx.Delete(resource.TablePdPool,
			map[string]interface{}{restdb.IDField: pdpool.GetID()}); err != nil {
			return err
		}

		return sendDeletePdPoolCmdToDDIAgent(subnet.SubnetId, pdpool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete pdpool %s with subnet %s failed: %s",
				pdpool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func getPdPoolLeasesCount(pdpool *resource.PdPool) (uint64, error) {
	if pdpool.Capacity == 0 {
		return 0, nil
	}

	beginAddr, endAddr := pdpool.GetRange()
	resp, err := grpcclient.GetDHCPGrpcClient().GetPool6LeasesCount(context.TODO(),
		&dhcp_agent.GetPool6LeasesCountRequest{
			SubnetId:     subnetIDStrToUint32(pdpool.Subnet),
			BeginAddress: beginAddr,
			EndAddress:   endAddr,
		})
	return resp.GetLeasesCount(), err
}

func sendDeletePdPoolCmdToDDIAgent(subnetID uint32, pdpool *resource.PdPool) error {
	req, err := proto.Marshal(&dhcp_agent.DeletePDPoolRequest{
		SubnetId: subnetID,
		Prefix:   pdpool.Prefix,
	})

	if err != nil {
		return fmt.Errorf("marshal delete pdpool request failed: %s", err.Error())
	}

	// return kafkaproducer.GetKafkaProducer().SendDHCPCmd(kafkaconsumer.DeletePDPool, req)
	return services.NewDHCPAgentService().SendDHCPCmd(services.DeletePDPool, req)
}
