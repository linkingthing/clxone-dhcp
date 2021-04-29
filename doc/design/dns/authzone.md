# 权威区 AuthZone
* 视图下的子资源，支持正向区和反向区创建，同时支持主辅区配置
* 字段包含：
  * 区名称 name
  * 区类型 zoneType，枚举类型，支持：
    * 正向区 standard
    * 反向区 arpa
  * TTL ttl
  * 区角色 role，枚举类型，支持：
    * 主区 master
    * 辅区 slave
  * 主区服务器 masters，当role是slave时配置
  * 辅区服务器 slaves，当role是master时配置
  * 记录个数 rrCount
  * 备注 comment
* 支持增、删、改、查
* 支持修改的字段
  * ttl
  * role  
    * 主区切换成辅区，会先删除当前区的所有纪录，然后向主区请求AXFR纪录
    * 当辅区切换成主区，只修改role值，不删除区纪录。
  * masters
  * slaves
  * comment
* 支持查询过滤
  * 区名称 name
  * 区类型 zone_type
  * 区角色 role
  * 主／辅区服务器 address
  * 备注 comment
* 创建主区，系统自动创建区的SOA记录，NS记录及NS记录对应的A记录
* 如果想创建辅区，需要在配置文件中配置notify的地址，用于接收主区notify消息

		｛
			server:
				notify_addr: 0.0.0.0:55553
		 ｝

  * 成功创建辅区必须满足以下条件：
    * 主区存在
    * 主区与辅区网络连通
    * 辅区可以从主区拿到AXFR全量记录数据，AXFR应答消息的Answer段要求以SOA纪录为起始和结束记录，例如：

			JAIN.AD.JP.         IN SOA serial=3
    		JAIN.AD.JP.         IN NS  NS.JAIN.AD.JP.
    		NS.JAIN.AD.JP.      IN A   133.69.136.1
    		JAIN-BB.JAIN.AD.JP. IN A   133.69.136.3
    		JAIN-BB.JAIN.AD.JP. IN A   192.41.197.2
    		JAIN.AD.JP.         IN SOA serial=3
                 
  * 辅区创建成功后，当主区有修改时，辅区会收到notify消息，然后辅区会使用当前SOA记录，向主区发送IXFR请求增量更新记录，增量应答消息Answer仍然以SOA纪录为起始和结束纪录，例如：serial 2 和 3之间是删除纪录，3和3之间是增加纪录
  
			JAIN.AD.JP.         IN SOA serial=3
			JAIN.AD.JP.         IN SOA serial=2
			JAIN-BB.JAIN.AD.JP. IN A   133.69.136.4
			JAIN-BB.JAIN.AD.JP. IN A   192.41.197.2
			JAIN.AD.JP.         IN SOA serial=3
			JAIN-BB.JAIN.AD.JP. IN A   133.69.136.3
			JAIN.AD.JP.         IN SOA serial=3



# 权威纪录 AuthRr
* 权威区的子资源
* 字段支持
  * 记录名称 name
  * 记录类型 rrType
    *  支持A AAAA CNAME HINFO MX NS NAPTR PTR SRV SOA TXT
  * TTL ttl
  * 记录值 rdata
  * 是否启用 enabled
  * 备注 comment
* 支持增、删、改、查
* 增加检查
  * SOA为创建区默认创建，不能单独创建
  * NS和MX记录的记录值对应的域名，必须先存在，才能创建，但不能是CNAME类型的记录
  * CNAME与其它类型的记录类型互斥
* 删除检查
  * 不能删除SOA记录
  * 不能删除区的最后一条NS记录
  * NS记录的rdata对应的记录，必须存在A、AAAA、NS其中一条，如下，不能删除ns.joe.com这条记录
  
  			
		@ 			NS 	ns.joe.com
		ns.joe.com 	A 	1.1.1.1 
* 修改
  * 字段：
    * ttl
    * rdata
    * comment
  * 检查逻辑与创建时检查NS、MX、CNAME相同
* 支持动作，辅区只支持导出操作。其他操作只有主区支持
  * action=enable 启用
  * action=diable 停用，因为要删除dns服务器中的记录，所以检查逻辑和删除相同
  * action=exportcsvtemplate 导出模版
    * 出参：
      * path: 模版文件的绝对路径
  * action=importcsv 导入记录，区创建会创建SOA记录，导入时不能导入SOA记录
    * 入参：
      * name: 导入文件的名字, 必填参数记录名称、纪录类型、TTL、纪录值，默认启用
  * action=exportcsv 导出记录
    * 出参：
      * path: 导出文件的绝对路径
