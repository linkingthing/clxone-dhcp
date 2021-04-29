# 地址规划
## 概要
对于给定的一个或多个地址段，提供灵活的策略来规划剩余的网络位，通过在网络位中编码额外的树状信息，一方面提高网络聚合，另一方面给ip地址提供更多的信息。

## 动机和目标
- 支持IPv4/IPv6网段，网络位规划
- 展示规划后生成的地址段

## 地址规划
```text

|<--- 21 bit -->|<-----------43 bit ------------->|
---------------------------------------------------
      ISP       |         待规划网络位            |

|<-- 4 bit -->|<-2 bit->|<--  6 bit  -->|<------- 31 bit --------->|
---------------------------------------------------------------------
 业务平台     |业务标识 |   省份标识     |        单位标识          |


|<-- 4 bit -->|<-- 6 bit -->|<--  8 bit  -->|<----- 25 bit  ------->|
---------------------------------------------------------------------
 电视用户     |  省份标识   |  地市标识     |     社区标识          |
``` 
对于除指定前缀之后的剩余的网络位，用户可以灵活的根据用途，地理位置等，确定
- 用多少位来代表
- 每个取值对应的实际意义
上图中，4位用户用来代表属性（用途）， 有16个取值，0(0000)代表业务平台，1（0001）代表电视用户，其他的值目前没有设定，留给以后扩展。

## 展示规划后的地址段
对于不同的规划，特别对于IPv6，可能生成很多网段，但是很多字段用户并没有指定对应的实际意义只是留做将来扩展，这种情况下，我们只展示设定了实际值的网络段。


## 详细设计
```text
+----------------+
|Plan            |
|  Name          |
|  Prefixs       |
|  MaxMaskWidths |    +-------------------+
|  SemanticNodes +--->|SemanticNode       |
|  Lockedby      |    |  Plan             |    +--------------------+
+----------------+    |  Name             |    |PlanNode            |
                      |  ParentSemanticId |    |  Plan 	            |
                      |  AutoCreate       |    |  Name              |
                      |  StepSize         |    |  SemanticId        |
                      |  Sequence         |    |  ParentPlanNodeId  |
                      |  Modified         |    |  Prefix            |
                      |  PlanNodes        +--->|  MaxMaskWidth      |
                      |  Ipv4s            |    |  Sequence          |
                      +-------------------+    |  BitWidth          |
                                               |  Value             |
                                               +--------------------+   

```
### Plan
| 参数名称            | 是否必填  | 数据类型         | 备注(输入输出是站在后端server的角度而言)                   |
| ------------------| -------- | --------------- | ---------------------------------------------------- |
| Name              | 否       | string          | 输入参数，plan name                                    |
| Prefixs           | 是       | []string        | 输入参数，ipv6地址前缀数组，包含掩码宽度，如：2300:1200::/32 |
| MaxMaskWidths     | 是       | []int           | 输入参数，最大掩码宽度数组，当前最大掩码宽度为64              |
| SemanticNodes     | 是       | []*SemanticNode | 输入参数，语义节点数组                                   |
| Lockedby          | 否       | string          | 输出参数，显示加锁状态，如果为空，表示未被锁定，可以编辑该Plan；如果被某用户加锁，输出其用户名 |
| ResponsorDispatch | 否       | *Dispatch       | 输入参数，当前Plan包含（如果有）的Dispatch结构           |
对一个或多个指定网络前缀（仅限IPv6）的地址规划，称之为一个Plan。Plan的每个前缀的掩码宽度和最大掩码宽度必须相等。IPv6网络前缀目前最长可规划位宽为64位。
Plan是由若干个语义节点（SemanticNode）构成的树状结构（语义树），除了包含一棵语义树，又包含若干棵网络树（每个Plan前缀作为一棵网络树的根节点）。

#### SemanticNode 语义节点
| 参数名称            | 是否必填  | 数据类型         | 备注(输入输出是站在后端server的角度而言)                |
| ------------------| -------- | ---------------| -------------------------------------------------- |
| Plan              | 否       | string         | 输出参数，当前语义节点所属的Plan id                     |
| Name              | 否       | string         | 输入参数，当前语义节点名称                              |
| ParentSemanticId  | 是       | string         | 输入参数，当前语义节点的父SemanticNode id               |
| AutoCreate        | 是       | bool           | 输入参数，当前语义节点自动创建为true，否则为false          |
| SubnodeBitWidth   | 是       | int            | 输入参数，用位宽表示当前语义节点的子语义节点个数，与子语义节点的所有Plannode中的BitWidth相等   |
| StepSize          | 是       | int            | 输入参数，当前语义节点自动创建时包含的子网个数              |
| Sequence          | 否       | int            | 输入参数，当前语义节点与其他兄弟语义节点的排序顺序           |
| Modified          | 否       | string Options | 输入参数，options=no,structured,info, 标识当前语义节点被修改的状态，包括自身属性和子节点的语义结构  |
| PlanNodes         | 否       | []*PlanNode    | 输入参数，当前语义节点包含的PlanNode数组                  |
| Ipv4s             | 否       | []string       | 输入参数，当前语义节点包含的IPv4前缀数组                   |
| SponsorDispatchId | 否       | string         | 输出参数，当前语义节点包含（如果有）的Dispatch结构ID        |
| SponsorDispatch   | 否       | *Dispatch      | 输入参数，当前语义节点包含（如果有）的Dispatch结构          |
1. Modified标识该节点是否被修改，以及被修改的不同状态，用于前端调用create和update接口的时候，具体情况如下：
  Modified=structured，反应如下情况:
  - 该节点自身是新增节点
  - 该节点自身的PlanNodes（网络节点）被删除
  - 该节点的子节点被删除
  Modified=info，反应如下情况:
  - 该节点的string或int类型的字段被修改
  - 该节点的PlanNodes（网络节点）有新增
  Modified=no，反应如下情况:
  - 该节点没有上述的任一改变
  - 该节点新增子节点
  - 该节点的子节点被修改（不包括子节点被删除）
 准确设置Modified值，不仅有利于程序优化，也是保证逻辑正确的要求。

Dispatch结构表示其所在语义节点（或Plan）的下发与上报操作信息（如果有）：

| 参数名称            | 是否必填  | 数据类型         | 备注(输入输出是站在后端server的角度而言)                          |
| ------------------| -------- | ---------------| ------------------------------------------------------------ |
| Plan              | 否       | string         | 输出参数，当前Dispatch所属的Plan ID                              |
| SemanticNode      | 否       | string         | 输入参数，当前Dispatch所属的语义节点ID                             |
| IsSponsor         | 否       | bool           | 输入参数，当前下发上报操作是否为当前节点（或Plan）主动发起              |
| RemoteAddr        | 否       | string         | 输入参数，当前Dispatch的发起者IP地址和端口                          |
| LastAction        | 否       | string         | 输入参数，当前Dispatch的最后一次行为（Dispatch，Report，Repeal）     |
| IsActionSucceeded | 否       | bool           | 输出参数，当前Dispatch的最后一次行为是否成功                         |
对语义节点包含的Dispatch来说，只能是主动下发信息，即IsSponsor为true；对Plan包含的Dispatch来说，只能是接受的下发信息，即IsSponsor为false。

2. 每个语义节点可以包含一个或多个网络节点（PlanNode），每个网络节点包含一个网络前缀。
3. 每个SemanticNode节点通过ParentSemanticId的级联关系组成一棵语义树，根语义节点的ParentSemanticId值为"0"。一个Plan只能有一个根语义节点（SemanticNode）。

#### PlanNode 网络节点
| 参数名称           | 是否必填   | 数据类型         | 备注(输入输出是站在后端server的角度而言)                                         |
| ----------------- | -------- | ---------------| --------------------------------------------------------------------------- |
| Plan              | 否       | string         | 输出参数，当前网络节点所属的Plan id                                               |
| Name              | 否       | string         | 输入参数，当前网络节点名称，与所属的语义节点名称相同                                  |
| SemanticId        | 是       | string         | 输入参数，当前网络节点所在的SemanticNode id                                       |
| ParentPlanNodeId  | 是       | string         | 输入参数，当前网络节点的父PlanNode（父SemanticNode中的某个Plannode）id              |
| Prefix            | 否       | string         | 输入参数，当前网络节点的网络前缀，由父PlanNode的Prefix和自己的BitWidth与Value值计算得到 |
| MaxMaskWidth      | 否       | int            | 输入参数，当前网络规划的最大可规划位宽，同一个网络前缀下的所有网络节点的该值均相等          |
| Sequence          | 否       | int            | 输入参数，当前网络节点与其他兄弟语义节点的排序顺序                                    |
| BitWidth          | 否       | int            | 输入参数，当前网络节点所占的位宽                                                   |
| Value             | 否       | string         | 输入参数，当前网络节点在所属的位宽里的取值，如位宽为N，则Value取值为[1, 2^N - 1]         |
1. 根语义节点包含若个根网络节点。每个根网络节点的Prefix由用户输入的前缀填充，每个Prefix对应一颗网络树。根网络节点的ParentPlanNodeId值为"0"，BitWidth和Value默认填充为0。
2. 每个网络节点通过ParentPlanNodeId的级联关系组成一棵网络树,BitWidth大于0表示有效设置，否则该网络节点无法生成合法前缀，也不能成为其他网络节点的父节点。

### 关于Plan锁
 - Plan在编辑前，需先获取锁（即后端加锁）；前端的显示则相反，前端默认显示为关锁状态，不可编辑，在调用后端加锁接口且加锁成功后（后端返回的Lockedby值为自身user name），前端显示为开锁状态，可以编辑。
 - 普通用户只能在Plan未锁定（后端锁）的情况下，进行加锁操作；当被某用户加锁后，任何普通用户都不能再进行加锁或释放锁操作，也即不能对Plan进行编辑。
 - 超级用户Admin可以在任何情况下获取Plan锁的控制，即不管是否被其他用户锁定，Admin都可以抢占该锁，并进行自己的加锁或开锁操作。
 - 锁操作仅以用户名为唯一判断依据，与用户操作的终端无关。即持锁用户可以从多个终端登录编辑Plan，但每个终端的最新状态均以最后保存到后端的信息为准。
注意：目前代码已暂时关闭Plan锁的功能，因该机制不能阻止同一个user在多个客户端登录造成的其中一个客户端的编辑状态丢失。

### 关于Plan导入导出
导入导出文档格式示例：

| Level1 |       IPv6         | MaxMaskWidth | IPv4         | Level2 | BitWidth | Value | Level3 | BitWidth | Value |     
| ------ | ------------------ | ------------ | ------------ | ------ | -------- | ----- | ------ | -------- | ----- |
| 新规划1 |                    |              | 10.1.0.0/16  |        |          |       |        |          |       |
| 新规划1 |                    |              | 10.2.0.0/16  |        |          |       |        |          |       |
| 新规划1 | 2600:1600:100::/32 | 64           |              |       |          |       |        |          |       |
| 新规划1 |                    |              |              | 人事部 | 8        | 1     |        |          |       |
| 新规划1 |                    |              |              | 人事部 | 8        | 1     | 招聘组 | 8        | 1     |
| 新规划1 |                    |              |              | 人事部 | 8        | 1     | 招聘组 | 8        | 2     |
| 新规划1 | 2600:1600:200::/32 | 64           |              |       |          |       |       |          |       |
| 新规划1 |                    |              | 10.1.32.0/20 | 售后部 |          |       |       |          |       |
| 新规划1 |                    |              | 10.1.33.0/20 | 售后部 |          |       |       |          |       |
| 新规划1 |                    |              |              | 售后部 | 8        | 1     |       |          |       |
| 新规划1 |                    |              | 10.1.34.0/24 | 售后部 | 8        | 1     | 维修组 |          |       |
| 新规划1 |                    |              |              | 售后部 | 8        | 1     | 维修组 | 16       | 1     |
| 新规划1 |                    |              |              | 售后部 | 8        | 1     | 后勤组 | 16       | 2     |
| 新规划1 |                    |              | 10.1.40.0/20 | 人事部 |          |       |       |          |       |
| 新规划1 |                    |              |              | 人事部 | 8        | 3     |       |          |       |
| 新规划1 |                    |              |              | 人事部 | 8        | 3     | 行政组 | 8        | 2     |
| 新规划1 |                    |              |              | 研发部 |          |       |       |          |       |
| 新规划1 |                    |              |              | 研发部 |          |       | 测试组 |          |       |
1. 导入导出文件的编码格式只支持UTF-8编码。采用CSV格式，以逗号分隔每项数据。
2. 表头的字符串为保留字，不能修改。
3. 表头项：Level1，IPv6，MaxMaskWidth，IPv4为必有字段，不可缺少。Level[N]（N>=2）根据实际层级需要递增添加，每增加一级Level，需同时增加LevelX, BitWidth, Value三个字段。
4. 数据行：按照网络树深度优先的方式递归输出。多棵网络树按照地址顺序依次排列。
5. 根PlanNode行：根网络节点的前缀仅输出一次，其下输出的所有语义节点均表示属于该网络前缀，直到遇到下一个根网络节点。
6. Ipv4行：如果某地址前缀所属的语义节点(包括根语义节点)包含IPv4地址，则先输出IPv4地址，每个地址一行；同时输出该语义节点的名称（在对应的LevelX项表示），但不输出该语义节点的PlanNode信息（根语义节点的IPv6和MaxMaskWidth，其他节点的bitwidth和value）。
7. 非根PlanNode行：除了IPv4行外，每行输出一个语义节点的一个网络节点信息，同时在其他Level输出其所有的父网络节点信息，但不输出该网络节点的子节点信息。
8. 纯语义节点行：根网络节点下如果存在纯语义节点，则该语义节点的PlanNode信息（BitWidth, Value）均为空，并且该纯语义节点的所有子节点的PlanNode信息也均为空。
9. 一个语义节点可能包含多个网络前缀，分别表示在不同的网络前缀下。

导入的规则：
1. 遵从上述所有规则。
2. 每行数据必定属于某个语义节点，但可能不包含网络节点信息，用最后一个LevelX不为空的方式，表示该行数据属于哪个语义节点。
3. IPv6和IPv4地址均不允许重复出现。

### 关于Plan的下发与上报
1. 功能解释与控制开关
下发：是指上级系统把自己的某个语义节点，通过调用下级系统API（action=Dispatch）的方式，在下级系统创建以该语义节点为根语义节点的Plan的过程。
上报：是指下级系统把上级系统下发的语义节点（以及新创建的语义节点），通过调用上级系统API（action=Report）的方式，回传给上级系统，并自动更新该语义节点所在Plan的过程。
下发撤销：是指上级系统对已下发的操作进行撤销的过程，是通过调用下级系统API（action=Repeal）的方式，删除以该语义节点为根节点的Plan的过程；但如果该Plan在下级系统中已被规划，且其网络前缀已被委派，DHCP，或再次下发，则该Plan无法被删除，下发撤销也将失败。

下发与上报功能由 系统管理 -> 系统联动 下的配置开关控制，其中：
  下发服务设置：
  - 在某一级系统设置“启动下发任务”，表示本系统能够向下级系统进行下发操作，同时也表示允许下级系统向本系统进行上报操作。
  - “系统名称”和“IP地址”存储的是下级系统的名称和IP。
  上报服务设置：
  - 在某一级系统设置“启动上报任务”，并在“上级系统IP地址”中设置对应IP地址，表示本系统能够且仅接受对应IP地址的上级系统的下发操作，也表示本系统可以向对应IP地址的上级系统进行上报操作。

2. 下发与上报对前端的含义与要求
下发与上报的所有Request消息体结构均为：DispatchRequest，定义如下：

| 参数名称        | 是否必填   | 数据类型          | 备注                                                     |
| -------------- | -------- | ---------------- | ------------------------------------------------------- |
| RemoteAddr     | 是       | string           | 调用方的IP地址和端口，如10.0.0.183:58081                     |
| SemanticNodes  | 是       | []*SemanticNode  | 下发或上报的语义节点数组，如果是撤销Action，则只需要根语义节点信息  |

在上级和子级系统的下发与上报功能均已使能的情况下，前端调用接口的方式如下：
（此操作包括如下参与方：A系统前端，A系统后端，B系统前端，B系统后端）

A系统下发到B系统：
 - A系统前端调用B系统后端API（B/apis/linkingthing.com/ipam/v1/plans?action=dispatch），填充选定的语义节点信息。
 - 如果上一步执行成功（无错误信息返回），则A系统前端再调用A系统后端对应plan的update API（A/apis/linkingthing.com/ipam/v1/plans/xxxx），在下发的特定语义节点中加入Dispatch信息，并更新到A系统后端。
B系统上报到A系统：
 - B系统前端调用A系统后端API（A/apis/linkingthing.com/ipam/v1/plans?action=report），填充以之前下发的语义节点为根语义节点的规划后的语义节点数组。
 - 如果上一步执行成功（无错误信息返回），则B系统前端再调用B系统后端对应plan的update API（A/apis/linkingthing.com/ipam/v1/plans/xxxx），在plan中更新Dispatch信息，并保存到B系统后端。
A系统撤销下发：
 - A系统前端调用B系统后端API（B/apis/linkingthing.com/ipam/v1/plans?action=repeal），填充之前下发的根语义节点信息。
 - 如果上一步执行成功（无错误信息返回），则A系统前端再调用A系统后端对应plan的update API（A/apis/linkingthing.com/ipam/v1/plans/xxxx），在下发的特定语义节点中删除Dispatch信息，并更新到A系统后端。

### 关于Action 
注意:除特殊没有planId以外,所有的action在url格式为:xxx/planId?action=xxx。

* 更新plan信息:updateplaninfo  
目前只修改plan的名称，即name。
传入参数:plan的实体，有效字段为:id,name。只更新name。
返回:成功即200，否则为失败。

* 更新语义名称:updatesemanticinfo  
目前只更新semanticNode的name，其余字段忽略。
传入参数:semanticNode实体，有效字段:id,name。
返回:成功即200，否则为失败。

* 一键规划与自定义规划:autoformulatesemantic  
入参:
```text
{
    #当前语义节点的ID
    "parentSemanticId": {"type": "string"},
    #需要选择规划的语义前缀，一键规划则传当前语义节点所有的前缀，自定义规划填充选择的前缀
    "prefixs": {"type":"array","elemType": "string"},
    #需要规划的子语义节点，内容是子语义的数组。核心参数是id。一键规划填入所有的子语义节点，自定义填入选择的子语义。
    "semanticNodes": {"type":"array","elemType": "SemanticNode"}
}
如果请求成功会返回以下结构体:
{
    "state": {"type": "string"},
    "message" :{"type": "string"}
}
```
注意:(state=conflict，同时message=冲突子网)表示有冲突的地址委派  
此时需要弹窗让客户选择:当前规划中message已经被前缀委派，给出2个选项，【手动处理】或者【忽略】，手动处理即取消的意思，
忽略则需要请求另一个action:formulateignoreconflict，并且请求内容不变，这里最好给出提示如果忽略会造成地址使用不了等严重后果。

* 一键规划忽略冲突:formulateignoreconflict  
入参参考autoformulatesemantic
返回200即成功，成功的同时会生产一条告警信息。

* 清空规划:cleansemanticplannodes  
请求参数同autoformulatesemantic
返回200即成功。

* 新增地址块:addsemanticplannode  
返回处理模式同autoformulatesemantic，如果state=conflict同样需要弹窗提醒用户选择。
```text
"input": {
    "parentSemanticId": {"type": "string"},
    "prefixs": {"type":"array","elemType": "string"},
    "prefixNumbers": {"type":"array","elemType": "string"},
    "semantic": {"type":"struct","elemType": "SemanticNode"}
},
"output": {
    "state": {"type": "string"},
    "message" :{"type": "string"},
}
```

* 新增地址块忽略冲突:addplannodeignoreconflict  
入参同addsemanticplannode
返回200即成功

* 删除语义:deletesemantic  
请求参数:语义结构题，核心参数为id。
返回200即成功。

* 修改语义ipv4:updatesemanticipv4  
入参:语义结构体:核心参数:id,ipv4s
返回200即成功

* 更新语义节点数量以及地址块个数:updatesemanticnumber  
入参:
```text
{
    "semanticId": {"type": "string"},#更新的语义id
    "subNodeNumbers": {"type": "int","description": "sub semanticNode numbers"},//子语义的个数
    "stepSize": {"type": "int","description": "step length"}//地址块个数
}
```
返回:"semanticNodes": {"type":"array","elemType": "SemanticNode"}，返回新增的语义结构体数组。
注意:如果返回的数组不为空，那么再次请求:autoformulatesemantic，其中semanticNodes填充为返回的语义数组。

* 更新语义位宽:updatebitwidth  
入参:语义结构体，核心参数:id,stepsize。
返回200即成功。


