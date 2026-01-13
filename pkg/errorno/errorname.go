package errorno

type ErrName string

const (
	ErrNameName        ErrName = "name"
	ErrNameRegexp      ErrName = "regexp"
	ErrNameOffset      ErrName = "offset"
	ErrNameCapacity    ErrName = "capacity"
	ErrNameFingerprint ErrName = "fingerprint"
	ErrNameTime        ErrName = "time"
	ErrNameFullName    ErrName = "fullName"
	ErrNameID          ErrName = "id"
	ErrNameComment     ErrName = "comment"
	ErrNamePrefix      ErrName = "prefix"
	ErrNameFile        ErrName = "file"
	ErrNameImport      ErrName = "import"
	ErrNameExport      ErrName = "export"
	ErrNameTemplate    ErrName = "template"
	ErrNameTableHeader ErrName = "tableHeader"
	ErrNameDNS         ErrName = "DNS"
	ErrNameEUI64       ErrName = "EUI64"
	ErrNameBatchDelete ErrName = "batchDelete"

	ErrNameGateway                 ErrName = "gateway"
	ErrNameResponsiblePerson       ErrName = "responsiblePerson"
	ErrNameTelephone               ErrName = "telephone"
	ErrNameUsage                   ErrName = "usage"
	ErrNameNetworkAllocationMethod ErrName = "allocationMethod"
	ErrNameNetworkPurpose          ErrName = "purpose"
	ErrNameNetworkParentId         ErrName = "parentId"
	ErrNamePlanNetwork             ErrName = "planNetwork"
	ErrNameUser                    ErrName = "user"

	ErrNameSharedNetwork            ErrName = "sharedNetwork"
	ErrNameConfig                   ErrName = "config"
	ErrNameClientClass              ErrName = "clientClass"
	ErrNameClientClassStrategy      ErrName = "clientClassStrategy"
	ErrNameClientClassStrategyAnd   ErrName = "clientClassStrategyAnd"
	ErrNameClientClassStrategyOr    ErrName = "clientClassStrategyOr"
	ErrNameDhcpNode                 ErrName = "dhcpNode"
	ErrNameDhcpServerNode           ErrName = "dhcpServerNode"
	ErrNameDhcpSentryNode           ErrName = "dhcpSentryNode"
	ErrNameDhcpPool                 ErrName = "dhcpPool"
	ErrNameDhcpReservation          ErrName = "dhcpReservation"
	ErrNameDhcpReservedPool         ErrName = "dhcpReservedPool"
	ErrNameReservedPdPool           ErrName = "reservedPdPool"
	ErrNamePdPool                   ErrName = "pdPool"
	ErrNameLifetime                 ErrName = "validLifetime"
	ErrNameMinLifetime              ErrName = "minValidLifetime"
	ErrNameMaxLifetime              ErrName = "maxValidLifetime"
	ErrNamePreferLifetime           ErrName = "preferredLifetime"
	ErrNamePinger                   ErrName = "pinger"
	ErrNameAdmit                    ErrName = "admit"
	ErrNameRateLimit                ErrName = "rateLimit"
	ErrNameDuid                     ErrName = "duid"
	ErrNameLPS                      ErrName = "LPS"
	ErrNameAddressCode              ErrName = "addressCode"
	ErrNameAddressCodeLayout        ErrName = "addressCodeLayout"
	ErrNameAddressCodeLayoutSegment ErrName = "addressCodeLayoutSegment"
	ErrNameLabel                    ErrName = "label"
	ErrNameDelegatedLen             ErrName = "delegatedLen"
	ErrNameOui                      ErrName = "oui"
	ErrNameOrganization             ErrName = "organization"
	ErrNameNetwork                  ErrName = "network"
	ErrNameNetworkMask              ErrName = "networkMask"
	ErrNameNetworkV4                ErrName = "networkV4"
	ErrNameNetworkV4Detail          ErrName = "networkV4Detail"
	ErrNameNetworkV6                ErrName = "networkV6"
	ErrNameRootNetworkV6            ErrName = "rootNetworkV6"
	ErrNameNetworkV6Detail          ErrName = "networkV6Detail"
	ErrNameNetworkAllocateIp        ErrName = "allocateIp"
	ErrNameNetworkCreateMode        ErrName = "createMode"
	ErrNameNetworkPool              ErrName = "networkPool"
	ErrNameNetworkLease             ErrName = "networkLease"
	ErrNameIp                       ErrName = "ip"
	ErrNameIpv4                     ErrName = "ipv4"
	ErrNameIpv6                     ErrName = "ipv6"
	ErrNameVersion                  ErrName = "version"
	ErrNameAssignStatus             ErrName = "assignStatus"
	ErrNameLease                    ErrName = "lease"
	ErrNameCondition                ErrName = "condition"
	ErrNameParams                   ErrName = "params"
	ErrNameCharacter                ErrName = "character"
	ErrNameNumber                   ErrName = "number"
	ErrNameSpan                     ErrName = "span"
	ErrNameCmd                      ErrName = "cmd"
	ErrNameCode                     ErrName = "code"
	ErrNameDescription              ErrName = "description"
	ErrNameVendorId                 ErrName = "vendorId"
	ErrNameOperatingSystem          ErrName = "operatingSystem"
	ErrNameClientType               ErrName = "clientType"
	ErrNameTftpServer               ErrName = "option 66"
	ErrNameBootFile                 ErrName = "option 67"
	ErrNameRelayAgent               ErrName = "option 82"
	ErrNameRelayAgentIf             ErrName = "option 18"
	ErrNameRelayAgentCircuitId      ErrName = "relayAgentCircuitId"
	ErrNameRelayAgentRemoteId       ErrName = "relayAgentRemoteId"
	ErrNameRelayAgentAddresses      ErrName = "relayAgentAddresses"
	ErrNameIfName                   ErrName = "ifName"
	ErrNameCapWapACAddresses        ErrName = "capwapACAddresses"
	ErrNameAssetType                ErrName = "AssetType"
	ErrNameManufacturer             ErrName = "Manufacturer"
	ErrNameModel                    ErrName = "Model"
	ErrNameAssetOperatingSystem     ErrName = "OperatingSystem"
	ErrNameAccessNetworkTime        ErrName = "AccessNetworkTime"
	ErrNameAsset                    ErrName = "asset"
	ErrNameDomainSearchList         ErrName = "domainSearchList"
	ErrNameAutoReservationType      ErrName = "autoReservationType"
	ErrNameV6Prefix64               ErrName = "v6Prefix64"

	ErrNameMetric      ErrName = "metric"
	ErrNameUsedRatio   ErrName = "usedRatio"
	ErrNameDevice      ErrName = "device"
	ErrNameDeviceType  ErrName = "deviceType"
	ErrNameEquipment   ErrName = "equipment"
	ErrNameApplication ErrName = "application"
	ErrNameMac         ErrName = "mac"
	ErrNameHostname    ErrName = "hostname"
	ErrNameDeviceFlag  ErrName = "deviceFlag"

	ErrDBNameInsert ErrName = "dbInsert"
	ErrDBNameUpdate ErrName = "dbUpdate"
	ErrDBNameQuery  ErrName = "dbQuery"
	ErrDBNameDelete ErrName = "dbDelete"
	ErrDBNameCount  ErrName = "dbCount"
	ErrDBNameAlter  ErrName = "dbAlter"
	ErrDBNameExists ErrName = "dbExists"
)

const (
	ErrMethodCreate   = "create"
	ErrMethodUpdate   = "update"
	ErrMethodDelete   = "delete"
	ErrMethodList     = "list"
	ErrMethodGet      = "get"
	ErrMethodAction   = "action"
	ErrMethodReporter = "reporter"
	ErrMethodRevoke   = "revoke"
	ErrMethodMerge    = "merge"
	ErrMethodPing     = "ping"
)

var ErrNameMap = map[ErrName]string{
	ErrNameName:                    "名称",
	ErrNameRegexp:                  "匹配值",
	ErrNameOffset:                  "起始位置",
	ErrNameCapacity:                "数量",
	ErrNameFingerprint:             "指纹编码",
	ErrNameTime:                    "时间",
	ErrNameFullName:                "全名称",
	ErrNameComment:                 "备注",
	ErrNameImport:                  "导入数据",
	ErrNameExport:                  "导出数据",
	ErrNameTemplate:                "模板",
	ErrNameTableHeader:             "表头",
	ErrNamePrefix:                  "子网前缀",
	ErrNameFile:                    "文件",
	ErrNameGateway:                 "网关",
	ErrNameResponsiblePerson:       "负责人",
	ErrNameTelephone:               "联系电话",
	ErrNameUsage:                   "用途",
	ErrNameNetworkAllocationMethod: "子网分配方式",
	ErrNameNetworkPurpose:          "用途",
	ErrNameNetworkParentId:         "父级ID",
	ErrNamePlanNetwork:             "规划子网",
	ErrNameUser:                    "用户",
	ErrNameNetworkPool:             "子网地址池",
	ErrNameNetworkLease:            "子网租赁",
	ErrNameVendorId:                "厂商标识",
	ErrNameOperatingSystem:         "操作系统",
	ErrNameClientType:              "客户端类型",
	ErrNameRelayAgentCircuitId:     "中继路由电路标识",
	ErrNameRelayAgentRemoteId:      "中继路由远程标识",
	ErrNameRelayAgentAddresses:     "中继路由链路地址",
	ErrNameIfName:                  "网卡名字",
	ErrNameCapWapACAddresses:       "AC地址列表",
	ErrNameAssetType:               "资产类型",
	ErrNameManufacturer:            "资产厂商",
	ErrNameModel:                   "资产型号",
	ErrNameAssetOperatingSystem:    "操作系统",
	ErrNameAccessNetworkTime:       "入网时间",
	ErrNameAsset:                   "分配资产",
	ErrNameBatchDelete:             "批量删除",
	ErrNameDomainSearchList:        "域名后缀列表",
	ErrNameAutoReservationType:     "自动固定地址",
	ErrNameV6Prefix64:              "NAT64前缀",

	ErrDBNameInsert: "写入数据",
	ErrDBNameUpdate: "更新数据",
	ErrDBNameQuery:  "查询数据",
	ErrDBNameDelete: "删除数据",
	ErrDBNameCount:  "统计数据",
	ErrDBNameAlter:  "修改表",
	ErrDBNameExists: "查询数据是否存在",

	ErrMethodAction:   "操作",
	ErrMethodCreate:   "创建",
	ErrMethodUpdate:   "更新",
	ErrMethodDelete:   "删除",
	ErrMethodGet:      "查询",
	ErrMethodList:     "查询",
	ErrMethodReporter: "上报",
	ErrMethodRevoke:   "撤回",
	ErrMethodMerge:    "合并",
	ErrMethodPing:     "Ping",

	ErrNameSharedNetwork:            "共享网络",
	ErrNameConfig:                   "配置",
	ErrNameClientClass:              "OPTION",
	ErrNameClientClassStrategy:      "OPTION策略",
	ErrNameClientClassStrategyAnd:   "满足全部",
	ErrNameClientClassStrategyOr:    "满足一个",
	ErrNameDhcpNode:                 "DHCP节点",
	ErrNameDhcpServerNode:           "服务器节点",
	ErrNameDhcpSentryNode:           "哨兵节点",
	ErrNameDhcpPool:                 "动态地址池",
	ErrNameDhcpReservation:          "固定地址",
	ErrNameDhcpReservedPool:         "保留地址池",
	ErrNameReservedPdPool:           "保留前缀委派地址池",
	ErrNamePdPool:                   "前缀委派地址池",
	ErrNameLifetime:                 "租约时长",
	ErrNameMinLifetime:              "最短租约时长",
	ErrNameMaxLifetime:              "最长租约时长",
	ErrNamePreferLifetime:           "首选租约时长",
	ErrNamePinger:                   "Ping检测器",
	ErrNameAdmit:                    "接入配置",
	ErrNameRateLimit:                "限速配置",
	ErrNameAddressCode:              "地址编码",
	ErrNameAddressCodeLayout:        "地址编码标识",
	ErrNameAddressCodeLayoutSegment: "地址编码标识编码",
	ErrNameLabel:                    "标签类型",
	ErrNameDelegatedLen:             "委派长度",
	ErrNameOui:                      "网卡厂商",
	ErrNameOrganization:             "组织机构",
	ErrNameNetwork:                  "子网",
	ErrNameNetworkMask:              "子网掩码",
	ErrNameNetworkV4:                "IPv4子网",
	ErrNameNetworkV4Detail:          "IPv4子网详情",
	ErrNameNetworkV6:                "IPv6子网",
	ErrNameRootNetworkV6:            "IPv6根子网",
	ErrNameNetworkV6Detail:          "IPv6子网详情",
	ErrNameNetworkAllocateIp:        "分配IP",
	ErrNameNetworkCreateMode:        "子网创建方式",
	ErrNameIp:                       "IP地址",
	ErrNameIpv4:                     "IPv4地址",
	ErrNameIpv6:                     "IPv6地址",
	ErrNameVersion:                  "版本",
	ErrNameAssignStatus:             "分配状态",
	ErrNameLease:                    "租赁",
	ErrNameMetric:                   "指标信息",
	ErrNameUsedRatio:                "使用率",
	ErrNameDevice:                   "终端资产",
	ErrNameDeviceType:               "终端类型",
	ErrNameEquipment:                "设备资产",
	ErrNameApplication:              "应用资产",
	ErrNameMac:                      "Mac地址",
	ErrNameHostname:                 "主机",
	ErrNameDeviceFlag:               "设备标识",
	ErrNameCondition:                "查询条件",
	ErrNameParams:                   "参数",
	ErrNameCharacter:                "字符数",
	ErrNameNumber:                   "数值",
	ErrNameSpan:                     "跨度",
	ErrNameCmd:                      "cmd命令",
	ErrNameCode:                     "编码",
	ErrNameDescription:              "描述",
}

func localizeErrName(name ErrName) string {
	if cn, ok := ErrNameMap[name]; ok {
		return cn
	}
	return string(name)
}
