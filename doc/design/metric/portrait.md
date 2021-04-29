
# Portrait
## 概要
画像包括了资产画像, 区域画像和应用访问趋势三个资源, 资产画像是clxone-dhcp 每5分钟去扫描路由，收集资源的在线，离线信息，更新到数据库，客户请求时，直接从数据库中搜索；区域画像和应用访问趋势是由ddi-agent将dns访问记录上报给elasticsearch搜索引擎，客户请求时，clxone-dhcp会去es中搜索相应数据，然后返回给客户。

## 动机和目的
实时监测资产状态，IP访问统计，应用访问统计等业务指标，方便用户查看资产在线状态，ip访问区域分布和应用访问趋势

## 资源
### RegionPortrait - 区域画像
* 父资源为metric
* 字段：province, hits
* 支持操作
  * 获取（List [domain]）
    * List会返回一个月内访问DNS服务器的top 10的省份和访问次数
* 每个字段介绍如下
  * province - 省份名称
  * hits - 访问次数

### AppTrend - 应用访问趋势
* 父资源为metric
* 字段： domain, hits, trend
* 支持操作
* 获取（list [domain])
    * List 会返回一个月内每天top5的应用访问数，可以指定某一个域名单独进行统计，不指定域名的情况下则统计所有域名的地域访问次数排名
    * domain为可选参数，如果指定则表示对该域名进行单独统计
* 每个字段介绍如下
  * domain - 域名
  * hits - 访问总次数
  * trend 为数组，每个元素包含如下属性
    * date - 日期
    * hits - 当日访问次数

### AssetPortrait - 资产画像
* 父资源为metric
* 字段：DeviceTotal, OnlineTotal, OfflineTotal，AbnormalTotal, StateStatistics
* 支持操作（list[province]）
  * list会返回一个月内终端数量总数，在线总数，离线总数，异常总数和安省份排名top5的在线，离线，异常数
  * 支持province参数，如果支持该参数，则按照省份统计top城市的在线，离线，异常数。
* 每个字段介绍如下
  * DeviceTotal - 设备总数
  * OnlineTotal - 在线设备总数
  * OfflineTotal - 离线设备总数
  * AbnormalTotal - 异常设备总数
  StateStatistics 为数组，每个元素的属性如下
    * Region - 省份/城市
    * Online - 在线数
    * Offline - 离线数
    * Abnormal - 异常数