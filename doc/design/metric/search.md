# Search
### 概要
快速查询功能分为ip资产查询和域名访问查询
  
  * ip资产查询主要查询该ip所属规划，所属子网，所属终端信息，以及ip访问域名的top纪录和ip被分配的历史纪录
  * 域名访问查询主要查询访问该域名的top ip信息，ip信息包含所属规划，所属子网，所属终端信息
  
  
### 资产查询 assetsearch
* metric的顶级资源，包含字段
  * 资产类型 assetType，枚举类型：ip和domain
  * IP资产信息 ipAsset
  * 域名资产信息 domainAsset

* IP资产包含字段:
  * 子网信息 subnet
    * 子网地址 subnet
    * 子网类型 networkType
    * 语义节点信息 semanticName 
  * ip状态 ipState
    * 地址类型 ipType
    * 地址状态 ipState
  * 终端信息 device 
    * 终端名字 name
    * 终端类型 deviceType
    * 终端MAC mac
    * 上联设备 uplinkEquipment
    * 上联端口 uplinkPort
    * Vlan vlanId
    * 机房 computerRoom
    * 机柜 computerRack
    * 部署服务 deployedService
    * 所属部门 department
    * 负责人 responsiblePerson
    * 联系电话 telephone
  * 分配历史 allocatedHistories (array)
    * 终端MAC mac
    * 地址类型 ipType
    * 地址状态 ipState
    * 分配时间 time
  * 访问历史 browsedHistories (array)
    * 访问域名信息 browsedDomain
      * 访问域名 domain
      * 访问次数 count
    * 访问域名top10地址信息 browserTopIps
      * 访问地址 ip
      * 访问次数 count

* IP资产字段对应    
  * 基础信息
    * 地址类型：ip状态的地址类型
    * 地址状态：ip状态的地址状态
  * 子网信息
    * 所属子网：子网信息的子网地址
    * 子网类型：子网信息的子网类型
    * 组织机构：子网信息的语义节点信息
  * 分配信息
    * 终端名称：终端信息的终端名字
    * 终端类型：终端信息的终端类型
    * 终端MAC：终端信息的终端MAC
    * 上联设备：终端信息的上联设备／上联端口 拼接
    * Vlan：终端信息的Vlan
    * 负责人：终端信息的负责人
    * 位置信息：终端信息的机房／机柜 拼接
    * 所属部门：终端信息的所属部门
    * 部署服务：终端信息的部署服务
    * 联系电话：终端信息的联系电话
  * 访问历史
    * 访问域名：访问历史的访问域名 
    * 访问次数：访问历史的访问次数
  * 分配历史：
    * 终端MAC：分配历史的终端MAC
    * 地址类型：分配历史的地址类型
    * 分配时间：分配历史的分配时间
  
* 域名资产包含字段：
  * top ip详情 ips
    * topip信息 topIp
      * 访问源地址 ip  
      * 访问次数 count
    * 子网信息 subnet
    * ip状态 ipState
    * 终端信息 asset
    * 访问历史 browsedDomains
      * 访问域名 domain
      * 访问次数 count
      
  * 权威纪录信息 authrrs
    * 视图名称 view
    * 区名称 zone
    * 纪录名称 rrName
    * 纪录类型 rrType
    * TTL  ttl
    * 纪录值 rdata

* 字段对应(top ip 表)
  * 访问源地址：topip信息的访问源地址
  * 访问次数：topip信息的访问次数
  * 所属子网：优先使用子网信息的子网地址，如果为空，其次使用规划信息的前缀信息
  * 所属终端：终端信息的终端名字
  * 上联设备：终端信息的上联设备
  * 上联端口：终端信息的上联端口  

* 查询一次查询一个ip或者域名，所以使用collection方法是需要添加filter，例如查询IP 10.0.0.7 url前缀/assetsearches?input=10.0.0.7，或者查询域名www.lx.com url前缀/assetsearches?input=www.lx.com
* 支持时间段选择period，按天计算，支持最近1天、7天、30天，即period=1 默认所有时间
* 支持top 个数，支持top 10，20，50，100，即top=10 默认top 10
* IP资产的访问历史中的每个纪录支持查询跳转到域名资产，域名资产的访问历史的IP支持查询跳转到IP资产

