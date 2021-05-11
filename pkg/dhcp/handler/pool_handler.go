package handler

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	//"github.com/linkingthing/clxone-dhcp/pkg/eventbus"
	"github.com/linkingthing/clxone-dhcp/pkg/grpcclient"
	"github.com/linkingthing/clxone-dhcp/pkg/kafkaproducer"
	"github.com/linkingthing/clxone-dhcp/pkg/util"
	"github.com/linkingthing/ddi-agent/pkg/dhcp/kafkaconsumer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

type PoolHandler struct {
}

func NewPoolHandler() *PoolHandler {
	return &PoolHandler{}
}

func (p *PoolHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	pool := ctx.Resource.(*resource.Pool)
	if err := pool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create pool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkPoolCouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := recalculatePoolCapacity(tx, subnet.GetID(), pool); err != nil {
			return fmt.Errorf("recalculate pool capacity failed: %s", err.Error())
		}

		if _, err := tx.Update(resource.TableSubnet, map[string]interface{}{
			"capacity": subnet.Capacity + pool.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
		}

		pool.Subnet = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return err
		}

		return sendCreatePoolCmdToDDIAgent(subnet.SubnetId, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create pool %s with subnet %s failed: %s", pool.String(), subnet.GetID(), err.Error()))
	}

	//eventbus.PublishResourceCreateEvent(pool)
	return pool, nil
}

func checkPoolCouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet, pool *resource.Pool) error {
	if err := setSubnetFromDB(tx, subnet); err != nil {
		return err
	}

	if err := checkSubnetIfCanCreateDynamicPool(subnet); err != nil {
		return err
	}

	if pool.Template != "" {
		if err := pool.ParseAddressWithTemplate(tx, subnet); err != nil {
			return err
		}
	} else {
		if subnet.Version != pool.Version {
			return fmt.Errorf("pool %s version is diff from subnet %s version", pool.String(), subnet.Subnet)
		}
	}

	if checkIPsBelongsToIpnet(subnet.Ipnet, pool.BeginAddress, pool.EndAddress) == false {
		return fmt.Errorf("pool %s not belongs to subnet %s", pool.String(), subnet.Subnet)
	}

	if conflictPool, conflict, err := checkPoolConflictWithSubnetPools(tx, subnet.GetID(), pool); err != nil {
		return err
	} else if conflict {
		return fmt.Errorf("pool %s conflicts with pool %s in subnet %s",
			pool.String(), conflictPool, subnet.GetID())
	}

	if staticAddress, conflict, err := checkPoolConflictWithSubnetStaticAddress(tx,
		subnet.GetID(), pool); err != nil {
		return err
	} else if conflict {
		return fmt.Errorf("pool %s conflicts with static address %s in subnet %s",
			pool.String(), staticAddress, subnet.GetID())
	}

	return nil
}

func checkSubnetIfCanCreateDynamicPool(subnet *resource.Subnet) error {
	if subnet.Version == util.IPVersion6 {
		if ones, _ := subnet.Ipnet.Mask.Size(); ones != 64 {
			return fmt.Errorf("only subnet which mask len is 64 can create dynamic pool, current mask len is %d", ones)
		}
	}

	return nil
}

func checkPoolConflictWithSubnetPools(tx restdb.Transaction, subnetID string, pool *resource.Pool) (string, bool, error) {
	conflictPool, conflict, err := checkPoolConflictWithSubnetPool(tx, subnetID, pool)
	if err != nil || conflict {
		return conflictPool.String(), conflict, err
	}

	return checkIPConflictWithSubnetPDPool(tx, subnetID, pool.BeginAddress)
}

func checkPoolConflictWithSubnetPool(tx restdb.Transaction, subnetID string, pool *resource.Pool) (*resource.Pool, bool, error) {
	var pools []*resource.Pool
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &pools); err != nil {
		return nil, false, fmt.Errorf("get pools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, p := range pools {
		if p.CheckConflictWithAnother(pool) {
			return p, true, nil
		}
	}

	return nil, false, nil
}

func checkIPConflictWithSubnetPDPool(tx restdb.Transaction, subnetID, ip string) (string, bool, error) {
	var pdpools []*resource.PdPool
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &pdpools); err != nil {
		return "", false, fmt.Errorf("get pdpools with subnet %s from db failed: %s", subnetID, err.Error())
	}

	for _, pdpool := range pdpools {
		subnet := pdpool.String()
		if checkIPsBelongsToSubnet(subnet, ip) {
			return subnet, true, nil
		}
	}

	return "", false, nil
}

func checkIPsBelongsToSubnet(subnet string, ips ...string) bool {
	_, ipnet, _ := net.ParseCIDR(subnet)
	return checkIPsBelongsToIpnet(*ipnet, ips...)
}

func checkIPsBelongsToIpnet(ipnet net.IPNet, ips ...string) bool {
	for _, ip := range ips {
		if ipnet.Contains(net.ParseIP(ip)) == false {
			return false
		}
	}

	return true
}

func checkPoolConflictWithSubnetStaticAddress(tx restdb.Transaction, subnetID string, pool *resource.Pool) (string, bool, error) {
	var staticAddresses []*resource.StaticAddress
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &staticAddresses); err != nil {
		return "", false, err
	}

	for _, staticAddress := range staticAddresses {
		if pool.Contains(staticAddress.IpAddress) {
			return staticAddress.String(), true, nil
		}
	}

	return "", false, nil
}

func recalculatePoolCapacity(tx restdb.Transaction, subnetID string, pool *resource.Pool) error {
	var reservations []*resource.Reservation
	if err := tx.Fill(map[string]interface{}{"subnet": subnetID}, &reservations); err != nil {
		return err
	}

	for _, reservation := range reservations {
		if pool.Contains(reservation.IpAddress) {
			pool.Capacity -= reservation.Capacity
		}
	}

	return nil
}

func sendCreatePoolCmdToDDIAgent(subnetID uint32, pool *resource.Pool) error {
	var req []byte
	var err error
	cmd := kafkaconsumer.CreatePool4
	if pool.Version == util.IPVersion4 {
		req, err = proto.Marshal(&pb.CreatePool4Request{
			SubnetId:      subnetID,
			BeginAddress:  pool.BeginAddress,
			EndAddress:    pool.EndAddress,
			DomainServers: pool.DomainServers,
			Routers:       pool.Routers,
			ClientClass:   pool.ClientClass,
		})
	} else {
		cmd = kafkaconsumer.CreatePool6
		req, err = proto.Marshal(&pb.CreatePool6Request{
			SubnetId:     subnetID,
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
			DnsServers:   pool.DomainServers,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal create pool request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}

func subnetIDStrToUint32(subnetID string) uint32 {
	id, _ := strconv.Atoi(subnetID)
	return uint32(id)
}

func (p *PoolHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	var pools resource.Pools
	var reservations resource.Reservations
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := tx.Fill(map[string]interface{}{"subnet": subnet.GetID()}, &pools); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{"subnet": subnet.GetID()}, &reservations)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list pools with subnet %s from db failed: %s", subnet.GetID(), err.Error()))
	}

	poolsLeases := loadPoolsLeases(subnet, pools, reservations)
	for _, pool := range pools {
		setPoolLeasesUsedRatio(pool, poolsLeases[pool.GetID()])
	}

	sort.Sort(pools)
	return pools, nil
}

func loadPoolsLeases(subnet *resource.Subnet, pools resource.Pools, reservations resource.Reservations) map[string]uint64 {
	resp, err := resource.LoadSubnetLeases(subnet)
	if err != nil {
		log.Warnf("get subnet %s leases failed: %s", subnet.GetID(), err.Error())
		return nil
	}

	if len(resp.GetLeases()) == 0 {
		return nil
	}

	reservationMap := make(map[string]struct{})
	for _, reservation := range reservations {
		reservationMap[reservation.IpAddress] = struct{}{}
	}

	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if _, ok := reservationMap[lease.GetAddress()]; ok {
			continue
		}

		for _, pool := range pools {
			if pool.Capacity != 0 && pool.Contains(lease.GetAddress()) {
				count := leasesCount[pool.GetID()]
				count += 1
				leasesCount[pool.GetID()] = count
			}
		}
	}

	return leasesCount
}

func (p *PoolHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	poolID := ctx.Resource.GetID()
	var pools resource.Pools
	var reservations resource.Reservations
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools); err != nil {
			return err
		}

		return tx.Fill(map[string]interface{}{"subnet": subnetID}, &reservations)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get pool %s with subnet %s from db failed: %s", poolID, subnetID, err.Error()))
	}

	if len(pools) != 1 {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("no found pool %s with subnet %s", poolID, subnetID))
	}

	leasesCount, err := loadPoolLeasesCount(pools[0], reservations)
	if err != nil {
		log.Warnf("get pool %s with subnet %s from db failed: %s", poolID, subnetID, err.Error())
	}

	setPoolLeasesUsedRatio(pools[0], leasesCount)
	return pools[0], nil
}

func loadPoolLeasesCount(pool *resource.Pool, reservations resource.Reservations) (uint64, error) {
	if pool.Capacity == 0 {
		return 0, nil
	}

	var resp *pb.GetLeasesResponse
	var err error
	if pool.Version == util.IPVersion4 {
		resp, err = grpcclient.GetDHCPGrpcClient().GetPool4Leases(context.TODO(),
			&pb.GetPool4LeasesRequest{
				SubnetId:     subnetIDStrToUint32(pool.Subnet),
				BeginAddress: pool.BeginAddress,
				EndAddress:   pool.EndAddress,
			})
	} else {
		resp, err = grpcclient.GetDHCPGrpcClient().GetPool6Leases(context.TODO(),
			&pb.GetPool6LeasesRequest{
				SubnetId:     subnetIDStrToUint32(pool.Subnet),
				BeginAddress: pool.BeginAddress,
				EndAddress:   pool.EndAddress,
			})
	}

	if err != nil {
		return 0, err
	}

	if len(resp.GetLeases()) == 0 {
		return 0, nil
	}

	if len(reservations) == 0 {
		return uint64(len(resp.GetLeases())), nil
	}

	reservationMap := make(map[string]struct{})
	for _, reservation := range reservations {
		reservationMap[reservation.IpAddress] = struct{}{}
	}

	var leasesCount uint64
	for _, lease := range resp.GetLeases() {
		if _, ok := reservationMap[lease.GetAddress()]; ok == false {
			leasesCount += 1
		}
	}

	return leasesCount, nil
}

func setPoolLeasesUsedRatio(pool *resource.Pool, leasesCount uint64) {
	if leasesCount != 0 && pool.Capacity != 0 {
		pool.UsedCount = leasesCount
		pool.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(pool.Capacity))
	}
}

func (p *PoolHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	pool := ctx.Resource.(*resource.Pool)
	if err := pool.ValidateParams(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update pool params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setPoolFromDB(tx, pool); err != nil {
			return err
		}

		if _, err := tx.Update(resource.TablePool, map[string]interface{}{
			"domain_servers": pool.DomainServers,
			"routers":        pool.Routers,
			"client_class":   pool.ClientClass,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendUpdatePoolCmdToDDIAgent(subnetID, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update pool %s with subnet %s failed: %s", pool.String(), subnetID, err.Error()))
	}

	return pool, nil
}

func setPoolFromDB(tx restdb.Transaction, pool *resource.Pool) error {
	var pools []*resource.Pool
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pool.GetID()}, &pools); err != nil {
		return fmt.Errorf("get pool from db failed: %s", err.Error())
	}

	if len(pools) == 0 {
		return fmt.Errorf("no found pool %s", pool.GetID())
	}

	pool.Subnet = pools[0].Subnet
	pool.BeginAddress = pools[0].BeginAddress
	pool.EndAddress = pools[0].EndAddress
	pool.Version = pools[0].Version
	pool.Capacity = pools[0].Capacity
	return nil
}

func sendUpdatePoolCmdToDDIAgent(subnetID string, pool *resource.Pool) error {
	var req []byte
	var err error
	cmd := kafkaconsumer.UpdatePool4
	if pool.Version == util.IPVersion4 {
		req, err = proto.Marshal(&pb.UpdatePool4Request{
			SubnetId:      subnetIDStrToUint32(subnetID),
			BeginAddress:  pool.BeginAddress,
			EndAddress:    pool.EndAddress,
			DomainServers: pool.DomainServers,
			Routers:       pool.Routers,
			ClientClass:   pool.ClientClass,
		})
	} else {
		cmd = kafkaconsumer.UpdatePool6
		req, err = proto.Marshal(&pb.UpdatePool6Request{
			SubnetId:     subnetIDStrToUint32(subnetID),
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
			DnsServers:   pool.DomainServers,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal update pool request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}

func (p *PoolHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	pool := ctx.Resource.(*resource.Pool)
	var reservations resource.Reservations
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnetFromDB(tx, subnet); err != nil {
			return err
		}

		if err := setPoolFromDB(tx, pool); err != nil {
			return err
		}

		if err := tx.Fill(map[string]interface{}{"subnet": subnet.GetID()}, &reservations); err != nil {
			return err
		}

		if leasesCount, err := loadPoolLeasesCount(pool, reservations); err != nil {
			return fmt.Errorf("get pool %s leases count failed: %s", pool.String(), err.Error())
		} else if leasesCount != 0 {
			return fmt.Errorf("can not delete pool with %d ips had been allocated", leasesCount)
		}

		if _, err := tx.Update(resource.TableSubnet, map[string]interface{}{
			"capacity": subnet.Capacity - pool.Capacity,
		}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
			return fmt.Errorf("update subnet %s capacity to db failed: %s", subnet.GetID(), err.Error())
		}

		if _, err := tx.Delete(resource.TablePool, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendDeletePoolCmdToDDIAgent(subnet.SubnetId, pool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete pool %s with subnet %s failed: %s",
			pool.String(), subnet.GetID(), err.Error()))
	}

	//eventbus.PublishResourceDeleteEvent(pool)
	return nil
}

func sendDeletePoolCmdToDDIAgent(subnetID uint32, pool *resource.Pool) error {
	var req []byte
	var err error
	cmd := kafkaconsumer.DeletePool4
	if pool.Version == util.IPVersion4 {
		req, err = proto.Marshal(&pb.DeletePool4Request{
			SubnetId:     subnetID,
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
		})
	} else {
		cmd = kafkaconsumer.DeletePool6
		req, err = proto.Marshal(&pb.DeletePool6Request{
			SubnetId:     subnetID,
			BeginAddress: pool.BeginAddress,
			EndAddress:   pool.EndAddress,
		})
	}

	if err != nil {
		return fmt.Errorf("marshal delete pool request failed: %s", err.Error())
	}

	return kafkaproducer.GetKafkaProducer().SendDHCPCmd(cmd, req)
}

func (h *PoolHandler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameValidTemplate:
		return h.validTemplate(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *PoolHandler) validTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet)
	pool := ctx.Resource.(*resource.Pool)
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("parse action refresh input invalid"))
	}

	pool.Template = templateInfo.Template

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkPoolCouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("template %s invalid: %s", pool.Template, err.Error()))
	}

	return &resource.TemplatePool{BeginAddress: pool.BeginAddress, EndAddress: pool.EndAddress}, nil
}
