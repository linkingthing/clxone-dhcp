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
	FieldNameSubnet                   = "子网地址*"
	FieldNameSubnetName               = "子网名称"
	FieldNameValidLifetime            = "租约时长"
	FieldNameMaxValidLifetime         = "最长租约时长"
	FieldNameMinValidLifetime         = "最短租约时长"
	FieldNamePreferredLifetime        = "首选租约时长"
	FieldNameSubnetMask               = "子网掩码"
	FieldNameRouters                  = "默认网关"
	FieldNameDomainServers            = "DNS"
	FieldNameIfaceName                = "网卡名字"
	FieldNameOption66                 = "TFTP服务地址"
	FieldNameOption67                 = "启动文件名称"
	FieldNameOption108                = "IPv6-Only时长"
	FieldNameOption138                = "AC地址列表"
	FieldNameRelayCircuitId           = "中继电路标识"
	FieldNameRelayRemoteId            = "中继远程标识"
	FieldNameRelayAddresses           = "中继路由地址"
	FieldNameOption18                 = "中继网卡名称"
	FieldNameOption32                 = "信息刷新时长"
	FieldNameOption52                 = "AC地址列表"
	FieldNameNodes                    = "节点列表"
	FieldNameEUI64                    = "EUI64"
	FieldNameEmbedIPv4                = "嵌入IPv4"
	FieldNameAddressCode              = "地址编码"
	FieldNameWhiteClientClassStrategy = "白名单策略"
	FieldNameWhiteClientClasses       = "白名单"
	FieldNameBlackClientClassStrategy = "黑名单策略"
	FieldNameBlackClientClasses       = "黑名单"
	FieldNameNextServer               = "启动服务地址"

	FieldNamePools         = "动态地址池"
	FieldNameReservedPools = "保留地址池"
	FieldNameReservations  = "固定地址池"
	FieldNamePdPools       = "前缀委派"

	FieldNameAssetName         = "资产名称*"
	FieldNameHwAddress         = "资产MAC*"
	FieldNameAssetType         = "资产类型"
	FieldNameManufacturer      = "资产厂商"
	FieldNameModel             = "资产型号"
	FieldNameOperatingSystem   = "操作系统"
	FieldNameAccessNetworkTime = "入网时间"

	FieldNameCode  = "编码值（十六进制）*"
	FieldNameValue = "编码用途*"

	FailReasonLocalization = "失败原因"
)

var (
	TableHeaderSubnet4 = []string{
		FieldNameSubnet, FieldNameSubnetName, FieldNameIfaceName,
		FieldNameWhiteClientClassStrategy, FieldNameWhiteClientClasses,
		FieldNameBlackClientClassStrategy, FieldNameBlackClientClasses,
		FieldNameValidLifetime, FieldNameMaxValidLifetime, FieldNameMinValidLifetime,
		FieldNameSubnetMask, FieldNameRouters, FieldNameDomainServers,
		FieldNameOption66, FieldNameOption67,
		FieldNameRelayCircuitId, FieldNameRelayRemoteId, FieldNameRelayAddresses,
		FieldNameOption108, FieldNameOption138, FieldNameNodes, FieldNameNextServer,
		FieldNamePools, FieldNameReservedPools, FieldNameReservations,
	}

	TableHeaderSubnet6 = []string{
		FieldNameSubnet, FieldNameSubnetName,
		FieldNameEUI64, FieldNameEmbedIPv4,
		FieldNameAddressCode, FieldNameIfaceName,
		FieldNameWhiteClientClassStrategy, FieldNameWhiteClientClasses,
		FieldNameBlackClientClassStrategy, FieldNameBlackClientClasses,
		FieldNameValidLifetime, FieldNameMaxValidLifetime, FieldNameMinValidLifetime,
		FieldNamePreferredLifetime, FieldNameOption18, FieldNameDomainServers,
		FieldNameOption32, FieldNameOption52, FieldNameRelayAddresses, FieldNameNodes,
		FieldNamePools, FieldNameReservedPools, FieldNameReservations, FieldNamePdPools,
	}

	TableHeaderAsset = []string{
		FieldNameAssetName, FieldNameHwAddress, FieldNameAssetType, FieldNameManufacturer,
		FieldNameModel, FieldNameOperatingSystem, FieldNameAccessNetworkTime,
	}

	TableHeaderSegment = []string{FieldNameValue, FieldNameCode}

	TableHeaderSubnet4Fail = append(TableHeaderSubnet4, FailReasonLocalization)
	TableHeaderSubnet6Fail = append(TableHeaderSubnet6, FailReasonLocalization)
	TableHeaderAssetFail   = append(TableHeaderAsset, FailReasonLocalization)
	TableHeaderSegmentFail = append(TableHeaderSegment, FailReasonLocalization)

	SubnetMandatoryFields  = []string{FieldNameSubnet}
	AssetMandatoryFields   = []string{FieldNameAssetName, FieldNameHwAddress}
	SegmentMandatoryFields = []string{FieldNameValue, FieldNameCode}

	TableHeaderSubnet4Len     = len(TableHeaderSubnet4)
	TableHeaderSubnet6Len     = len(TableHeaderSubnet6)
	TableHeaderAssetLen       = len(TableHeaderAsset)
	TableHeaderSegmentLen     = len(TableHeaderSegment)
	TableHeaderSubnet4FailLen = len(TableHeaderSubnet4Fail)
	TableHeaderSubnet6FailLen = len(TableHeaderSubnet6Fail)
	TableHeaderAssetFailLen   = len(TableHeaderAssetFail)
	TableHeaderSegmentFailLen = len(TableHeaderSegmentFail)

	TemplateSubnet4 = [][]string{{
		"127.0.0.0/8", "template", "ens33",
		"满足全部", "option60\noption61",
		"满足一个", "option3\noption6",
		"14400", "28800", "7200", "255.0.0.0", "127.0.0.1", "114.114.114.114\n8.8.8.8",
		"linkingthing", "tftp.bin", "Gi1/1/1", "11:11:11:11:11:11", "127.0.0.1",
		"1800", "127.0.0.1\n127.0.0.2", "127.0.0.2\n127.0.0.3", "127.0.0.1",
		"127.0.0.6-127.0.0.100-备注1\n127.0.0.106-127.0.0.200-备注2",
		"127.0.0.1-127.0.0.5-备注3\n127.0.0.200-127.0.0.255-备注4",
		"mac$11:11:11:11:11:11$127.0.0.66$备注5\nhostname$linking$127.0.0.101$备注6",
	}}

	TemplateSubnet6 = [][]string{
		[]string{"2001::/32", "template1", "关闭", "关闭", "", "ens33",
			"满足全部", "option60\noption61",
			"满足一个", "option3\noption6",
			"14400", "28800", "7200", "14400",
			"Gi0/0/1", "2400:3200::1\n2400:3200::baba:1", "3600", "2001::255",
			"2001::1\n2001::2", "127.0.0.2\n127.0.0.3", "", "", "",
			"2001:0:2001::-48-64-备注1\n2001:0:2002::-48-64-备注2"},
		[]string{"2002::/64", "template2", "关闭", "关闭", "", "eno1",
			"满足全部", "option16-1",
			"满足一个", "option17-1",
			"14400", "28800", "7200", "14400",
			"Gi0/0/2", "2400:3200::1", "3600", "2002::255",
			"2002::1\n2002::2", "127.0.0.3\n127.0.0.4",
			"2002::6-2002::1f-备注1\n2002::26-2002::3f-备注2",
			"2002::1-2002::5-备注3\n2002::20-2002::25-备注4",
			"duid$0102$ips$2002::11_2002::12$备注5\nmac$33:33:33:33:33:33$ips$2002::32_2002::33$备注6\nhostname$linking$ips$2002::34_2002::35$备注7",
			""},
		[]string{"2003::/64", "template3", "开启", "关闭", "", "eth0",
			"满足全部", "option16-2",
			"满足一个", "option17-2",
			"14400", "28800", "7200", "14400",
			"Gi0/0/3", "2400:3200::baba:1", "3600", "2003::255",
			"2003::1\n2003::2", "127.0.0.4\n127.0.0.5", "", "", "", ""},
		[]string{"2004::/64", "template3", "关闭", "开启", "", "eth0",
			"满足全部", "option16-2",
			"满足一个", "option17-2",
			"14400", "28800", "7200", "14400",
			"Gi0/0/3", "2400:3200::baba:1", "3600", "2004::255",
			"2004::1\n2004::2", "127.0.0.4\n127.0.0.5", "", "", "", ""},
		[]string{"2005::/64", "template4", "关闭", "关闭", "a1", "eth0",
			"满足全部", "option16-3",
			"满足一个", "option17-3",
			"14400", "28800", "7200", "14400",
			"Gi0/0/3", "2400:3200::baba:1", "3600", "2005::255",
			"2005::1\n2005::2", "127.0.0.4\n127.0.0.5", "", "", "", ""},
	}

	TemplateAsset = [][]string{
		[]string{"a1", "11:11:11:11:11:11", "mobile", "huawei", "p40", "android", "2023-10-31"},
		[]string{"a2", "22:22:22:22:22:22", "pc", "huawei", "matebook pro", "windows11", "2023-10-31"},
	}

	TemplateSegment = [][]string{
		[]string{"手机", "01"},
		[]string{"缺省", "ff"},
	}
)

func splitFieldWithoutSpace(field string) []string {
	return strings.Split(strings.Replace(strings.TrimSpace(field), " ", "", -1), resource.CommonDelimiter)
}

func localizationSubnet4ToStrSlice(subnet4 *resource.Subnet4) []string {
	return []string{
		subnet4.Subnet, subnet4.Tags, subnet4.IfaceName,
		localizationClientClassStrategy(subnet4.WhiteClientClassStrategy),
		strings.Join(subnet4.WhiteClientClasses, resource.CommonDelimiter),
		localizationClientClassStrategy(subnet4.BlackClientClassStrategy),
		strings.Join(subnet4.BlackClientClasses, resource.CommonDelimiter),
		uint32ToString(subnet4.ValidLifetime),
		uint32ToString(subnet4.MaxValidLifetime),
		uint32ToString(subnet4.MinValidLifetime),
		subnet4.SubnetMask, strings.Join(subnet4.Routers, resource.CommonDelimiter),
		strings.Join(subnet4.DomainServers, resource.CommonDelimiter),
		subnet4.TftpServer, subnet4.Bootfile,
		subnet4.RelayAgentCircuitId, subnet4.RelayAgentRemoteId,
		strings.Join(subnet4.RelayAgentAddresses, resource.CommonDelimiter),
		uint32ToString(subnet4.Ipv6OnlyPreferred),
		strings.Join(subnet4.CapWapACAddresses, resource.CommonDelimiter),
		strings.Join(subnet4.Nodes, resource.CommonDelimiter), subnet4.NextServer,
	}
}

func localizationSubnet6ToStrSlice(subnet6 *resource.Subnet6) []string {
	return []string{
		subnet6.Subnet, subnet6.Tags,
		localizationBoolSwitch(subnet6.EmbedIpv4),
		localizationBoolSwitch(subnet6.UseEui64),
		subnet6.AddressCodeName, subnet6.IfaceName,
		localizationClientClassStrategy(subnet6.WhiteClientClassStrategy),
		strings.Join(subnet6.WhiteClientClasses, resource.CommonDelimiter),
		localizationClientClassStrategy(subnet6.BlackClientClassStrategy),
		strings.Join(subnet6.BlackClientClasses, resource.CommonDelimiter),
		uint32ToString(subnet6.ValidLifetime),
		uint32ToString(subnet6.MaxValidLifetime),
		uint32ToString(subnet6.MinValidLifetime),
		uint32ToString(subnet6.PreferredLifetime),
		subnet6.RelayAgentInterfaceId,
		strings.Join(subnet6.DomainServers, resource.CommonDelimiter),
		uint32ToString(subnet6.InformationRefreshTime),
		strings.Join(subnet6.CapWapACAddresses, resource.CommonDelimiter),
		strings.Join(subnet6.RelayAgentAddresses, resource.CommonDelimiter),
		strings.Join(subnet6.Nodes, resource.CommonDelimiter),
	}
}

func localizationAssetToStrSlice(asset *resource.Asset) []string {
	return []string{
		asset.Name, asset.HwAddress, asset.AssetType,
		asset.Manufacturer, asset.Model, asset.OperatingSystem,
		asset.AccessNetworkTime,
	}
}

func localizationSegmentToStrSlice(segment *resource.AddressCodeLayoutSegment) []string {
	return []string{localizationSegmentValue(segment.Value), segment.Code}
}

func localizationSegmentValue(value string) string {
	if value == "unknown" {
		return "缺省"
	} else {
		return value
	}
}

func internationalizationSegmentValue(value string) string {
	if value == "缺省" {
		return "unknown"
	} else {
		return value
	}
}

func localizationClientClassStrategy(strategy string) string {
	if strategy == resource.ClientClassStrategyAnd {
		return "满足全部"
	} else {
		return "满足一个"
	}
}

func localizationBoolSwitch(b bool) string {
	if b {
		return "开启"
	} else {
		return "关闭"
	}
}

func internationalizationBoolSwitch(b string) bool {
	if b == "开启" {
		return true
	} else {
		return false
	}
}

func internationalizationClientClassStrategy(strategy string) string {
	if strategy == "满足全部" {
		return resource.ClientClassStrategyAnd
	} else {
		return resource.ClientClassStrategyOr
	}
}

func uint32ToString(lifetime uint32) string {
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
	buf.WriteString(subnet4.Tags)
	buf.WriteString("','")
	buf.WriteString(subnet4.IfaceName)
	buf.WriteString("','")
	buf.WriteString(subnet4.WhiteClientClassStrategy)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet4.WhiteClientClasses, ","))
	buf.WriteString("}','")
	buf.WriteString(subnet4.BlackClientClassStrategy)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet4.BlackClientClasses, ","))
	buf.WriteString("}','")
	buf.WriteString(uint32ToString(subnet4.ValidLifetime))
	buf.WriteString("','")
	buf.WriteString(uint32ToString(subnet4.MaxValidLifetime))
	buf.WriteString("','")
	buf.WriteString(uint32ToString(subnet4.MinValidLifetime))
	buf.WriteString("','")
	buf.WriteString(subnet4.NextServer)
	buf.WriteString("','")
	buf.WriteString(subnet4.SubnetMask)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet4.Routers, ","))
	buf.WriteString("}','{")
	buf.WriteString(strings.Join(subnet4.DomainServers, ","))
	buf.WriteString("}','")
	buf.WriteString(subnet4.TftpServer)
	buf.WriteString("','")
	buf.WriteString(subnet4.Bootfile)
	buf.WriteString("','")
	buf.WriteString(subnet4.RelayAgentCircuitId)
	buf.WriteString("','")
	buf.WriteString(subnet4.RelayAgentRemoteId)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet4.RelayAgentAddresses, ","))
	buf.WriteString("}','")
	buf.WriteString(uint32ToString(subnet4.Ipv6OnlyPreferred))
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet4.CapWapACAddresses, ","))
	buf.WriteString("}','{")
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
	buf.WriteString(reservation4.Hostname)
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
	buf.WriteString(subnet6.Tags)
	buf.WriteString("','")
	buf.WriteString(subnet6.IfaceName)
	buf.WriteString("','")
	buf.WriteString(subnet6.WhiteClientClassStrategy)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet6.WhiteClientClasses, ","))
	buf.WriteString("}','")
	buf.WriteString(subnet6.BlackClientClassStrategy)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet6.BlackClientClasses, ","))
	buf.WriteString("}','")
	buf.WriteString(uint32ToString(subnet6.ValidLifetime))
	buf.WriteString("','")
	buf.WriteString(uint32ToString(subnet6.MaxValidLifetime))
	buf.WriteString("','")
	buf.WriteString(uint32ToString(subnet6.MinValidLifetime))
	buf.WriteString("','")
	buf.WriteString(uint32ToString(subnet6.PreferredLifetime))
	buf.WriteString("','")
	buf.WriteString(subnet6.RelayAgentInterfaceId)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet6.DomainServers, ","))
	buf.WriteString("}','")
	buf.WriteString(uint32ToString(subnet6.InformationRefreshTime))
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet6.CapWapACAddresses, ","))
	buf.WriteString("}','{")
	buf.WriteString(strings.Join(subnet6.RelayAgentAddresses, ","))
	buf.WriteString("}','")
	if subnet6.RapidCommit {
		buf.WriteString("true")
	} else {
		buf.WriteString("false")
	}
	buf.WriteString("','")
	if subnet6.EmbedIpv4 {
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
	buf.WriteString(subnet6.AddressCode)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(subnet6.Nodes, ","))
	buf.WriteString("}','")
	buf.WriteString(subnet6.Capacity)
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
	buf.WriteString(pool6.Capacity)
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
	buf.WriteString(pool6.Capacity)
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
	buf.WriteString("','")
	buf.WriteString(reservation6.Hostname)
	buf.WriteString("','{")
	buf.WriteString(strings.Join(reservation6.IpAddresses, ","))
	buf.WriteString("}','{")
	buf.WriteString(ipsToString(reservation6.Ips))
	buf.WriteString("}','{")
	buf.WriteString(strings.Join(reservation6.Prefixes, ","))
	buf.WriteString("}','{")
	buf.WriteString(ipnetsToString(reservation6.Ipnets))
	buf.WriteString("}','")
	buf.WriteString(reservation6.Capacity)
	buf.WriteString("','")
	buf.WriteString(reservation6.Comment)
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(subnetId, 10))
	buf.WriteString("'),")
	return buf.String()
}

func ipsToString(ips []net.IP) string {
	ipstrs := make([]string, 0, len(ips))
	for _, ip := range ips {
		ipstrs = append(ipstrs, ip.String())
	}

	return strings.Join(ipstrs, ",")
}

func ipnetsToString(ipnets []net.IPNet) string {
	ipnetstrs := make([]string, 0, len(ipnets))
	for _, ipnet := range ipnets {
		ipnetstrs = append(ipnetstrs, ipnet.String())
	}

	return strings.Join(ipnetstrs, ",")
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
	buf.WriteString(uint32ToString(pdpool.PrefixLen))
	buf.WriteString("','")
	buf.WriteString(pdpool.PrefixIpnet.String())
	buf.WriteString("','")
	buf.WriteString(uint32ToString(pdpool.DelegatedLen))
	buf.WriteString("','")
	buf.WriteString(pdpool.Capacity)
	buf.WriteString("','")
	buf.WriteString(pdpool.Comment)
	buf.WriteString("','")
	buf.WriteString(strconv.FormatUint(subnetId, 10))
	buf.WriteString("'),")
	return buf.String()
}

func assetToInsertDBSqlString(asset *resource.Asset) string {
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(asset.HwAddress)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(asset.Name)
	buf.WriteString("','")
	buf.WriteString(asset.HwAddress)
	buf.WriteString("','")
	buf.WriteString(asset.AssetType)
	buf.WriteString("','")
	buf.WriteString(asset.Manufacturer)
	buf.WriteString("','")
	buf.WriteString(asset.Model)
	buf.WriteString("','")
	buf.WriteString(asset.OperatingSystem)
	buf.WriteString("','")
	buf.WriteString(asset.AccessNetworkTime)
	buf.WriteString("'),")
	return buf.String()

}

func segmentToInsertDBSqlString(layoutId string, segment *resource.AddressCodeLayoutSegment) string {
	id, _ := uuid.Gen()
	var buf bytes.Buffer
	buf.WriteString("('")
	buf.WriteString(id)
	buf.WriteString("','")
	buf.WriteString(time.Now().Format(time.RFC3339))
	buf.WriteString("','")
	buf.WriteString(segment.Code)
	buf.WriteString("','")
	buf.WriteString(segment.Value)
	buf.WriteString("','")
	buf.WriteString(layoutId)
	buf.WriteString("'),")
	return buf.String()
}
