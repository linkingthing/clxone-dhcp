# DHCP Cement
## 概览
配置godhcp，资源包含

* DHCPv4:
  * subnet4 DHCPv4子网
  * pool4 DHCPv4动态地址池
  * reservedpool4 DHCPv4保留地址池
  * reservation4 DHCPv4固定地址
  * pool4template DHCPv4地址池模版
  * clientclass4 DHCPv4 Option60
  * agent4 DHCPv4节点
  * sharednetwork4 DHCPv4共享网络
  * subnetlease4 DHCPv4子网租赁

* DHCPv6:
  * subnet6 DHCPv6子网
  * pool6 DHCPv6动态地址池
  * reservedpool6 DHCPv6保留地址池
  * pdpool DHCPv6前缀委派
  * reservedpdpool DHCPv6保留前缀
  * reservation6 DHCPv6固定地址
  * pool6template DHCPv6地址池模版
  * clientclass6 DHCPv4 Option16
  * agent6 DHCPv6节点
  * subnetlease6 DHCPv6子网租赁

* Common
  * dhcpconfig DHCP全局配置
  * dhcpfingerprint DHCP指纹
  * dhcpoui 网卡厂商
  * admit DHCP准入
  * admitmac 准入MAC
  * admitduid 准入DUID
  * admitfingerprint 准入指纹
  * ratelimit DHCP限速
  * ratelimitmac 限速MAC
  * ratelimitduid 限速DUID
  * pinger PING检测

## Pinger
* DHCP模块的顶级资源，用于配置ping检测
* 字段
  * enabled 是否开启 
    * 必填
    * 默认为false
    * 可更新
  * timeout 检测时延
    * 必填
    * 可更新
* 支持改、查
* 改

		PUT /apis/linkingthing.com/dhcp/v1/pingers/4dd73f9f40a2e2078099dcd06b95bb2a
		{
			"enabled": true,
			"timeout": 10
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/pingers
		GET /apis/linkingthing.com/dhcp/v1/pingers/4dd73f9f40a2e2078099dcd06b95bb2a


## DhcpOui 
* DHCP模块的顶级资源，用于识别网卡厂商
* 字段
  * oui OUI
    * 必填
    * 不可更新
    * 必须为48位MAC地址的前24位
  * organization 厂商
    * 必填
    * 可更新
  * isReadOnly 是否只读
    * 不可更新
    * 系统默认oui为true，用户自定义添加的oui为false
* 支持增、删、改、查
* 系统默认oui不支持更新、删除
* 增

		POST /apis/linkingthing.com/dhcp/v1/dhcpouis
		{
			"oui": "11:22:33",
			"organization": "huawei"
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/dhcpouis/11:22:33

* 改

		PUT /apis/linkingthing.com/dhcp/v1/dhcpouis/11:22:33
		{
			"organization": cisco
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/dhcpouis
		GET /apis/linkingthing.com/dhcp/v1/dhcpouis?oui=11:22:33
		
		GET /apis/linkingthing.com/dhcp/v1/dhcpouis/11:22:33

## DhcpFingerprint
* DHCP模块的顶级资源，用于DHCP指纹库的扩展
* 字段
  * fingerprint 指纹编码
    * 必填
    * 不可更新
    * 由数字和逗号组成，数字必须在1，254之间
  * vendorId 厂商标示
    * 可更新
  * operatingSystem 操作系统
    * 可更新
  * clientType 客户端类型
    * 可更新
  * matchPattern 厂商匹配模式
    * 可更新
    * 支持 equal，prefix，suffix，keyword，regexp
  * isReadOnly 是否只读
    * 不可更新
    * 系统默认的fingerprint为true，用户自定义的fingerprint为false
* 支持增、删、改、查
* 系统默认的fingerprint不支持更新、删除
* 增

		POST /apis/linkingthing.com/dhcp/v1/dhcpfingerprints
		{
			"fingerprint": "1,28,2,121,15,6,12,40,41,42,26,119,3,121,249,33,252,42,17",
    		"vendorId": "",
    		"operatingSystem": "RedHat/Fedora-based Linux",
    		"clientType": "Linux",
    		"matchPattern": "equal"
		}
* 删

		DELETE /apis/linkingthing.com/dhcp/v1/dhcpfingerprints/62d5a24a4027522e80e5569c843d117f

* 改

		PUT /apis/linkingthing.com/dhcp/v1/dhcpfingerprints/62d5a24a4027522e80e5569c843d117f
		{
			"vendorId": "dhcpcd%",
    		"operatingSystem": "RedHat/Fedora-based Linux",
    		"clientType": "Linux",
    		"matchPattern": "prefix"
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/dhcpfingerprints
		GET /apis/linkingthing.com/dhcp/v1/dhcpfingerprints?fingerprint=1,28&operatingSystem=Linux&clientType=Linux&vendorId=dhcpd
		
		GET /apis/linkingthing.com/dhcp/v1/dhcpfingerprints/62d5a24a4027522e80e5569c843d117f

## Admit
* DHCP模块的顶级资源，用于配置准入是否开启
* 字段
  * enabled 是否开启
    * 必填
    * 可更新
    * 默认为false
* 支持改、查
* 改
		
		PUT /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada
		{
			"enabled": true
		}
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/admits
		GET /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada

## AdmitMac
* DHCP模块的admit子资源，用于配置客户端mac是否被准入
* 字段
  * hwAddress 硬件地址
    * 必填
    * 必须为有效net.HardwareAddr地址
    * 不可更新
  * comment 备注
    * 可更新
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitmacs
		{
			"hwAddress": "11:22:33:44:55:66",
			"comment": "clxone"
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitmacs/11:22:33:44:55:66

* 改

		PUT /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitmacs/11:22:33:44:55:66
		{
			"comment": "clxone ipam"
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitmacs
		GET /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitmacs?hw_address=11:22:33:44:55:66
		
		GET /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitmacs/11:22:33:44:55:66

## AdmitDUID
* DHCP模块的admit子资源，用于配置客户端duid是否被准入
* 字段
  * duid DUID
    * 必填
    * 不可更新
    * 不能为空
  * comment 备注
    * 可更新
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitduids
		{
			"duid": "000100012937ef93b05cda255cdf"
			"comment": "clxone"
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitduids/000100012937ef93b05cda255cdf

* 改

		PUT /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitduids/000100012937ef93b05cda255cdf
		{
			"comment": "clxone ipam6"
		}
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitduids
		GET /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitduids?duid=000100012937ef93b05cda255cdf
		
		GET /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitduids/000100012937ef93b05cda255cdf
		
## AdmitFingerprint
* DHCP模块的admit子资源，用于配置客户端指纹类型是否被准入
* 字段
  * clientType 客户端类型
    * 必填
    * 不可为空，:支持
      * Android
      * AP
      * Apple
      * Linux
      * Others
      * Printer
      * Router
      * Switch
      * VoIP
      * Windows
* 支持增、删、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitfingerprints
		{
			"clientType": "Linux"
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitfingerprints/Linux
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitfingerprints
		GET /apis/linkingthing.com/dhcp/v1/admits/sdaasfawfsada/admitfingerprints/Linux

## RateLimit
* DHCP模块的顶级资源，配置限速是否开启
* 字段
  * enabled 是否开启
    * 必填
    * 默认为false
    * 可更新
* 支持改、查
* 改
		
		PUT /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529
		{
			"enabled": true
		}
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/ratelimits
		GET /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529
		
## RateLimitMac
* DHCP模块的ratelimit的子资源，配置限速的mac
* 字段
  * hwAddress MAC地址
    * 必填
    * 不可更新
    * 必须为有效的net.HardwareAddr地址
  * rateLimit 限速指标
    * 必填
    * 可更新
  * comment 备注
    * 可更新
* 支持增、删、改、查
* 增
		
		POST /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitmacs
		{
			"hwAddress": "11:22:33:44:55:66",
			"rateLimit": 100,
			"comment": "test 100"
		}
		
* 删

		DELETE /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitmacs/11:22:33:44:55:66
		
* 改

		PUT /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitmacs/11:22:33:44:55:66
		{
			"rateLimit": 50,
			"comment": "test 50"
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitmacs
		GET /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitmacs?hw_address=11:22:33:44:55:66
		
		GET /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitmacs/11:22:33:44:55:66

## RateLimitDuid
* DHCP模块的ratelimit的子资源，配置限速的duid
* 字段
  * duid DUID
    * 必填
    * 不可更新
    * 不可为空
  * rateLimit 限速指标
    * 必填
    * 可更新
  * comment 备注
    * 可更新
* 支持增、删、改、查
* 增
		
		POST /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitduids
		{
			"duid": "000100012937ef93b05cda255cdf",
			"rateLimit": 100,
			"comment": "test 100"
		}
		
* 删

		DELETE /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitduids/000100012937ef93b05cda255cdf
		
* 改

		PUT /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitduids/000100012937ef93b05cda255cdf
		{
			"rateLimit": 50,
			"comment": "test 50"
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitduids
		GET /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitduids?duid=000100012937ef93b05cda255cdf
		
		GET /apis/linkingthing.com/dhcp/v1/ratelimits/c033e72b4057b7a3803c1db97f339529/ratelimitduids/000100012937ef93b05cda255cdf


## DhcpConfig
* DHCP模块的顶级资源，配置subnet4、subnet6的租赁时间和DNS
* 字段
  * validLifetime  默认租约时长
    * 默认值为 14400
    * 可更新
    * 不可小于3600
  * maxValidLifetime 最大租约时长
    * 默认值为 28800
    * 可更新
    * 不可小于3600
  * minValidLifetime 最小租约时长
    * 默认值为 10800
    * 可更新
    * 不可小于3600
  * domainServers DNS服务器列表
    * 可更新
    * 每个dns服务器必须为有效ip地址
* 支持改、查
* 时间大小关系满足 minValidLifetime <= validLifetime <= maxValidLifetime
* 改

		PUT /apis/linkingthing.com/dhcp/v1/dhcpconfigs/dhcpglobalconfig
		{
			"validLifetime": 14400,
			"maxValidLifetime": 28800,
			"minValidLifetime": 7200,
			"domainServers": ["114.114.114.114", "8.8.8.8"]
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/dhcpconfigs
		
		GET /apis/linkingthing.com/dhcp/v1/dhcpconfigs/dhcpglobalconfig

## Agent4
* DHCP模块的顶级资源，下发DHCPv4配置时，用于选择DHCP的节点
* 字段
  * name 节点名字
  * ip 节点地址
* 从monitor服务获取sentry节点信息，如果节点是活跃的且满足以下其一
  * 如果某个节点存在vip，说明dhcp sentry为ha部署，只取该节点
  * 如果任何节点都没有vip，说明dhcp sentry为单机部署或者集群部署，取获取到的所有节点
* 仅支持查询

		GET /apis/linkingthing.com/dhcp/v1/agent4s
		
		GET /apis/linkingthing.com/dhcp/v1/agent4s/10.0.0.98
		
## ClientClass4
* DHCP模块的顶级资源，配置DHCPv4的option60
* 字段
  * name 名字
  	* 必填
  	* 不可更新
  	* 不可为空
  * regexp 正则表达式的值
    * 必填
    * 可更新
    * 不可为空
* 不能删除被subnet4引用的clientclass4
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/clientclass4s
		{
			"name": "op60",
			"regexp": "LXDHCPV4OP60"
		}

* 删 

		DELETE /apis/linkingthing.com/dhcp/v1/clientclass4s/op60		
* 改

		PUT /apis/linkingthing.com/dhcp/v1/clientclass4s/op60
		{
			"regexp": "LXDHCPV4OP60"
		}
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/clientclass4s
		
		GET /apis/linkingthing.com/dhcp/v1/clientclass4s/op60
		
* 目前只支持相等的正则表达式，所以regexp的值为LXDHCPV4OP60，意味着在DHCP服务器的配置为

		name: op60
    	code: 60
    	test: option vendor-class-identifier == 'LXDHCPV4OP60'


## Pool4Template
* DHCP模块的顶级资源，配置DHCPv4的动态地址池模版，用于生成Subnet4的地址池Pool4
* 字段
	* name 模版名字
	  * 必填
	  * 不可更新
	  * 不可为空
	* beginOffset 起始偏移量
	  * 必填
	  * 可更新
	  * 有效值为（0，65535）
	* capacity 容量
	  * 必填
	  * 可更新
	  * 有效值为（0，65535）
	* comment 备注
	  * 可更新
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/pool4templates
		{
			"name": "tp4_10",
			"beginOffset": 25,
			"capacity": 10,
			"comment": "from 25 to 35",
		}
		
* 删

		
		DELETE /apis/linkingthing.com/dhcp/v1/pool4templates/tp4_10
	
* 改

		PUT /apis/linkingthing.com/dhcp/v1/pool4templates/tp4_10
		{
			"beginOffset": 25,
			"capacity": 10,
			"comment": "from 25 to 35",
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/pool4templates
		GET /apis/linkingthing.com/dhcp/v1/pool4templates?name=tp4_10
		
		GET /apis/linkingthing.com/dhcp/v1/pool4templates/tp4_10		

## Subnet4

* DHCP模块的顶级资源，配置DHCPv4子网
* 字段
  * subnet 子网地址
    * 必填
    * 不可更新
    * 必须为有效net.IPNet
  * validLifetime 租约生命周期
    * 可更新
    * 默认值为DhcpConfig的validLifetime
  * maxValidLifetime 最大租约生命周期
    * 可更新
    * 默认值为DhcpConfig的maxValidLifetime 
  * minValidLifetime 最小租约生命周期 
    * 可更新
    * 默认值为DhcpConfig的minValidLifetime
  * subnetMask 子网掩码
    * 可更新
    * 必须为有效的net.IP且为IPv4
  * domainServers 域名服务器列表 （DNS）
    * 可更新
    * 每个dns必须为有效的net.IP且为IPv4
    * 默认值为DhcpConfig的domainServers
  * routers 网关列表
    * 可更新
    * 每个router必须为有效的net.IP且为IPv4
  * clientClass 自定义属性（option60）
    * 可更新
    * 必须在clientclass4中存在
  * ifaceName 网卡名字
    * 可更新
  * nextServer 启动服务器地址(web端暂不提供)
    * 可更新
  * relayAgentAddresses 中继路由地址列表 （option82）
    * 可更新
    * 每个relay必须为有效的net.IP且为IPv4
  * tftpServer TFTP服务器地址(option66)
    * 可更新
    * 必须为有效url.URL
  * bootfile  TFTP服务器启动文件(option67)
    * 可更新
  * tags 子网名字
    * 可更新
  * nodes 节点IP列表
    * 不可通过PUT更新，只能通过action＝update_nodes更新
    * 每个节点必须为有效的net.IP
  * nodeNames 节点名字列表
    * 不可更新
    * 实时从monitor服务根据节点IP获取节点名字
  * capacity 子网容量
    * 不可更新
  * usedRatio 子网地址使用率
    * 不可更新
    * 实时从dhcp代理端获取租赁个数计算
  * usedCount 子网地址已使用个数
    * 不可更新
    * 实时从dhcp代理端获取租赁个数
* 支持增、删、改、查
* 其它检查
  * 目前支持1w子网管理
  * subnet不能与已存在的相互包含 
  * 时间参数必须都大于3600且满足minValidLifetime <= validLifetime <= maxValidLifetime
  * 有租赁信息的子网不可以删除
  * 被共享网络使用的子网不能删除
* 增
	
		POST /apis/linkingthing.com/dhcp/v1/subnet4s
		{
			"subnet": "10.0.0.0/24",
    		"validLifetime": 14400,
    		"maxValidLifetime": 28800,
    		"minValidLifetime": 7200,
    		"subnetMask": "255.255.255.0",
    		"domainServers": ["114.114.114.114"],
    		"routers": ["10.0.0.2"],
    		"clientClass": "op60",
    		"ifaceName": "ens33",
    		"nextServer": "10.0.0.254"
    		"relayAgentAddresses": ["10.0.0.3"],
    		"tftpServer": "http://www.linkingthing.com/tftp.xml",
    		"bootfile": "TFTP.bin",
    		"tags": "lx>dev>ipam",
    		"networkType": "server",
    		"nodes": ["10.0.0.91", "10.0.0.92"]
    	}
  
* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet4s/1
		
* 改
		
		PUT /apis/linkingthing.com/dhcp/v1/subnet4s/1
		{
    		"subnetMask": "255.255.255.0",
    		"validLifetime": 14400,
    		"maxValidLifetime": 28800,
    		"minValidLifetime": 7200,
    		"subnetMask": "255.255.255.0",
    		"domainServers": ["114.114.114.114"],
    		"routers": ["10.0.0.2"],
    		"clientClass": "op60",
    		"ifaceName": "ens33",
    		"nextServer": "10.0.0.254"
    		"relayAgentAddresses": ["10.0.0.3"],
    		"tftpServer": "http://www.linkingthing.com/tftp.xml",
    		"bootfile": "TFTP.bin",
    		"tags": "lx>dev>ipam",
    		"networkType": "server",
    		"nodes": ["10.0.0.90", "10.0.0.91"]
    	}

* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet4s
		GET /apis/linkingthing.com/dhcp/v1/subnet4s?subnet=10.0.0.0/24
		
		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1

* Action
  * importcsv
  * exportcsv
  * exportcsvtemplate
  * update_nodes 更新子网节点配置
    * input 
      * nodes 节点列表
      
			POST /apis/linkingthing.com/dhcp/v1/subnet4s/1?action=update_nodes
			{
				"nodes": ["10.0.0.91", "10.0.0.92"]
			}

  * could_be_created 检查subnet是否可以创建
    * input
      * subnet 子网
      
			POST /apis/linkingthing.com/dhcp/v1/subnet4s?action=could_be_created
			{
				"subnet": "10.0.0.0/16"
			}
      		
  * list_with_subnets
    * input
      * subnets 子网列表
    * output
      * subnet4s Subnet4列表
      
			POST  /apis/linkingthing.com/dhcp/v1/subnet4s?action=list_with_subnets
			{
				"subnets": ["1.0.0.0/16","2.0.0.0/16", "3.0.0.0/16"],
			}

## SharedNetwork4
* DHCP模块的顶级资源，配置共享网络
* 字段
  * name 名字
    * 必填
    * 不可为空
  * subnetIds subnet4的ID列表
    * 用于创建和更新，不用于显示
    * 个数不能少于2个
    * 每个ID对应的子网必须存在
    * 每个ID不在被其他共享网络使用，即每个共享网络使用的子网互斥
    * 每个ID对应子网的节点列表不能为空，且必须存在共同交集，即子网1与子网2的节点交集，其它子网的节点列表必须包含这个交集
  * subnets subnet4的subnet列表
    * 仅用于展示
  * comment 备注
    * 可更新
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/sharednetwork4s
		{
			"name": "s1",
			"subnetIds": [1,2,3,4,5],
			"comment": "shared 12345"
		}
* 删

		DLETE /apis/linkingthing.com/dhcp/v1/sharednetwork4s/d8e8d7b24050c23080318063667cb5e5
		
* 改

		PUT /apis/linkingthing.com/dhcp/v1/sharednetwork4s/d8e8d7b24050c23080318063667cb5e5
		{
			"name": "s2",
			"subnetIds": [2,3,4,5],
			"comment": "shared 2345"
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/sharednetwork4s
		GET /apis/linkingthing.com/dhcp/v1/sharednetwork4s?name=s1
		
		GET /apis/linkingthing.com/dhcp/v1/sharednetwork4s/d8e8d7b24050c23080318063667cb5e5


## Pool4
* DHCP模块subnet4的子资源，配置subnet4的地址池
* 字段
  * beginAddress 开始地址
    * 不可更新
    * 若有值，必须为有效的net.IP且为IPv4
    * 如果来源于pool4template，通过子网的起始地址＋pool4template的起始偏移量beginOffset计算得出
  * endAddress 结束地址
    * 不可更新
    * 若有值，必须为有效的net.IP且为IPv4
    * 如果来源于pool4template，通过地址池的开始地址beginAddress＋pool4template的容量capacity － 1计算得出
  * template 地址池模版名字
    * 不可更新
    * 若有值，必须存在对应的pool4template
  * capacity 地址池容量
    * 不可更新
    * 首先通过endAddress － beginAddress ＋ 1 计算或者pool4template的capacity得出初始容量，然后减去地址池范围内的固定地址和保留地址的个数，最终得到该地址池的容量
  * usedRatio 地址池地址使用率
    * 不可更新
  * usedCount 地址池地址已使用个数
    * 不可更新
  * comment 备注
    * 可更新
* 其它检查
  * beginAddress和endAddress与template二者必填其一
  * endAddress不能小于beginAddress
  * beginAddress和endAddress必须在父资源subnet的范围内
  * 所有pool4之间不能有交集
  * 如果地址池有租赁信息，不能被删除
  * 创建和删除时，根据地址池的容量更新子网的容量
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s
		{
			"beginAddress": "10.0.0.25",
    		"endAddress": "10.0.0.35",
    		"comment"："25-35"
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s
		{
			"template": "tp4_10",
			"comment"："use template tp4_10"
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s/22e0dfaf40b445a280606c43a7c86b89
		
* 改

		PUT /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s/22e0dfaf40b445a280606c43a7c86b89
		{
			"comment"："update comment"
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s
		
		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s/22e0dfaf40b445a280606c43a7c86b89

* Action 
  * valid_template 检查该模版是否可以用于创建动态地址池
    * 入参
      * template 模版名字
    * 出参
      * beginAddress 开始地址
      * endAddress 结束地址
      
			POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s
			{
			  "template": "tp4_10"
			}

## ReservedPool4
* DHCP模块subnet4的子资源，配置subnet4的保留地址池，与Reservation4互斥
* 字段
  * beginAddress 开始地址
  * endAddress 结束地址
  * template 地址池模版名字
  * capacity 地址池容量
  * usedRatio 地址池地址使用率
  * usedCount 地址池地址已使用个数
  * comment 备注
* 创建和删除时，可能会影响地址池pool4及子网subnet4的容量
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s
		{
			"beginAddress": "10.0.0.1",
    		"endAddress": "10.0.0.5",
    		"comment": "keep 1-5"
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s
		{
			"template": "tp4_5",
			"comment"："use template tp4_5"
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s/22e0dfaf40b445a280606c43a7c86b89
		
* 改

		PUT /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s/22e0dfaf40b445a280606c43a7c86b89
		{
			"comment"："update comment"
		}
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s
		
		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s/22e0dfaf40b445a280606c43a7c86b89

* Action 
  * valid_template 检查该模版是否可以用于创建保留地址池
    * 入参
      * template 模版名字
    * 出参
      * beginAddress 开始地址
      * endAddress 结束地址


			POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s
			{
			  "template": "tp4_5"
			}
      
## Reservation4
* DHCP模块subnet4的子资源，配置subnet4的固定地址
* 字段
  * hwAddress 硬件地址
    * 必填
    * 不可更新
    * 必须为有效的net.HardwareAddr
  * ipAddress IP地址
    * 必填
    * 不可更新
    * 必须为有效的net.IP且为IPv4
  * capacity 容量
    * 不可更新
  * usedRatio 地址使用率
    * 不可更新
  * usedCount 已使用地址个数
    * 不可更新
  * comment 备注
    * 可更新
* 其它检查
  * hwAddress和ipAddress不能被其它保留地址使用
  * ipAddress必须属于子网范围
  * ipAddress与ReservedPool4互斥
  * 创建和删除时，可能会影响子网和地址池的容量
  * 不能删除已分配的固定地址
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservation4s
		{
			"hwAddress": "cc:64:a6:e0:5d:03",
			"ipAddress": "10.0.0.254",
			"comment": "cc"
		}
		
* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservation4s/ab86666240b199e080e2235d4e4982e2

* 改

		PUT /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservation4s/ab86666240b199e080e2235d4e4982e2
		{
			"comment": "ccd"
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservation4s
		
		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservation4s/ab86666240b199e080e2235d4e4982e2
		
## SubnetLease4
* DHCP模块subnet4的子资源，获取子网的所有租赁信息
* 字段
  * address IP地址
  * addressType IP地址类型（dynamic, reservation）
  * hwAddress MAC地址
  * hwAddressOrganization MAC厂商
  * clientId 客户端ID
  * validLifetime 租赁时长
  * expire 租赁过期时间
  * hostname 客户端主机名
  * fingerprint 指纹
  * vendorId 厂商
  * operatingSystem 操作系统
  * clientType 客户端类型
  * leaseState 租赁状态 （NORMAL, DECLINED, RECLAIMED）
* 其它检查
  * 删除租赁会在管理端保存已回收状态的租赁信息，如果服务端完成回收，获取子网的所有租赁信息时才会触发从管理端删除。
  * 删除租赁会检查该租赁是否是已回收状态，如果是，不做任何操作
* 支持获取和删除

		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/lease4s
		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/lease4s?ip=10.0.0.232
		
		DELETE /apis/linkingthing.com/dhcp/v1/subnet4s/1/lease4s/10.0.0.232
	
## Agent6
* DHCP模块的顶级资源，下发DHCPv6配置时，用于选择DHCP的节点
* 字段
  * name 节点名字 
  * ip 节点地址
* 仅支持查询

		GET /apis/linkingthing.com/dhcp/v1/agent6s
		
		GET /apis/linkingthing.com/dhcp/v1/agent6s/10.0.0.98
	
## ClientClass6
* DHCP模块的顶级资源，配置DHCPv6的option16
* 字段
  * name 名字
    * 必填
    * 不可更新
    * 不可为空
  * regexp 正则表达式的值
    * 必填
    * 不可更新
    * 不可为空
* 不能删除被subnet6引用的clientclass6
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/clientclass6s
		{
			"name": "op16",
			"regexp": "LXDHCPV6OP16"
		}

* 删 

		DELETE /apis/linkingthing.com/dhcp/v1/clientclass6s/op16		
* 改

		PUT /apis/linkingthing.com/dhcp/v1/clientclass6s
		{
			"regexp": "LXDHCPV6OP16"
		}
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/clientclass6s
		
		GET /apis/linkingthing.com/dhcp/v1/clientclass6s/op16
		
* 目前只支持相等的正则表达式，所以regexp的值为LXDHCPV6OP16，意味着在DHCP6服务器的配置为

		name: op16
    	code: 16
    	test: option vendor-class-identifier == 'LXDHCPV6OP16'


## Pool6Template
* DHCP模块的顶级资源，配置DHCPv6的动态地址池模版，用于生成Subnet6的地址池Pool6
* 字段
	* name 模版名字
	  * 必填
	  * 不可更新
	  * 不能为空
	* beginOffset 起始偏移量
	  * 必填
	  * 可更新
	  * 有效值为(0, 2147483647)
	* capacity 容量
	  * 必填
	  * 可更新
	  * 有效值为(0, 2147483647)
	* comment 备注
	  * 可更新
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/pool6templates
		{
			"name": "tp6_10",
			"beginOffset": 25,
			"capacity": 10,
			"comment": "from 25 to 35",
		}
		
* 删

		
		DELETE /apis/linkingthing.com/dhcp/v1/pool6templates/tp6_10
	
* 改

		PUT /apis/linkingthing.com/dhcp/v1/pool6templates/tp6_10
		{
			"beginOffset": 25,
			"capacity": 10,
			"comment": "from 25 to 35",
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/pool6templates
		GET /apis/linkingthing.com/dhcp/v1/pool6templates?name=tp6_10
		
		GET /apis/linkingthing.com/dhcp/v1/pool6templates/tp6_10
		
				
## Subnet6

* DHCP模块的顶级资源，配置DHCPv6子网
* 字段
  * subnet 子网地址
    * 必填
    * 不可更新
    * 必须为有效net.IPNet
  * validLifetime 租约生命周期
    * 可更新
    * 默认值为DhcpConfig的validLifetime
  * maxValidLifetime 最大租约生命周期
    * 可更新
    * 默认值为DhcpConfig的maxValidLifetime
  * minValidLifetime 最小租约生命周期
    * 可更新
    * 默认值为DhcpConfig的minValidLifetime
  * preferredLifetime 首选租约生命周期
    * 可更新
    * 默认值为DhcpConfig的validLifetime
    * 有效值为[minValidLifetime，validLifetime]
  * domainServers 域名服务器列表 （DNS）
    * 可更新
    * 每个dns必须为有效的net.IP且为IPv6
    * 默认值为DhcpConfig的domainServers
  * clientClass 自定义属性 （option16）
    * 可更新
    * 必须在clientclass6中存在
  * ifaceName 网卡名字
    * 可更新
  * relayAgentInterfaceId 中继路由网卡 （option18）
    * 可更新
  * relayAgentAddresses 中继路由地址列表
    * 可更新
    * 每个relay必须为有效的net.IP且为IPv6
  * tags 子网名字
    * 可更新
  * nodes 节点列表
    * 不可通过PUT更新，只能通过action＝update_nodes更新
    * 每个节点必须为有效的net.IP
  * nodeNames 节点名字列表
    * 不可更新
    * 实时从monitor服务根据节点IP获取节点名字
  * useEui64 是否启用EUI64分配地址
    * 可更新
    * 只有子网掩码为64的子网才能开启
  * capacity 子网容量
    * 不可更新
  * usedRatio 子网地址使用率
    * 不可更新
  * usedCount 子网地址已使用个数
    * 不可更新
* 其它检查
  * 目前支持1w子网管理
  * subnet6不能与已存在的相互包含 
  * 时间参数必须都大于3600且满足minValidLifetime <= preferredLifetime <= validLifetime <= maxValidLifetime
  * 有租赁信息的子网不可以删除
  * 有地址池的子网不能开启EUI64
* 支持增、删、改、查
* 增
	
		POST /apis/linkingthing.com/dhcp/v1/subnet6s
		{
			"subnet": "fd00:10::/64",
    		"validLifetime": 14400,
    		"maxValidLifetime": 28800,
    		"minValidLifetime": 7200,
    		"preferredLifetime": 14400,
    		"domainServers": ["2400:3200::1"],
    		"clientClass": "op16",
    		"ifaceName": "ens33",
    		"relayAgentInterfaceId": "Gi0/0/3",
    		"relayAgentAddresses": ["fd00:10::3"],
    		"tags": "lx>dev>ipam",
    		"networkType": "server",
    		"nodes": ["10.0.0.90", "10.0.0.91"],
    	}
  
* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1
		
* 改
		
		PUT /apis/linkingthing.com/dhcp/v1/subnet6s/1
		{
    		"validLifetime": 14400,
    		"maxValidLifetime": 28800,
    		"minValidLifetime": 7200,
    		"preferredLifetime": 14400,
    		"domainServers": ["2400:3200::1"],
    		"clientClass": "op16",
    		"ifaceName": "ens33",
    		"relayAgentInterfaceId": "Gi0/0/3",
    		"relayAgentAddresses": ["fd00:10::3"],
    		"tags": "lx>dev>ipam",
    		"networkType": "server",
    		"nodes": ["10.0.0.91", "10.0.0.92"]
    	}

* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet6s
		GET /apis/linkingthing.com/dhcp/v1/subnet6s?subnet=fd00:10::/64
		
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1

* Action
  * update_nodes 更新子网节点配置
    * input 
      * nodes 节点列表

			POST /apis/linkingthing.com/dhcp/v1/subnet6s/1?action=update_nodes
			{
    			"nodes": ["10.0.0.91", "10.0.0.92"]
			}
			
  * could_be_created 检查subnet是否可以创建
    * input
      * subnet 子网

			POST /apis/linkingthing.com/dhcp/v1/subnet6s?action=could_be_created
			{
				"subnet": "fd00:10::/32"
			}
			
  * list_with_subnets
    * input
      * subnets 子网列表
    * output
      * subnet6s Subnet6列表

			POST /apis/linkingthing.com/dhcp/v1/subnet6s?action=list_with_subnets
			{
				"subnets": ["fd00:10::/64", "fd00:20::/64", "fd00:30::/64"]
			}
  
## Pool6
* DHCP模块subnet6的子资源，配置subnet6的地址池
* 字段
  * beginAddress 开始地址
    * 不可更新
    * 若有值，必须为有效的net.IP且为IPv6
    * 如果来源于pool6template，通过子网的起始地址＋pool6template的起始偏移量beginOffset计算得出
  * endAddress 结束地址
    * 不可更新
    * 若有值，必须为有效的net.IP且为IPv6
    * 如果来源于pool6template，通过地址池的开始地址beginAddress＋pool6template的容量capacity － 1计算得出
  * template 地址池模版名字
    * 不可更新
    * 若有值，必须存在对应的pool4template
  * capacity 地址池容量
    * 不可更新
    * 首先通过endAddress － beginAddress ＋ 1 计算或者pool6template的capacity得出初始容量，然后减去地址池范围内的固定地址和保留地址的个数，最终得到该地址池的容量
  * usedRatio 地址池地址使用率
    * 不可更新
  * usedCount 地址池地址已使用个数
    * 不可更新
  * comment 备注
    * 可更新
* 其它检查
  * beginAddress和endAddress与template二者必填其一
  * endAddress不能小于beginAddress
  * beginAddress和endAddress必须在子网的范围内
  * 所有pool6之间不能有交集
  * 如果地址池有租赁信息，不能被删除
  * 创建和删除时，根据地址池的容量更新子网的容量
  * 开启EUI64的子网不能创建地址池
  * 子网掩码小于64的不能创建地址池
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s
		{
			"beginAddress": "fd00:10::25",
    		"endAddress": "fd00:10::35",
    		"comment": "25-35"
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s
		{
			"template": "tp6_10",
			"comment": "use tp6_10"
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s/22e0dfaf40b445a280606c43a7c86b89
	
* 改

		PUT /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s/22e0dfaf40b445a280606c43a7c86b89
		{
			"comment": "2535"
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s
		
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s/22e0dfaf40b445a280606c43a7c86b89

* Action 
  * valid_template 检查该模版是否可以用于创建动态地址池
    * 入参
      * template 模版名字
    * 出参
      * beginAddress 开始地址
      * endAddress 结束地址
      
			POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s
			{
			  "template": "tp6_10"
			}
					
## ReservedPool6
* DHCP模块subnet6的子资源，配置subnet6的保留地址池，与Reservation6互斥
* 字段
  * beginAddress 开始地址
  * endAddress 结束地址
  * template 地址池模版名字
  * capacity 地址池容量
  * usedRatio 地址池地址使用率
  * usedCount 地址池地址已使用个数
  * comment 备注
* 创建和删除时，可能会影响地址池pool6及子网subnet6的容量
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpool6s
		{
			"beginAddress": "fd00:10::1",
    		"endAddress": "fd00:10::5",
    		"comment": "25-35"
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpool6s
		{
			"template": "tp6_5",
			"comment": "25-35"
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpool6s/22e0dfaf40b445a280606c43a7c86b89

* 改

		PUT /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpool6s/22e0dfaf40b445a280606c43a7c86b89
		{
			"comment": "252335"
		}
			
* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpool6s
		
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpool6s/22e0dfaf40b445a280606c43a7c86b89
		
* Action 
  * valid_template 检查该模版是否可以用于创建保留地址池
    * 入参
      * template 模版名字
    * 出参
      * beginAddress 开始地址
      * endAddress 结束地址
      
			POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s
			{
			  "template": "tp6_10"
			}
			
## Reservation6
* DHCP模块subnet6的子资源，配置subnet6的固定地址
* 字段
  * duid 设备唯一标识符
    * 不可更新
  * hwAddress 硬件地址
    * 不可更新
    * 如果有值，必须为有效的net.HardwareAddr
  * ipAddresses IP地址列表
    * 不可更新
    * 如果有值，没有IP必须为有效的net.IP且为IPv6
  * prefixes 前缀列表 (web端暂不提供)
    * 不可更新
    * 如果有值，没有IP必须为有效的net.IPNet且为IPv6，掩码长度不能超过63
  * capacity 容量
    * 不可更新
    * ipAddresses或者prefixes的个数
  * usedRatio 地址使用率
    * 不可更新
  * usedCount 已使用地址个数
    * 不可更新
  * comment 备注
    * 可更新
* 其它检查
  * duid与hwAddress必须且只能存在一个，且不能被其它固定地址使用
  * ipAddresses与prefixes必须且只能存在一个，且不能被其它固定地址使用，同时必须在子网的范围内
  * 开启了EUI64的子网不能创建固定地址
  * 与ReservedPool6、ReservedPdPool互斥
  * 创建和删除时，可能会影响子网和地址池的容量
  * 不能删除已分配的固定地址
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s
		{
			"duid": "00042a342e29f765c199bacd5a1111119694",
			"ipAddresses": ["fd00:10::254"],
			"comment": "keep fd00:10::254 for duid"
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s
		{
			"duid": "00042a342e29f765c199bacd5a1111119694",
			"prefixes": ["fd00:20::/32"],
			"comment": "keep fd00:20::/32 for duid"
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s
		{
			"hwAddress": "cc:64:a6:e0:5d:03",
			"ipAddresses": ["fd00:10::254"],
			"comment": "keep fd00:10::254 for mac"
		}

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s
		{
			"hwAddress": "cc:64:a6:e0:5d:03",
			"prefixes": ["fd00:20::/32"],
			"comment": "keep fd00:20::/32 for mac"
		}
		
* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s/ab86666240b199e080e2235d4e4982e2
		
* 改

		PUT /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s/ab86666240b199e080e2235d4e4982e2
		{
			"comment": "keep fd00:20::/32 for mac cc:64:a6:e0:5d:03"
		}

* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s
		
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s/ab86666240b199e080e2235d4e4982e2
		

## PdPool
* DHCP模块subnet6的子资源，配置subnet6的前缀委派
* 字段
  * prefix 前缀地址
    * 必填
    * 不可更新
    * 必须为有效的net.IP且为IPv6
  * prefixLen 前缀长度
    * 必填
    * 不可更新
    * 有效值为（0，64）
  * delegatedLen 委派长度
    * 必填
    * 不可更新
    * 有效值为[prefixLen，64]
  * capacity 前缀容量
    * 不可更新
    * 通过(1 << (delegatedLen - prefixLen)) - 1计算得出
  * comment 备注
    * 可更新
* 其它检查
  * 开启EUI64的子网不能创建前缀委派
  * prefix和prefixLen必须在子网范围内
  * 子网的前缀委派不能相互包含，即prefix和prefixLen组成的IPNet不能相互包含
  * 有租赁信息的前缀委派不能删除
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/pdpools
		{
			"prefix": "fd10:10::"
			"prefixLen": 32,
			"delegatedLen": 48,
			"comment": "32-48"
		}
* 删
		
		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1/pdpools/86cddeec405362a780045990082056ad

* 改

		PUT /apis/linkingthing.com/dhcp/v1/subnet6s/1/pdpools/86cddeec405362a780045990082056ad
		{
			"comment": "32-48 with 32"
		}
	
* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/pdpools
		
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/pdpools/86cddeec405362a780045990082056ad
		
		
## ReservedPdPool（web端暂不提供）
* DHCP模块subnet6的子资源，配置subnet6的保留前缀委派，与Reservation6互斥
* 字段
  * prefix 前缀地址
  * prefixLen 前缀长度
  * delegatedLen 委派长度
  * capacity 前缀容量
  * comment 备注
* 子网的保留前缀委派不能相互包含
* 支持增、删、改、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpdpools
		{
			"prefix": "fd10:20::"
			"prefixLen": 32,
			"delegatedLen": 48,
			"comment": "keep 32-48"
		}
* 删
		
		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpdpools/86cddeec405362a780045990082056ad

* 改
		
		PUT /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpdpools/86cddeec405362a780045990082056ad
		{
			"comment": "keep 48"
		}
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpdpools
		
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpdpools/86cddeec405362a780045990082056ad
		
## SubnetLease6
* DHCP模块subnet6的子资源，获取子网的所有租赁信息
* 字段
  * address IP地址
  * addressType IP地址类型（dynamic, reservation）
  * prefixLen 前缀长度（仅PD有效）
  * duid DUID
  * iaid IAID
  * leaseType 租赁类型（IA_NA, IA_TA, IA_PD, IA_V4）
  * hwAddress MAC地址
  * hwAddressType MAC地址类型
  * hwAddressSource MAC地址来源 (DUID, IPv6_LINKLOCAL, CLIENT_LINKADDR, REMOTE_ID, DOCSIS_CMTS, DOCSIS_MODEM)
  * hwAddressOrganization MAC厂商
  * preferredLifetime 首选租赁时长
  * validLifetime 租赁时长
  * expire 租赁过期时间
  * hostname 客户端主机名
  * fingerprint 指纹
  * vendorId 厂商
  * operatingSystem 操作系统
  * clientType 客户端类型
  * leaseState 租赁状态 (NORMAL, DECLINED, RECLAIMED)
* 其它检查
  * 删除租赁会在管理端保存已回收状态的租赁信息，如果服务端完成回收，获取子网的所有租赁信息时才会触发从管理端删除。
  * 删除租赁会检查该租赁是否是已回收状态，如果是，不做任何操作
* 只支持获取和删除

		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/lease6s
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/lease6s?ip=2409:8762:317:120::2c
		
		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1/lease6s/2409:8762:317:120::2c
		
## 子网容量计算
* DHCPv4:
	*  pool4: 不计算reservedpool4、reservation4的地址
	*  subnet4: 所有pool4的容量 + 所有reservation4的容量
* DHCPv6
    *  pool6: 不计算reservedpool6、reservation6的地址
    *  pdpool: 不计算reservedpdpool、reservation6的前缀地址
    *  subnet6: 
    	* prefixLen == 64: 所有pool6的容量 + 所有reservation6的容量
    	* prefixLen < 64: 所有pdpool的容量 + 所有reservation6的容量
    
