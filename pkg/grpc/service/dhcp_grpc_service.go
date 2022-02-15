package service

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/linkingthing/cement/log"
	"github.com/linkingthing/clxone-dhcp/pkg/db"
	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
	grpcclient "github.com/linkingthing/clxone-dhcp/pkg/grpc/client"
	"github.com/linkingthing/clxone-dhcp/pkg/grpc/parser"
	pbdhcp "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp"
	pbdhcpagent "github.com/linkingthing/clxone-dhcp/pkg/proto/dhcp-agent"
	pbmonitor "github.com/linkingthing/clxone-dhcp/pkg/proto/monitor"
	restdb "github.com/linkingthing/gorest/db"
)

func genGetSubnetsContext(table restdb.ResourceType) listSubnetContext {
	listCtx := listSubnetContext{}
	sqls := []string{"select * from gr_" + string(table)}
	if listCtx.hasFilterSubnet == false {
		sqls = append(sqls, "order by subnet_id")
	}
	listCtx.sql = strings.Join(sqls, " ")
	return listCtx
}

type AgentRole string

const (
	AgentRoleSentry4 AgentRole = "sentry4"
	AgentRoleSentry6 AgentRole = "sentry6"
)

func getNodeNames(isv4 bool) (map[string]string, error) {
	dhcpNodes, err := grpcclient.GetMonitorGrpcClient().GetDHCPNodes(context.TODO(),
		&pbmonitor.GetDHCPNodesRequest{})
	if err != nil {
		return nil, fmt.Errorf("list dhcp agent4s failed: %s", err.Error())
	}
	sentryRole := AgentRoleSentry4
	if isv4 == false {
		sentryRole = AgentRoleSentry6
	}
	nodeNames := make(map[string]string)
	for _, node := range dhcpNodes.GetNodes() {
		if IsAgentService(node.GetServiceTags(), sentryRole) {
			if node.GetVirtualIp() != "" {
				return map[string]string{node.GetIpv4(): node.GetName()}, nil
			} else {
				nodeNames[node.GetIpv4()] = node.GetName()
			}
		}
	}
	return nodeNames, nil
}

func setSubnet4sNodeNames(subnets []*resource.Subnet4, nodeNames map[string]string) {
	for _, subnet := range subnets {
		subnet.NodeNames = getSubnetNodeNames(subnet.Nodes, nodeNames)
	}
}

func getSubnetNodeNames(nodes []string, nodeNames map[string]string) []string {
	var names []string
	for _, node := range nodes {
		if name, ok := nodeNames[node]; ok {
			names = append(names, name)
		}
	}
	return names
}

func IsAgentService(tags []string, role AgentRole) bool {
	for _, tag := range tags {
		if tag == string(role) {
			return true
		}
	}
	return false
}

func setSubnet4sLeasesUsedInfo(subnets []*resource.Subnet4, ctx listSubnetContext) error {
	if ctx.needSetSubnetsLeasesUsedInfo() == false || len(subnets) == 0 {
		return nil
	}
	var resp *pbdhcpagent.GetSubnetsLeasesCountResponse
	var err error
	if ctx.isUseIds() {
		var ids []uint64
		for _, subnet := range subnets {
			if subnet.Capacity != 0 {
				ids = append(ids, subnet.SubnetId)
			}
		}

		if len(ids) != 0 {
			resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets4LeasesCountWithIds(
				context.TODO(), &pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: ids})
		}
	} else {
		resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets4LeasesCount(
			context.TODO(), &pbdhcpagent.GetSubnetsLeasesCountRequest{})
	}
	if err != nil {
		return err
	}
	subnetsLeasesCount := resp.GetSubnetsLeasesCount()
	for _, subnet := range subnets {
		if subnet.Capacity != 0 {
			if leasesCount, ok := subnetsLeasesCount[subnet.SubnetId]; ok {
				subnet.UsedCount = leasesCount
				subnet.UsedRatio = fmt.Sprintf("%.4f",
					float64(leasesCount)/float64(subnet.Capacity))
			}
		}
	}

	return nil
}

type listSubnetContext struct {
	countSql        string
	sql             string
	params          []interface{}
	hasFilterSubnet bool
	hasPagination   bool
	hasExclude      bool
	hasShared       bool
}

func (l listSubnetContext) isUseIds() bool {
	return l.hasPagination || l.hasFilterSubnet
}

func (l listSubnetContext) needSetSubnetsLeasesUsedInfo() bool {
	return l.hasExclude == false && l.hasShared == false
}

func pbDhcpSubnet4sFromSubnet4List(subnetList []*resource.Subnet4) ([]*pbdhcp.DhcpSubnet4, error) {
	tmpList := make([]*pbdhcp.DhcpSubnet4, 0)
	for _, v := range subnetList {
		tmpList = append(tmpList, parser.EncodeDhcpSubnet4(v))
	}
	return tmpList, nil
}

func getSubnet4FromDB(tx restdb.Transaction, subnetId string) (*resource.Subnet4, error) {
	var subnets []*resource.Subnet4
	if err := tx.Fill(map[string]interface{}{restdb.IDField: subnetId},
		&subnets); err != nil {
		return nil, fmt.Errorf("get subnet %s from db failed: %s",
			subnetId, err.Error())
	}
	if len(subnets) == 0 {
		return nil, fmt.Errorf("get subnet %s is nil", subnetId)
	}
	return subnets[0], nil
}

func loadPool4sLeases(subnet *resource.Subnet4, pools []*resource.Pool4, reservations []*resource.Reservation4) (map[string]uint64, error) {
	resp, err := getSubnet4Leases(subnet.SubnetId)
	if err != nil {
		return nil, fmt.Errorf("get subnet %s leases failed: %s", subnet.GetID(), err.Error())
	}
	if len(resp.GetLeases()) == 0 {
		return nil, nil
	}
	reservationMap := reservationMapFromReservation4s(reservations)
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
	return leasesCount, nil
}

func getSubnet4Leases(subnetId uint64) (*pbdhcpagent.GetLeases4Response, error) {
	return grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Leases(context.TODO(),
		&pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
}

func reservationMapFromReservation4s(reservations []*resource.Reservation4) map[string]string {
	reservationMap := make(map[string]string)
	for _, reservation := range reservations {
		reservationMap[reservation.IpAddress] = reservation.HwAddress
	}
	return reservationMap
}

func setPool4LeasesUsedRatio(pool *resource.Pool4, leasesCount uint64) {
	if leasesCount != 0 && pool.Capacity != 0 {
		pool.UsedCount = leasesCount
		pool.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(pool.Capacity))
	}
}

func pdDhcpPool4FromPool4List(pools []*resource.Pool4) ([]*pbdhcp.DhcpPool4, error) {
	ret := make([]*pbdhcp.DhcpPool4, 0)
	for _, v := range pools {
		ret = append(ret, parser.EncodeDhcpPool4(v))
	}
	return ret, nil
}

func pbDhcpReservedPool4FromReservedPool4(pools []*resource.ReservedPool4) ([]*pbdhcp.DhcpReservedPool4, error) {
	tmpPools := make([]*pbdhcp.DhcpReservedPool4, 0)
	for _, v := range pools {
		tmpPools = append(tmpPools, parser.EncodeDhcpReservedPool4(v))
	}
	return tmpPools, nil
}

func getReservation4sLeasesCount(subnetId uint64, reservations []*resource.Reservation4) (map[string]uint64, error) {
	resp, err := getSubnet4Leases(subnetId)
	if err != nil {
		return nil, fmt.Errorf("get subnet %d leases failed: %s", subnetId, err.Error())
	}
	if len(resp.GetLeases()) == 0 {
		return nil, nil
	}
	reservationMap := reservationMapFromReservation4s(reservations)
	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if mac, ok := reservationMap[lease.GetAddress()]; ok &&
			mac == lease.GetHwAddress() {
			leasesCount[lease.GetAddress()] = 1
		}
	}
	return leasesCount, nil
}

func subnetIDStrToUint64(subnetID string) uint64 {
	id, _ := strconv.ParseUint(subnetID, 10, 64)
	return id
}

func setReservation4LeasesUsedRatio(reservation *resource.Reservation4, leasesCount uint64) {
	if leasesCount != 0 {
		reservation.UsedCount = leasesCount
		reservation.UsedRatio = fmt.Sprintf("%.4f",
			float64(leasesCount)/float64(reservation.Capacity))
	}
}

func pbDchpReservation4FormReservation4(reservations []*resource.Reservation4) ([]*pbdhcp.DhcpReservation4, error) {
	tmpList := make([]*pbdhcp.DhcpReservation4, 0)
	for _, v := range reservations {
		tmpList = append(tmpList, parser.EncodeDhcpReservation4(v))
	}
	return tmpList, nil
}

var ErrorIpNotBelongToSubnet = fmt.Errorf("ip not belongs to subnet")

func getReservation4sAndSubnetLease4s(tx restdb.Transaction, subnetId string) ([]*resource.Reservation4, []*resource.SubnetLease4, error) {
	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnetId},
		&reservations); err != nil {
		return nil, nil, fmt.Errorf("get reservation4s failed: %s", err.Error())
	}
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet4: subnetId},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease4s failed: %s", err.Error())
	}
	return reservations, subnetLeases, nil
}

func getSubnetLease4s(subnetId uint64, reservations []*resource.Reservation4, subnetLeases []*resource.SubnetLease4) ([]*resource.SubnetLease4, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet4Leases(context.TODO(),
		&pbdhcpagent.GetSubnet4LeasesRequest{Id: subnetId})
	if err != nil {
		log.Warnf("get subnet4 %d leases failed: %s", subnetId, err.Error())
		return nil, nil
	}
	reservationMap := reservationMapFromReservation4s(reservations)
	reclaimedSubnetLeases := make(map[string]*resource.SubnetLease4)
	for _, subnetLease := range subnetLeases {
		reclaimedSubnetLeases[subnetLease.Address] = subnetLease
	}
	var leases []*resource.SubnetLease4
	var reclaimleasesForRetain []string
	for _, lease := range resp.GetLeases() {
		lease4 := subnetLease4FromPbLease4AndReservations(lease, reservationMap)
		if reclaimedLease, ok := reclaimedSubnetLeases[lease4.Address]; ok &&
			reclaimedLease.Equal(lease4) {
			reclaimleasesForRetain = append(reclaimleasesForRetain, reclaimedLease.GetID())
			continue
		} else {
			leases = append(leases, lease4)
		}
	}
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Exec("delete from gr_subnet_lease4 where id not in ('" +
			strings.Join(reclaimleasesForRetain, "','") + "')")
		return err
	}); err != nil {
		log.Warnf("delete reclaim leases failed: %s", err.Error())
		return leases, nil
	}
	return leases, nil
}

func subnetLease4FromPbLease4AndReservations(lease *pbdhcpagent.DHCPLease4, reservationMap map[string]string) *resource.SubnetLease4 {
	subnetLease4 := SubnetLease4FromPbLease4(lease)
	if _, ok := reservationMap[subnetLease4.Address]; ok {
		subnetLease4.AddressType = resource.AddressTypeReservation
	}
	return subnetLease4
}

func pbDhcpSubnetLeasesFromSubnetLeases(subnetLeases []*resource.SubnetLease4) ([]*pbdhcp.DhcpSubnetLease4, error) {
	tmpList := make([]*pbdhcp.DhcpSubnetLease4, 0)
	for _, v := range subnetLeases {
		tmpList = append(tmpList, parser.EncodeDhcpSubnetLeases4(v))
	}
	return tmpList, nil
}

func getReservation4sAndSubnetLease4sWithIp(tx restdb.Transaction, subnet4 *resource.Subnet4, ip string) ([]*resource.Reservation4, []*resource.SubnetLease4, error) {
	if subnet4.Ipnet.Contains(net.ParseIP(ip)) == false {
		return nil, nil, ErrorIpNotBelongToSubnet
	}
	var reservations []*resource.Reservation4
	var subnetLeases []*resource.SubnetLease4
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnIpAddress: ip, resource.SqlColumnSubnet4: subnet4.GetID()},
		&reservations); err != nil {
		return nil, nil, fmt.Errorf("get reservation4 %s failed: %s", ip, err.Error())
	}
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnAddress: ip, resource.SqlColumnSubnet4: subnet4.GetID()},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease4 %s failed: %s", ip, err.Error())
	}
	return reservations, subnetLeases, nil
}

func getSubnetLease4sWithIp(subnetId uint64, ip string, reservations []*resource.Reservation4,
	subnetLeases []*resource.SubnetLease4) ([]*resource.SubnetLease4, error) {
	lease4, err := GetSubnetLease4WithoutReclaimed(subnetId, ip, subnetLeases)
	if err != nil {
		return nil, err
	} else if lease4 == nil {
		return nil, fmt.Errorf("get subnetLease4 ret lease4 is nil")
	}
	for _, reservation := range reservations {
		if reservation.IpAddress == lease4.Address {
			lease4.AddressType = resource.AddressTypeReservation
			break
		}
	}
	return []*resource.SubnetLease4{lease4}, nil
}

func setSubnet6sLeasesUsedInfo(subnets []*resource.Subnet6, useIds bool) error {
	if len(subnets) == 0 {
		return nil
	}
	var resp *pbdhcpagent.GetSubnetsLeasesCountResponse
	var err error
	if useIds {
		var ids []uint64
		for _, subnet := range subnets {
			if subnet.Capacity != 0 {
				ids = append(ids, subnet.SubnetId)
			}
		}
		if len(ids) != 0 {
			resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets6LeasesCountWithIds(
				context.TODO(), &pbdhcpagent.GetSubnetsLeasesCountWithIdsRequest{Ids: ids})
		}
	} else {
		resp, err = grpcclient.GetDHCPAgentGrpcClient().GetSubnets6LeasesCount(
			context.TODO(), &pbdhcpagent.GetSubnetsLeasesCountRequest{})
	}
	if err != nil {
		return err
	}
	subnetsLeasesCount := resp.GetSubnetsLeasesCount()
	for _, subnet := range subnets {
		if subnet.Capacity != 0 {
			if leasesCount, ok := subnetsLeasesCount[subnet.SubnetId]; ok {
				subnet.UsedCount = leasesCount
				subnet.UsedRatio = fmt.Sprintf("%.4f",
					float64(leasesCount)/float64(subnet.Capacity))
			}
		}
	}
	return nil
}

func setSubnet6sNodeNames(subnets []*resource.Subnet6, nodeNames map[string]string) {
	for _, subnet := range subnets {
		subnet.NodeNames = getSubnetNodeNames(subnet.Nodes, nodeNames)
	}
}

func pbDhcpSubnet6FromSubnet6(subnets []*resource.Subnet6) ([]*pbdhcp.DhcpSubnet6, error) {
	tmpList := make([]*pbdhcp.DhcpSubnet6, 0)
	for _, v := range subnets {
		tmpList = append(tmpList, parser.EncodeDhcpSubnet6(v))
	}
	return tmpList, nil
}

func pbDhcpFromSubnet6s(subnets []*resource.Subnet6) ([]*pbdhcp.DhcpSubnet6, error) {
	tmpList := make([]*pbdhcp.DhcpSubnet6, 0)
	for _, v := range subnets {
		tmpList = append(tmpList, parser.EncodeDhcpSubnet6(v))
	}
	return tmpList, nil
}

func getSubnet6FromDB(tx restdb.Transaction, subnetId string) (*resource.Subnet6, error) {
	var subnets []*resource.Subnet6
	if err := tx.Fill(map[string]interface{}{restdb.IDField: subnetId},
		&subnets); err != nil {
		return nil, fmt.Errorf("get subnet %s from db failed: %s", subnetId, err.Error())
	}
	if len(subnets) == 0 {
		return nil, fmt.Errorf("no found subnet %s", subnetId)
	}
	return subnets[0], nil
}

func loadPool6sLeases(subnet *resource.Subnet6, pools []*resource.Pool6, reservations []*resource.Reservation6) (map[string]uint64, error) {
	resp, err := getSubnet6Leases(subnet.SubnetId)
	if err != nil {
		return nil, fmt.Errorf("get subnet %s leases failed: %s", subnet.GetID(), err.Error())
	}
	if len(resp.GetLeases()) == 0 {
		return nil, nil
	}

	reservationMap := reservationMapFromReservation6s(reservations)
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
	return leasesCount, nil
}

func getSubnet6Leases(subnetId uint64) (*pbdhcpagent.GetLeases6Response, error) {
	return grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Leases(context.TODO(),
		&pbdhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
}

func setPool6LeasesUsedRatio(pool *resource.Pool6, leasesCount uint64) {
	if leasesCount != 0 && pool.Capacity != 0 {
		pool.UsedCount = leasesCount
		pool.UsedRatio = fmt.Sprintf("%.4f", float64(leasesCount)/float64(pool.Capacity))
	}
}

func pbDhcpPoolsFromPools(pools []*resource.Pool6) ([]*pbdhcp.DhcpPool6, error) {
	tmpList := make([]*pbdhcp.DhcpPool6, 0)
	for _, v := range pools {
		tmpList = append(tmpList, parser.EncodeDhcpPool6(v))
	}
	return tmpList, nil
}

func pbDhcpReservedPool6FromReservedPool6(pools []*resource.ReservedPool6) ([]*pbdhcp.DhcpReservedPool6, error) {
	tmpList := make([]*pbdhcp.DhcpReservedPool6, 0)
	for _, v := range pools {
		tmpList = append(tmpList, parser.EncodeDhcpReservedPool6(v))
	}
	return tmpList, nil
}

func getReservation6sLeasesCount(subnetId uint64, reservations []*resource.Reservation6) (map[string]uint64, error) {
	resp, err := getSubnet6Leases(subnetId)
	if err != nil {
		return nil, fmt.Errorf("get subnet %d leases failed: %s", subnetId, err.Error())
	}

	if len(resp.GetLeases()) == 0 {
		return nil, nil
	}
	reservationMap := make(map[string]*resource.Reservation6)
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress] = reservation
		}
	}
	leasesCount := make(map[string]uint64)
	for _, lease := range resp.GetLeases() {
		if reservation, ok := reservationMap[lease.GetAddress()]; ok {
			count := leasesCount[reservation.GetID()]
			if (len(reservation.Duid) != 0 && reservation.Duid == lease.GetDuid()) ||
				(len(reservation.HwAddress) != 0 &&
					reservation.HwAddress == lease.GetHwAddress()) {
				count += 1
			}
			leasesCount[reservation.GetID()] = count
		}
	}
	return leasesCount, nil
}

func reservationMapFromReservation6s(reservations []*resource.Reservation6) map[string]struct{} {
	reservationMap := make(map[string]struct{})
	for _, reservation := range reservations {
		for _, ipAddress := range reservation.IpAddresses {
			reservationMap[ipAddress] = struct{}{}
		}
	}
	return reservationMap
}

func setReservation6LeasesUsedRatio(reservation *resource.Reservation6, leasesCount uint64) {
	if leasesCount != 0 {
		reservation.UsedCount = leasesCount
		reservation.UsedRatio = fmt.Sprintf("%.4f",
			float64(leasesCount)/float64(reservation.Capacity))
	}
}

func pbDhcpReservation6FromReservation6(reservations []*resource.Reservation6) ([]*pbdhcp.DhcpReservation6, error) {
	tmpList := make([]*pbdhcp.DhcpReservation6, 0)
	for _, v := range reservations {
		tmpList = append(tmpList, parser.EncodeDhcpReservation6(v))
	}
	return tmpList, nil
}

func getSubnetLease6s(subnetId uint64, reservations []*resource.Reservation6, subnetLeases []*resource.SubnetLease6) ([]*resource.SubnetLease6, error) {
	resp, err := grpcclient.GetDHCPAgentGrpcClient().GetSubnet6Leases(context.TODO(),
		&pbdhcpagent.GetSubnet6LeasesRequest{Id: subnetId})
	if err != nil {
		return nil, fmt.Errorf("get subnet6 %d leases failed: %s", subnetId, err.Error())
	}

	reservationMap := reservationMapFromReservation6s(reservations)
	reclaimedSubnetLeases := make(map[string]*resource.SubnetLease6)
	for _, subnetLease := range subnetLeases {
		reclaimedSubnetLeases[subnetLease.Address] = subnetLease
	}
	var leases []*resource.SubnetLease6
	var reclaimleasesForRetain []string
	for _, lease := range resp.GetLeases() {
		lease6 := subnetLease6FromPbLease6AndReservations(lease, reservationMap)
		if reclaimedLease, ok := reclaimedSubnetLeases[lease6.Address]; ok &&
			reclaimedLease.Equal(lease6) {
			reclaimleasesForRetain = append(reclaimleasesForRetain, reclaimedLease.GetID())
		} else {
			leases = append(leases, lease6)
		}
	}
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		_, err := tx.Exec("delete from gr_subnet_lease6 where id not in ('" +
			strings.Join(reclaimleasesForRetain, "','") + "')")
		return err
	}); err != nil {
		return nil, fmt.Errorf("delete reclaim leases failed: %s", err.Error())
	}
	return leases, nil
}

func subnetLease6FromPbLease6AndReservations(lease *pbdhcpagent.DHCPLease6, reservationMap map[string]struct{}) *resource.SubnetLease6 {
	subnetLease6 := SubnetLease6FromPbLease6(lease)
	if _, ok := reservationMap[subnetLease6.Address]; ok {
		subnetLease6.AddressType = resource.AddressTypeReservation
	}
	return subnetLease6
}

func dpDhcpSubnetLease6FromSubnetLease6(subnetLeases []*resource.SubnetLease6) ([]*pbdhcp.DhcpSubnetLease6, error) {
	tmpList := make([]*pbdhcp.DhcpSubnetLease6, 0)
	for _, v := range subnetLeases {
		tmpList = append(tmpList, parser.EncodeDhcpSubnetLease6(v))
	}
	return tmpList, nil
}

func getReservation6sAndSubnetLease6sWithIp(tx restdb.Transaction, subnet6 *resource.Subnet6, ip string) ([]*resource.Reservation6, []*resource.SubnetLease6, error) {
	if subnet6.Ipnet.Contains(net.ParseIP(ip)) == false {
		return nil, nil, ErrorIpNotBelongToSubnet
	}
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := tx.FillEx(&reservations,
		"select * from gr_reservation6 where subnet6 = $1 and $2::text = any(ip_addresses)",
		subnet6.GetID(), ip); err != nil {
		return nil, nil, fmt.Errorf("get reservation6 %s failed: %s", ip, err.Error())
	}
	if err := tx.Fill(map[string]interface{}{
		resource.SqlColumnAddress: ip,
		resource.SqlColumnSubnet6: subnet6.GetID()},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease6 %s failed: %s", ip, err.Error())
	}
	return reservations, subnetLeases, nil
}

func getReservation6sAndSubnetLease6s(tx restdb.Transaction, subnetId string) ([]*resource.Reservation6, []*resource.SubnetLease6, error) {
	var reservations []*resource.Reservation6
	var subnetLeases []*resource.SubnetLease6
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&reservations); err != nil {
		return nil, nil, fmt.Errorf("get reservation6s failed: %s", err.Error())
	}
	if err := tx.Fill(map[string]interface{}{resource.SqlColumnSubnet6: subnetId},
		&subnetLeases); err != nil {
		return nil, nil, fmt.Errorf("get subnet lease6s failed: %s", err.Error())
	}
	return reservations, subnetLeases, nil
}

func getSubnetLease6sWithIp(subnetId uint64, ip string, reservations []*resource.Reservation6, subnetLeases []*resource.SubnetLease6) ([]*resource.SubnetLease6, error) {
	lease6, err := GetSubnetLease6WithoutReclaimed(subnetId, ip,
		subnetLeases)
	if err != nil {
		log.Debugf("get subnet6 %d leases failed: %s", subnetId, err.Error())
		return nil, nil
	} else if lease6 == nil {
		return nil, nil
	}
	for _, reservation := range reservations {
		for _, ipaddress := range reservation.IpAddresses {
			if ipaddress == lease6.Address {
				lease6.AddressType = resource.AddressTypeReservation
				break
			}
		}
		if lease6.AddressType == resource.AddressTypeReservation {
			break
		}
	}
	return []*resource.SubnetLease6{lease6}, nil
}
