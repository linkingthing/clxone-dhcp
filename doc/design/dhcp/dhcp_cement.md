# DHCP Cement
## 概览
配置godhcp，资源包含

* DHCPv4:
  * subnet4
  * pool4
  * reservedpool4
  * reservation4
  * pool4template
  * clientclass4
  * agent4

* DHCPv6:
  * subnet6
  * pool6
  * reservedpool6
  * pdpool
  * reservedpdpool
  * reservation6
  * pool6template
  * clientclass6
  * agent6

* Common
  * dhcpconfig
  * dhcpfingerprint

## DhcpFingerprint
* DHCP模块的顶级资源，用于DHCP指纹库的扩展
* 字段
  * fingerprint 指纹编码
  * vendorId 厂商标示
  * operatingSystem 操作系统
  * clientType 客户端类型
  * matchPattern 厂商匹配模式（equal，prefix，suffix，keyword，regexp）
  * isReadOnly 是否只读
* 支持增、删、改、查
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
		GET /apis/linkingthing.com/dhcp/v1/dhcpfingerprints/62d5a24a4027522e80e5569c843d117f


## DhcpConfig
* DHCP模块的顶级资源，配置subnet4、subnet6的租赁时间和DNS
* 字段
  * validLifetime  默认租约时长
  * maxValidLifetime 最大租约时长
  * minValidLifetime 最小租约时长
  * domainServers DNS服务器列表
* 支持改、查
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
  * ip 节点地址
* 仅支持查询

		GET /apis/linkingthing.com/dhcp/v1/agent4s
		GET /apis/linkingthing.com/dhcp/v1/agent4s/10.0.0.98
		
## ClientClass4
* DHCP模块的顶级资源，配置DHCPv4的自定义属性
* 字段
  * name 名字
  * regexp 正则表达式的值
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
	* beginOffset 起始偏移量
	* capacity 容量
	* comment 备注
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
		GET /apis/linkingthing.com/dhcp/v1/pool4templates/tp4_10		

## Subnet4

* DHCP模块的顶级资源，配置DHCPv4子网
* 字段
  * subnet 子网地址
  * validLifetime 租约生命周期
  * maxValidLifetime 最大租约生命周期 
  * minValidLifetime 最小租约生命周期 
  * subnetMask 子网掩码
  * domainServers 域名服务器列表 （DNS）
  * routers 网关列表
  * clientClass 自定义属性（option60）
  * ifaceName 网卡名字
  * nextServer 启动服务器地址(web端暂不提供)
  * relayAgentAddresses 中继路由地址列表 （option82）
  * tftpServer TFTP服务器地址(option66)
  * bootfile  TFTP服务器启动文件(option67)
  * tags 子网名字
  * networkType 子网类型
  * capacity 子网容量
  * usedRatio 子网地址使用率
  * usedCount 子网地址已使用个数
* 支持增、删、改、查
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
    		"networkType": "server"
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
    		"networkType": "server"
    	}

* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet4s
		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1


## Pool4
* DHCP模块subnet4的子资源，配置subnet4的地址池
* 字段
  * beginAddress 开始地址
  * endAddress 结束地址
  * template 地址池模版名字
  * capacity 地址池容量
  * usedRatio 地址池地址使用率
  * usedCount 地址池地址已使用个数
* 支持增、删、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s
		{
			"beginAddress": "10.0.0.25",
    		"endAddress": "10.0.0.35",
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s
		{
			"template": "tp4_10",
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s/22e0dfaf40b445a280606c43a7c86b89
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s
		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/pool4s/22e0dfaf40b445a280606c43a7c86b89


## ReservedPool4
* DHCP模块subnet4的子资源，配置subnet4的保留地址池，与Reservation4互斥
* 字段
  * beginAddress 开始地址
  * endAddress 结束地址
  * template 地址池模版名字
  * capacity 地址池容量
  * usedRatio 地址池地址使用率
  * usedCount 地址池地址已使用个数
* 支持增、删、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s
		{
			"beginAddress": "10.0.0.1",
    		"endAddress": "10.0.0.5",
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s
		{
			"template": "tp4_5",
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s/22e0dfaf40b445a280606c43a7c86b89
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s
		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservedpool4s/22e0dfaf40b445a280606c43a7c86b89


## Reservation4
* DHCP模块subnet4的子资源，配置subnet4的固定地址，与ReservedPool4互斥
* 字段
  * hwAddress 硬件地址
  * ipAddress IP地址
  * capacity 容量
  * usedRatio 地址使用率
  * usedCount 已使用地址个数
* 支持增、删、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservation4s
		{
			"hwAddress": "cc:64:a6:e0:5d:03",
			"ipAddress": "10.0.0.254",
		}
		
* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservation4s/ab86666240b199e080e2235d4e4982e2

* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservation4s
		GET /apis/linkingthing.com/dhcp/v1/subnet4s/1/reservation4s/ab86666240b199e080e2235d4e4982e2
	
## Agent6
* DHCP模块的顶级资源，下发DHCPv6配置时，用于选择DHCP的节点
* 字段
  * ip 节点地址
* 仅支持查询

		GET /apis/linkingthing.com/dhcp/v1/agent6s
		GET /apis/linkingthing.com/dhcp/v1/agent6s/10.0.0.98
	
## ClientClass6
* DHCP模块的顶级资源，配置DHCPv6的自定义属性
* 字段
  * name 名字
  * regexp 正则表达式的值
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
	* beginOffset 起始偏移量
	* capacity 容量
	* comment 备注
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
		GET /apis/linkingthing.com/dhcp/v1/pool6templates/tp6_10
		
				
## Subnet6

* DHCP模块的顶级资源，配置DHCPv6子网
* 字段
  * subnet 子网地址
  * validLifetime 租约生命周期
  * maxValidLifetime 最大租约生命周期
  * minValidLifetime 最小租约生命周期
  * preferredLifetime 首选租约生命周期
  * domainServers 域名服务器列表 （DNS）
  * clientClass 自定义属性 （option16）
  * ifaceName 网卡名字
  * relayAgentInterfaceId 中继路由网卡 （option18）
  * relayAgentAddresses 中继路由地址列表
  * tags 子网名字
  * networkType 子网类型
  * capacity 子网容量
  * usedRatio 子网地址使用率
  * usedCount 子网地址已使用个数
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
    		"networkType": "server"
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
    		"networkType": "server"
    	}

* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet6s
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1	
## Pool6
* DHCP模块subnet6的子资源，配置subnet6的地址池
* 字段
  * beginAddress 开始地址
  * endAddress 结束地址
  * template 地址池模版名字
  * capacity 地址池容量
  * usedRatio 地址池地址使用率
  * usedCount 地址池地址已使用个数
* 支持增、删、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s
		{
			"beginAddress": "fd00:10::25",
    		"endAddress": "fd00:10::35",
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s
		{
			"template": "tp6_10",
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s/22e0dfaf40b445a280606c43a7c86b89
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/pool6s/22e0dfaf40b445a280606c43a7c86b89
		
## ReservedPool6
* DHCP模块subnet6的子资源，配置subnet6的保留地址池，与Reservation6互斥
* 字段
  * beginAddress 开始地址
  * endAddress 结束地址
  * template 地址池模版名字
  * capacity 地址池容量
  * usedRatio 地址池地址使用率
  * usedCount 地址池地址已使用个数
* 支持增、删、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpool6s
		{
			"beginAddress": "fd00:10::1",
    		"endAddress": "fd00:10::5",
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpool6s
		{
			"template": "tp6_5",
		}

* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpool6s/22e0dfaf40b445a280606c43a7c86b89
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpool6s
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpool6s/22e0dfaf40b445a280606c43a7c86b89
		

## Reservation6
* DHCP模块subnet6的子资源，配置subnet6的固定地址，与ReservedPool6、ReservedPdPool互斥
* 字段
  * duid 设备唯一标识符
  * hwAddress 硬件地址
  * ipAddresses IP地址列表
  * prefixes 前缀列表 (web端暂不提供)
  * capacity 容量
  * usedRatio 地址使用率
  * usedCount 已使用地址个数
* 支持增、删、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s
		{
			"duid": "00042a342e29f765c199bacd5a1111119694",
			"ipAddresses": ["fd00:10::254"],
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s
		{
			"duid": "00042a342e29f765c199bacd5a1111119694",
			"prefixes": ["fd00:20::/32"],
		}
		
		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s
		{
			"hwAddress": "cc:64:a6:e0:5d:03",
			"ipAddresses": ["fd00:10::254"],
		}

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s
		{
			"hwAddress": "cc:64:a6:e0:5d:03",
			"prefixes": ["fd00:20::/32"],
		}
		
* 删

		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s/ab86666240b199e080e2235d4e4982e2

* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservation6s/ab86666240b199e080e2235d4e4982e2
		

## PdPool
* DHCP模块subnet6的子资源，配置subnet6的前缀委派
* 字段
  * prefix 前缀地址
  * prefixLen 前缀长度
  * delegatedLen 委派长度
  * capacity 前缀容量
* 支持增、删、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/pdpools
		{
			"prefix": "fd10:10::"
			"prefixLen": 32,
			"delegatedLen": 48,
		}
* 删
		
		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1/pdpools/86cddeec405362a780045990082056ad
		
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
* 支持增、删、查
* 增

		POST /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpdpools
		{
			"prefix": "fd10:20::"
			"prefixLen": 32,
			"delegatedLen": 48,
		}
* 删
		
		DELETE /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpdpools/86cddeec405362a780045990082056ad
		
* 查

		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpdpools
		GET /apis/linkingthing.com/dhcp/v1/subnet6s/1/reservedpdpools/86cddeec405362a780045990082056ad
		

## 容量计算
* DHCPv4:
	*  pool4: 不计算reservedpool4、reservation4的地址
	*  subnet4: 所有pool4的容量 + 所有reservation4的容量
* DHCPv6
    *  pool6: 不计算reservedpool6、reservation6的地址
    *  pdpool: 不计算reservedpdpool、reservation6的前缀地址
    *  subnet6: 
    	* prefixLen == 64: 所有pool6的容量 + 所有reservation6的容量
    	* prefixLen < 64: 所有pdpool的容量 + 所有reservation6的容量
    