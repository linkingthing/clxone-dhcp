package api

import (
	"fmt"
	"net"

	gohelperip "github.com/cuityhj/gohelper/ip"
	"github.com/linkingthing/cement/log"
	restdb "github.com/linkingthing/gorest/db"
	resterror "github.com/linkingthing/gorest/error"
	restresource "github.com/linkingthing/gorest/resource"

	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	dhcpservice "github.com/linkingthing/clxone-dhcp/pkg/dhcp/service"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
)

type ReservedPool4Handler struct {
}

func NewReservedPool4Handler() *ReservedPool4Handler {
	return &ReservedPool4Handler{}
}

func (p *ReservedPool4Handler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.ReservedPool4)
	if err := pool.Validate(); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("create reserved pool4 params invalid: %s", err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := checkReservedPool4CouldBeCreated(tx, subnet, pool); err != nil {
			return err
		}

		if err := updateSubnet4AndPoolsCapacityWithReservedPool4(tx, subnet,
			pool, true); err != nil {
			return err
		}

		pool.Subnet4 = subnet.GetID()
		if _, err := tx.Insert(pool); err != nil {
			return err
		}

		return sendCreateReservedPool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create reserved pool4 %s with subnet4 %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return pool, nil
}

func checkReservedPool4CouldBeCreated(tx restdb.Transaction, subnet *resource.Subnet4, pool *resource.ReservedPool4) error {
	if err := setSubnet4FromDB(tx, subnet); err != nil {
		return err
	}

	if pool.Template != "" {
		if err := pool.ParseAddressWithTemplate(tx, subnet); err != nil {
			return err
		}
	}

	if checkIPsBelongsToIpnet(subnet.Ipnet, pool.BeginIp, pool.EndIp) == false {
		return fmt.Errorf("reserved pool4 %s not belongs to subnet4 %s",
			pool.String(), subnet.Subnet)
	}

	return checkReservedPool4ConflictWithSubnet4Pools(tx, subnet.GetID(), pool)
}

func checkReservedPool4ConflictWithSubnet4Pools(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool4) error {
	if err := checkReservedPool4ConflictWithSubnet4ReservedPool4s(tx,
		subnetID, pool); err != nil {
		return err
	}

	return checkReservedPool4ConflictWithSubnet4Reservation4s(tx, subnetID, pool)
}

func checkReservedPool4ConflictWithSubnet4ReservedPool4s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool4) error {
	if reservedpools, err := getReservedPool4sWithBeginAndEndIp(tx, subnetID,
		pool.BeginIp, pool.EndIp); err != nil {
		return err
	} else if len(reservedpools) != 0 {
		return fmt.Errorf("reserved pool4 %s conflict with exists reserved pool4 %s",
			pool.String(), reservedpools[0].String())
	} else {
		return nil
	}
}

func getReservedPool4sWithBeginAndEndIp(tx restdb.Transaction, subnetID string, begin, end net.IP) ([]*resource.ReservedPool4, error) {
	var reservedpools []*resource.ReservedPool4
	if err := tx.FillEx(&reservedpools,
		"select * from gr_reserved_pool4 where subnet4 = $1 and begin_ip <= $2 and end_ip >= $3",
		subnetID, end, begin); err != nil {
		return nil, fmt.Errorf("get reserved pool4s with subnet4 %s from db failed: %s",
			subnetID, err.Error())
	} else {
		return reservedpools, nil
	}
}

func checkReservedPool4ConflictWithSubnet4Reservation4s(tx restdb.Transaction, subnetID string, pool *resource.ReservedPool4) error {
	if reservations, err := getReservation4sWithBeginAndEndIp(tx, subnetID,
		pool.BeginIp, pool.EndIp); err != nil {
		return err
	} else if len(reservations) != 0 {
		return fmt.Errorf("reserved pool4 %s conflict with exists reservation4 %s",
			pool.String(), reservations[0].String())
	} else {
		return nil
	}
}

func updateSubnet4AndPoolsCapacityWithReservedPool4(tx restdb.Transaction, subnet *resource.Subnet4, reservedPool *resource.ReservedPool4, isCreate bool) error {
	affectPools, err := recalculatePool4sCapacityWithReservedPool4(tx, subnet,
		reservedPool, isCreate)
	if err != nil {
		return fmt.Errorf("recalculate pool4s capacity failed: %s", err.Error())
	}

	if _, err := tx.Update(resource.TableSubnet4, map[string]interface{}{
		"capacity": subnet.Capacity,
	}, map[string]interface{}{restdb.IDField: subnet.GetID()}); err != nil {
		return fmt.Errorf("update subnet4 %s capacity to db failed: %s",
			subnet.GetID(), err.Error())
	}

	for affectPoolID, capacity := range affectPools {
		if _, err := tx.Update(resource.TablePool4, map[string]interface{}{
			"capacity": capacity,
		}, map[string]interface{}{restdb.IDField: affectPoolID}); err != nil {
			return fmt.Errorf("update subnet4 %s pool4 %s capacity to db failed: %s",
				subnet.GetID(), affectPoolID, err.Error())
		}
	}

	return nil
}

func recalculatePool4sCapacityWithReservedPool4(tx restdb.Transaction, subnet *resource.Subnet4, reservedPool *resource.ReservedPool4, isCreate bool) (map[string]uint64, error) {
	pools, err := getPool4sWithBeginAndEndIp(tx, subnet.GetID(),
		reservedPool.BeginIp, reservedPool.EndIp)
	if err != nil {
		return nil, err
	}

	affectedPool4s := make(map[string]uint64)
	for _, pool := range pools {
		reservedCount := getPool4ReservedCountWithReservedPool4(pool, reservedPool)
		if isCreate {
			affectedPool4s[pool.GetID()] = pool.Capacity - reservedCount
			subnet.Capacity -= reservedCount
		} else {
			affectedPool4s[pool.GetID()] = pool.Capacity + reservedCount
			subnet.Capacity += reservedCount
		}
	}

	return affectedPool4s, nil
}

func getPool4ReservedCountWithReservedPool4(pool *resource.Pool4, reservedPool *resource.ReservedPool4) uint64 {
	begin := gohelperip.IPv4ToUint32(pool.BeginIp)
	if reservedPoolBegin := gohelperip.IPv4ToUint32(
		reservedPool.BeginIp); reservedPoolBegin > begin {
		begin = reservedPoolBegin
	}

	end := gohelperip.IPv4ToUint32(pool.EndIp)
	if reservedPoolEnd := gohelperip.IPv4ToUint32(
		reservedPool.EndIp); reservedPoolEnd < end {
		end = reservedPoolEnd
	}

	return uint64(end - begin + 1)
}

func sendCreateReservedPool4CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.ReservedPool4) error {
	nodesForSucceed, err := sendDHCPCmdWithNodes(true, nodes, dhcpservice.CreateReservedPool4,
		reservedPool4ToCreateReservedPool4Request(subnetID, pool))
	if err != nil {
		if _, err := dhcpservice.GetDHCPAgentService().SendDHCPCmdWithNodes(
			nodesForSucceed, dhcpservice.DeleteReservedPool4,
			reservedPool4ToDeleteReservedPool4Request(subnetID, pool)); err != nil {
			log.Errorf("create subnet4 %d reserved pool4 %s failed, and rollback it failed: %s",
				subnetID, pool.String(), err.Error())
		}
	}

	return err
}

func reservedPool4ToCreateReservedPool4Request(subnetID uint64, pool *resource.ReservedPool4) *pbdhcpagent.CreateReservedPool4Request {
	return &pbdhcpagent.CreateReservedPool4Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (p *ReservedPool4Handler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	var pools []*resource.ReservedPool4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{"subnet4": subnetID, "orderby": "begin_ip"},
			&pools)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list reserved pool4s with subnet4 %s from db failed: %s",
				subnetID, err.Error()))
	}

	return pools, nil
}

func (p *ReservedPool4Handler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	subnetID := ctx.Resource.GetParent().GetID()
	poolID := ctx.Resource.GetID()
	var pools []*resource.ReservedPool4
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{restdb.IDField: poolID}, &pools)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get reserved pool4 %s with subnet4 %s from db failed: %s",
				poolID, subnetID, err.Error()))
	}

	if len(pools) != 1 {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("no found reserved pool4 %s with subnet4 %s", poolID, subnetID))
	}

	return pools[0], nil
}

func (p *ReservedPool4Handler) Delete(ctx *restresource.Context) *resterror.APIError {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.ReservedPool4)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := setSubnet4FromDB(tx, subnet); err != nil {
			return err
		}

		if err := setReservedPool4FromDB(tx, pool); err != nil {
			return err
		}

		if err := updateSubnet4AndPoolsCapacityWithReservedPool4(tx, subnet,
			pool, false); err != nil {
			return err
		}

		if _, err := tx.Delete(resource.TableReservedPool4, map[string]interface{}{
			restdb.IDField: pool.GetID()}); err != nil {
			return err
		}

		return sendDeleteReservedPool4CmdToDHCPAgent(subnet.SubnetId, subnet.Nodes, pool)
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete reserved pool4 %s with subnet4 %s failed: %s",
				pool.String(), subnet.GetID(), err.Error()))
	}

	return nil
}

func setReservedPool4FromDB(tx restdb.Transaction, pool *resource.ReservedPool4) error {
	var pools []*resource.ReservedPool4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: pool.GetID()},
		&pools); err != nil {
		return fmt.Errorf("get reserved pool4 from db failed: %s", err.Error())
	} else if len(pools) == 0 {
		return fmt.Errorf("no found reserved pool4 %s", pool.GetID())
	}

	pool.Subnet4 = pools[0].Subnet4
	pool.BeginAddress = pools[0].BeginAddress
	pool.BeginIp = pools[0].BeginIp
	pool.EndAddress = pools[0].EndAddress
	pool.EndIp = pools[0].EndIp
	pool.Capacity = pools[0].Capacity
	return nil
}

func sendDeleteReservedPool4CmdToDHCPAgent(subnetID uint64, nodes []string, pool *resource.ReservedPool4) error {
	_, err := sendDHCPCmdWithNodes(true, nodes, dhcpservice.DeleteReservedPool4,
		reservedPool4ToDeleteReservedPool4Request(subnetID, pool))
	return err
}

func reservedPool4ToDeleteReservedPool4Request(subnetID uint64, pool *resource.ReservedPool4) *pbdhcpagent.DeleteReservedPool4Request {
	return &pbdhcpagent.DeleteReservedPool4Request{
		SubnetId:     subnetID,
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress,
	}
}

func (h *ReservedPool4Handler) Action(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case resource.ActionNameValidTemplate:
		return h.validTemplate(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction,
			fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func (h *ReservedPool4Handler) validTemplate(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	subnet := ctx.Resource.GetParent().(*resource.Subnet4)
	pool := ctx.Resource.(*resource.ReservedPool4)
	templateInfo, ok := ctx.Resource.GetAction().Input.(*resource.TemplateInfo)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("parse action valid pool template input invalid"))
	}

	pool.Template = templateInfo.Template
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return checkReservedPool4CouldBeCreated(tx, subnet, pool)
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("template4 %s invalid: %s", pool.Template, err.Error()))
	}

	return &resource.TemplatePool{
		BeginAddress: pool.BeginAddress,
		EndAddress:   pool.EndAddress}, nil
}

func (p *ReservedPool4Handler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	pool := ctx.Resource.(*resource.ReservedPool4)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if rows, err := tx.Update(resource.TableReservedPool4, map[string]interface{}{
			"comment": pool.Comment,
		}, map[string]interface{}{restdb.IDField: pool.GetID()}); err != nil {
			return err
		} else if rows == 0 {
			return fmt.Errorf("no found reserved pool4 %s", pool.GetID())
		}

		return nil
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("update reserved pool4 %s with subnet4 %s failed: %s",
				pool.String(), ctx.Resource.GetParent().GetID(), err.Error()))
	}

	return pool, nil
}
