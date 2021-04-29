# ForwardZone

## 概要
转发区资源包含转发区、转发组、域名组、时间策略

#### 转发区 forwardzone
* 视图view下的子资源
* 字段包含
  * 转发类型 forwardItemType，枚举类型，支持：
    * root
    * domain
    * domain_group
  * 转发区名字 domain
    *  forwardItemType是root的时候是@
    *  forwardItemType是domain的时候是需要转发的区名字
  * 域名组id列表 domainGroupIds，forwardItemType是 domain_group 需要传
  * 转发组id列表 forwarderGroupIds
  * 时间策略 timeScheduler
  * 转发方式 forwardStyle
    * first
    * only
  * 备注 comment
* 支持增、删、改、查
* 有时间策略的转发区，在内存中做周期性检查时间有效性，来更新dns转发配置，检查周期为1分钟, 根据上一分钟的有效性和当前分钟内的有效性，对比发生变化的，将变化的配置添加或删除，都没有变化不做任何处理
* 增
  * 添加前做冲突检查，检查与新增的转发区的转发方式不同，且转发项的域名中有包含关系，且时间有重叠的
  * 如果没有时间策略，直接将配置发给dns节点
  * 如果有时间策略，添加到内存中，等待下一个检查周期
* 删
  * 没有时间策略的，删除dns节点配置
  * 有时间策略的，删除内存中的数据，如果当前的时间周期有效, 删除dns节点配置
* 改
  * 更新前做冲突检查，检查与新增的转发区的转发方式不同，且转发项的域名中有包含关系，且时间有重叠的
  * 支持更改的字段
    * forwarderGroupIds
    * timeScheduler
    * comment
  * 如果当前的timeScheduler为空，
    * 更新的timeScheduler为空，发送更新配置到dns节点
    * 更新的timeScheduler不为空，删除dns节点的这条转发配置，将新的转发规则添加到内存等待下一个检查周期
  * 如果当前的timeScheduler不为空
    * 更新的timeScheduler为空，删除内存中的数据，如果当前的时间周期无效，发送配置到dns节点
    * 更新的timeScheduler不为空，将新的转发规则更新到内存等待下一个检查周期
* 收到更新事件
  * 收到更新转发组事件
    * 如果没有转发区使用该转发组，不做任何处理
    * 如果使用该转发组的转发区，都有时间策略，不做任何处理，等待下一个检查周期，自动会生效
    * 将没有时间策略的转发区的配置更新到dns节点
  * 收到更新域名组事件
    * 如果没有转发区使用该域名组，不做任何处理
    * 如果使用该域名组的转发区，都有时间策略，不做任何处理，等待下一个检查周期，自动会生效
    * 将没有时间策略的转发区的配置更新到dns节点

#### 转发组 forwardergroup
* dns的顶级资源
* 字段包含
  * 名字 name
  * 转发组ip或者ip+端口列表 addresses
  * 备注 comment
* 支持增、删、改、查
* 不能删除被转发区使用的转发组
* 支持修改的字段
  * addresses 
  * comment
* 如果更新转发组时，addresses有变更，以事件的方式通知转发区

#### 域名组 domaingroup
* dns的顶级资源
* 字段包含
  * 名字 name
  * 域名列表 domains
  * 备注 comment
* 支持增、删、改、查
* 域名组之间的域名不能有包含关系
* 不能删除被转发区使用的域名组
* 支持修改的字段
  * domains
  * comment
* 如果更新域名组时，domains有变更，以事件的方式通知转发区

#### 时间策略 timescheduler
* dns的顶级资源
* 字段包含
  * 名字 name
  * 时间类型 timeType
    * 每天 daily
    * 每周 weekly
    * 每年 monthly
    * 日期 date
  * 时间段列表 timePeriods, timePeriod字段包含
    * beginTime
    * endTime
  * comment
* 支持增、删、改、查
* 增
  * 有效性规则检查，默认时间跨度不超过时间类型的范围，即每天的时间跨度不能超过24小时，每周的时间跨度不能超过7天
    * 每天：如果开始时间和结束时间的小时数相同，开始分钟不能大于结束分钟
    * 每周：如果开始周和结束周相同
      * 开始小时不能大于结束小时
      * 如果开始小时与结束小时相同，开始分钟不能大于结束分钟
    * 每年：如果开始月份小于等于结束月份，根据月日时间生成的时间，结束时间不能小于开始时间
    * 日期：结束时间大于开始时间
* 删
  * 不能删除正在被转发区使用的时间策略
* 支持修改的字段为
  * timeType
  * timePeriods
  * comment
* 时间策略是否有效，在内存中做周期性检查，检测周期为1分钟
* 范例：时间段列表timePeriods
  * 每天 daily  
    * 不跨天 3:00 - 5:00 对应 
    
			｛
				beginTime: "3:00",
				endTime: "5:00"
			 ｝
	* 跨天 23:00 － 5:00 对应
	    
			｛
				beginTime: "23:00",
				endTime: "5:00"
			 ｝
  * 每周 weekly 支持 周日 0 - 周六 6
    * 不跨周 周一 3:00 － 周五 16:00 对应
       
			｛
				beginTime: "1 3:00",
				endTime: "5 16:00"
			 ｝
  
	* 跨周 周五 17:00 － 周一 2:00
	    
			｛
				beginTime: "5 17:00",
				endTime: "1 2:00"
			 ｝		
  * 每年 monthly 支持月份 一月 1 － 十二月 12
    * 不跨年 1月1日 2:00 － 3月1日 3:00 对应
        
			｛
				beginTime: "1 1 2:00",
				endTime: "3 1 3:00"
			 ｝
    * 跨年 12月2日 3:00 － 2月1日 15:00
        
			｛
				beginTime: "12 2 3:00",
				endTime: "2 1 15:00"
			 ｝
  * 日期 date 如 2021年1月1日 3:00 － 2021年2月3日 1:00 对应 
    
			｛
				beginTime: "2021 1 1 3:00",
				endTime: "2021 2 3 1:00"
			 ｝  
