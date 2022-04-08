package service

import (
	"bytes"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/linkingthing/cement/uuid"

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
	FieldNameRelayAddresses    = "中继路由地址"
	FieldNameOption16          = "option16"
	FieldNameOption18          = "option18"
	FieldNameNodes             = "节点列表"
	FieldNameEUI64             = "EUI64"

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
		FieldNameOption60, FieldNameOption82, FieldNameOption66, FieldNameOption67,
		FieldNameNodes, FieldNamePools, FieldNameReservedPools, FieldNameReservations,
	}

	TableHeaderSubnet6 = []string{
		FieldNameSubnet, FieldNameSubnetName, FieldNameEUI64,
		FieldNameValidLifetime, FieldNameMaxValidLifetime, FieldNameMinValidLifetime,
		FieldNamePreferredLifetime, FieldNameDomainServers, FieldNameIfaceName,
		FieldNameRelayAddresses, FieldNameOption16, FieldNameOption18, FieldNameNodes,
		FieldNamePools, FieldNameReservedPools, FieldNameReservations, FieldNamePdPools,
	}

	SubnetMandatoryFields = []string{FieldNameSubnet}
	TableHeaderSubnet4Len = len(TableHeaderSubnet4)
	TableHeaderSubnet6Len = len(TableHeaderSubnet6)

	TemplateSubnet4 = [][]string{[]string{
		"1.0.0.0/24", "template", "14400", "28800", "7200", "255.255.255.0", "1.0.0.1",
		"114.114.114.114,8.8.8.8", "ens33", "option60", "1.0.0.255",
		"http://linkingthing.com/tftp", "tftp.bin", "1.0.0.2,1.0.0.3",
		"1.0.0.6-1.0.0.100-备注1,1.0.0.106-1.0.0.200-备注2",
		"1.0.0.1-1.0.0.5-备注3,1.0.0.200-1.0.0.255-备注4",
		"11:11:11:11:11:11-1.0.0.66-备注5,22:22:22:22:22:22-1.0.0.101-备注6",
	}}

	TemplateSubnet6 = [][]string{
		[]string{"2001::/32", "template1", "关闭", "14400", "28800", "7200", "14400",
			"2400:3200::1,2400:3200::baba:1", "ens33", "2001::255", "option16",
			"Gi0/0/1", "1.0.0.2,1.0.0.3", "", "", "",
			"2001:0:2001::-48-64-备注1,2001:0:2002::-48-64-备注2"},
		[]string{"2002::/64", "template2", "关闭", "14400", "28800", "7200", "14400",
			"2400:3200::1,2400:3200::baba:1", "ens33", "2002::255", "option16",
			"Gi0/0/2", "1.0.0.3,1.0.0.4",
			"2002::6-2002::1f-备注1,2002::26-2002::3f-备注2",
			"2002::1-2002::5-备注3,2002::20-2002::25-备注4",
			"duid-0102-ips-2002::11_2002::12-备注5, mac-33:33:33:33:33:33-ips-2002::32_2002::33-备注6",
			""},
		[]string{"2003::/64", "template3", "开启", "14400", "28800", "7200", "14400",
			"2400:3200::1,2400:3200::baba:1", "ens33", "2001::255", "option16", "Gi0/0/3",
			"1.0.0.4,1.0.0.5", "", "", "", ""},
	}
)

func localizationSubnet4ToStrSlice(subnet4 *resource.Subnet4, virtualIp string) []string {
	subnetNodes := virtualIp
	if virtualIp == "" {
		subnetNodes = strings.Join(subnet4.Nodes, ",")
	}

	return []string{
		subnet4.Subnet, subnet4.Tags,
		lifetimeToString(subnet4.ValidLifetime),
		lifetimeToString(subnet4.MaxValidLifetime),
		lifetimeToString(subnet4.MinValidLifetime),
		subnet4.SubnetMask, strings.Join(subnet4.Routers, ","),
		strings.Join(subnet4.DomainServers, ","), subnet4.IfaceName,
		subnet4.ClientClass, strings.Join(subnet4.RelayAgentAddresses, ","),
		subnet4.TftpServer, subnet4.Bootfile, subnetNodes,
	}
}

func localizationSubnet6ToStrSlice(subnet6 *resource.Subnet6, virtualIp string) []string {
	subnetNodes := virtualIp
	if virtualIp == "" {
		subnetNodes = strings.Join(subnet6.Nodes, ",")
	}

	return []string{
		subnet6.Subnet, subnet6.Tags, eui64ToString(subnet6.UseEui64),
		lifetimeToString(subnet6.ValidLifetime),
		lifetimeToString(subnet6.MaxValidLifetime),
		lifetimeToString(subnet6.MinValidLifetime),
		lifetimeToString(subnet6.PreferredLifetime),
		strings.Join(subnet6.DomainServers, ","), subnet6.IfaceName,
		strings.Join(subnet6.RelayAgentAddresses, ","), subnet6.ClientClass,
		subnet6.RelayAgentInterfaceId, subnetNodes,
	}
}

func eui64ToString(eui64 bool) string {
	if eui64 {
		return "开启"
	} else {
		return "关闭"
	}
}

func eui64FromString(eui64 string) bool {
	if eui64 == "开启" {
		return true
	} else {
		return false
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
	buf.WriteString(pool4.BeginIp.String())
	buf.WriteString("','")
	buf.WriteString(pool4.EndAddress)
	buf.WriteString("','")
	buf.WriteString(pool4.EndIp.String())
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(pool4.Capacity, 10))
	buf.WriteString("','")
	buf.WriteString(pool4.Comment)
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
	buf.WriteString(pool4.BeginIp.String())
	buf.WriteString("','")
	buf.WriteString(pool4.EndAddress)
	buf.WriteString("','")
	buf.WriteString(pool4.EndIp.String())
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(pool4.Capacity, 10))
	buf.WriteString("','")
	buf.WriteString(pool4.Comment)
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
	buf.WriteString(reservation4.Ip.String())
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(reservation4.Capacity, 10))
	buf.WriteString("','")
	buf.WriteString(reservation4.Comment)
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(subnetId, 10))
	buf.WriteString("'),")
	return buf.String()
}

func subnet6ToInsertDBSqlString(subnet6 *resource.Subnet6) string {
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(subnet6.GetID())
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(subnet6.Subnet)
	buf.WriteString("','")
	buf.WriteString(subnet6.Ipnet.String())
	buf.WriteString("','")
	buf.WriteString(subnet6.GetID())
	buf.WriteString("','")
	buf.WriteString(lifetimeToString(subnet6.ValidLifetime))
	buf.WriteString("','")
	buf.WriteString(lifetimeToString(subnet6.MaxValidLifetime))
	buf.WriteString("','")
	buf.WriteString(lifetimeToString(subnet6.MinValidLifetime))
	buf.WriteString("','")
	buf.WriteString(lifetimeToString(subnet6.PreferredLifetime))
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet6.DomainServers, ","))
	buf.WriteString("}','")
	buf.WriteString(subnet6.ClientClass)
	buf.WriteString("','")
	buf.WriteString(subnet6.IfaceName)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet6.RelayAgentAddresses, ","))
	buf.WriteString("}','")
	buf.WriteString(subnet6.RelayAgentInterfaceId)
	buf.WriteString("','")
	buf.WriteString(subnet6.Tags)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet6.Nodes, ","))
	buf.WriteString("}','")
	if subnet6.RapidCommit {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}
	buf.WriteString("','")
	if subnet6.UseEui64 {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(subnet6.Capacity, 10))
	buf.WriteString("'),")
	return buf.String()
}

func pool6ToInsertDBSqlString(subnetId uint64, pool6 *resource.Pool6) string {
	id, _ := uuid.Gen()
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(id)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(pool6.BeginAddress)
	buf.WriteString("','")
	buf.WriteString(pool6.BeginIp.String())
	buf.WriteString("','")
	buf.WriteString(pool6.EndAddress)
	buf.WriteString("','")
	buf.WriteString(pool6.EndIp.String())
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(pool6.Capacity, 10))
	buf.WriteString("','")
	buf.WriteString(pool6.Comment)
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(subnetId, 10))
	buf.WriteString("'),")
	return buf.String()
}

func reservedPool6ToInsertDBSqlString(subnetId uint64, pool6 *resource.ReservedPool6) string {
	id, _ := uuid.Gen()
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(id)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(pool6.BeginAddress)
	buf.WriteString("','")
	buf.WriteString(pool6.BeginIp.String())
	buf.WriteString("','")
	buf.WriteString(pool6.EndAddress)
	buf.WriteString("','")
	buf.WriteString(pool6.EndIp.String())
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(pool6.Capacity, 10))
	buf.WriteString("','")
	buf.WriteString(pool6.Comment)
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(subnetId, 10))
	buf.WriteString("'),")
	return buf.String()
}

func reservation6ToInsertDBSqlString(subnetId uint64, reservation6 *resource.Reservation6) string {
	id, _ := uuid.Gen()
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(id)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(reservation6.Duid)
	buf.WriteString("','")
	buf.WriteString(reservation6.HwAddress)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(reservation6.IpAddresses, ","))
	buf.WriteString("}','{")
	buf.WriteString(ipsToString(reservation6.Ips))
	buf.WriteString("}','{")
	buf.WriteString(strings.Join(reservation6.Prefixes, ","))
	buf.WriteString("}','")
	buf.WriteString(strconv.FormatUint(reservation6.Capacity, 10))
	buf.WriteString("','")
	buf.WriteString(reservation6.Comment)
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(subnetId, 10))
	buf.WriteString("'),")
	return buf.String()
}

func ipsToString(ips []net.IP) string {
	var ipstrs []string
	for _, ip := range ips {
		ipstrs = append(ipstrs, ip.String())
	}

	return strings.Join(ipstrs, ",")
}

func pdpoolToInsertDBSqlString(subnetId uint64, pdpool *resource.PdPool) string {
	id, _ := uuid.Gen()
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(id)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(pdpool.Prefix)
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(uint64(pdpool.PrefixLen), 10))
	buf.WriteString("','")
	buf.WriteString(pdpool.PrefixIpnet.String())
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(uint64(pdpool.DelegatedLen), 10))
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(pdpool.Capacity, 10))
	buf.WriteString("','")
	buf.WriteString(pdpool.Comment)
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(subnetId, 10))
	buf.WriteString("'),")
	return buf.String()
}
