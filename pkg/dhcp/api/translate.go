package api

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/zdnscloud/cement/uuid"

	"github.com/linkingthing/clxone-dhcp/pkg/dhcp/resource"
)

const (
	FieldNameSubnet            = "子网地址*"
	FieldNameSubnetName        = "子网名称"
	FieldNameValidLifetime     = "租约时长"
	FieldNameMaxValidLifetime  = "最大租约时长"
	FieldNameMinValidLifetime  = "最小租约时长"
	FieldNamePreferredLifetime = "首选租约时长"
	FieldNameSubnetMask        = "子网掩码"
	FieldNameRouters           = "默认网关"
	FieldNameDomainServers     = "DNS"
	FieldNameIfaceName         = "网卡名字"
	FieldNameOption60          = "option60"
	FieldNameOption82          = "option82"
	FieldNameOption66          = "option66"
	FieldNameOption67          = "option67"
	FiledNameRelayAddresses    = "中继路由地址"
	FieldNameOption16          = "option16"
	FieldNameOption18          = "option18"
	FieldNameNodes             = "节点列表"

	FieldNamePools         = "动态地址池"
	FieldNameReservedPools = "保留地址池"
	FieldNameReservations  = "固定地址池"
	FieldNamePdPools       = "前缀委派地址池"
)

var (
	TableHeaderSubnet4 = []string{
		FieldNameSubnet, FieldNameSubnetName,
		FieldNameValidLifetime, FieldNameMaxValidLifetime, FieldNameMinValidLifetime,
		FieldNameSubnetMask, FieldNameRouters, FieldNameDomainServers, FieldNameIfaceName,
		FieldNameOption60, FieldNameOption82, FieldNameOption66, FieldNameOption67, FieldNameNodes,
		FieldNamePools, FieldNameReservedPools, FieldNameReservations,
	}

	TableHeaderSubnet6 = []string{
		FieldNameSubnet, FieldNameSubnetName,
		FieldNameValidLifetime, FieldNameMaxValidLifetime, FieldNameMinValidLifetime,
		FieldNamePreferredLifetime, FieldNameDomainServers, FieldNameIfaceName,
		FiledNameRelayAddresses, FieldNameOption16, FieldNameOption18, FieldNameNodes,
		FieldNamePools, FieldNameReservedPools, FieldNameReservations, FieldNamePdPools,
	}

	SubnetMandatoryFields = []string{FieldNameSubnet}
	TableHeaderSubnet4Len = len(TableHeaderSubnet4)
)

func localizationSubnet4ToStrSlice(subnet4 *resource.Subnet4) []string {
	return []string{
		subnet4.Subnet, subnet4.Tags,
		lifetimeToString(subnet4.ValidLifetime),
		lifetimeToString(subnet4.MaxValidLifetime),
		lifetimeToString(subnet4.MinValidLifetime),
		subnet4.SubnetMask, strings.Join(subnet4.Routers, ","),
		strings.Join(subnet4.DomainServers, ","), subnet4.IfaceName,
		subnet4.ClientClass, strings.Join(subnet4.RelayAgentAddresses, ","),
		subnet4.TftpServer, subnet4.Bootfile, strings.Join(subnet4.Nodes, ","),
	}
}

func lifetimeToString(lifetime uint32) string {
	return strconv.FormatUint(uint64(lifetime), 10)
}

func subnet4ToInsertDBSqlString(subnet4 *resource.Subnet4) string {
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(subnet4.GetID())
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(subnet4.Subnet)
	buf.WriteString("','")
	buf.WriteString(subnet4.Ipnet.String())
	buf.WriteString("','")
	buf.WriteString(subnet4.GetID())
	buf.WriteString("','")
	buf.WriteString(lifetimeToString(subnet4.ValidLifetime))
	buf.WriteString("','")
	buf.WriteString(lifetimeToString(subnet4.MaxValidLifetime))
	buf.WriteString("','")
	buf.WriteString(lifetimeToString(subnet4.MinValidLifetime))
	buf.WriteString("','")
	buf.WriteString(subnet4.SubnetMask)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet4.DomainServers, ","))
	buf.WriteString("}','{")
	buf.WriteString(strings.Join(subnet4.Routers, ","))
	buf.WriteString("}','")
	buf.WriteString(subnet4.ClientClass)
	buf.WriteString("','")
	buf.WriteString(subnet4.TftpServer)
	buf.WriteString("','")
	buf.WriteString(subnet4.Bootfile)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet4.RelayAgentAddresses, ","))
	buf.WriteString("}','")
	buf.WriteString(subnet4.IfaceName)
	buf.WriteString("','")
	buf.WriteString(subnet4.NextServer)
	buf.WriteString("','")
	buf.WriteString(subnet4.Tags)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet4.Nodes, ","))
	buf.WriteString("}','")
	buf.WriteString(strconv.FormatUint(subnet4.Capacity, 10))
	buf.WriteString("'),")
	return buf.String()
}

func pool4ToInsertDBSqlString(subnetId uint64, pool4 *resource.Pool4) string {
	id, _ := uuid.Gen()
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(id)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(pool4.BeginAddress)
	buf.WriteString("','")
	buf.WriteString(pool4.EndAddress)
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(pool4.Capacity, 10))
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(subnetId, 10))
	buf.WriteString("'),")
	return buf.String()
}

func reservedPool4ToInsertDBSqlString(subnetId uint64, pool4 *resource.ReservedPool4) string {
	id, _ := uuid.Gen()
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(id)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(pool4.BeginAddress)
	buf.WriteString("','")
	buf.WriteString(pool4.EndAddress)
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(pool4.Capacity, 10))
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(subnetId, 10))
	buf.WriteString("'),")
	return buf.String()
}

func reservation4ToInsertDBSqlString(subnetId uint64, reservation4 *resource.Reservation4) string {
	id, _ := uuid.Gen()
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(id)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(reservation4.HwAddress)
	buf.WriteString("','")
	buf.WriteString(reservation4.IpAddress)
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(reservation4.Capacity, 10))
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(subnetId, 10))
	buf.WriteString("'),")
	return buf.String()
}
