# AgentEvent
## 概览
配置反显功能，当controller创建、更新、删除dns以及dhcp相关的操作时，通过kafka异步下发消息，然后dns以及dhcp进行消费并做相应的操作。
而这里存在的问题是无法知道dns以及dhcp操作成功与否，配置反显的目的就是一目了然的看到dns以及dhcp操作反馈。

## 设计
反显流程：
* 客户端操作dns或者dhcp，例如创建、更新、删除。
* controller通过验证处理完毕入库并且下发kafka消息。
* dns或者dhcp服务器消费kafka消息，然后根据下发命名做相应的操作，并且操作完毕通过kafka上报反馈成功状态以及失败原因。
* controller消费kafka消息，收到agentevent消息后入库并且存入内存中，用链表形式存储，
其中目前设定只存1000条数据，超过1000条就删除最老的数据，并且每隔30天或者controller重启的时候检测数据库数据内容，清理超过1000条的数据或者30以前的数据。
* 客户端登陆系统以后会与controller建立长链接，首次建立连接以后服务器会推送链表中的所以消息给客户端，以后则增量推送。客户端也保持最多1000条记录。超过的自行删除。
* 用户通过配置反显消息能清晰的了解到dns以及dhcp操作是否成功。

## 资源
#### 配置反显（AgentEvent）
* 顶级资源：Node（节点IP），NodeType（节点类型），Resource（操作资源），Method（操作方法），Succeed（是否成功），
ErrorMessage（失败信息），CmdMessage（操作具体内容），OperationTime（操作时间）。
* 只支持websocket连接获取数据，并且是服务器主动推送。
* 当且Succeed为false的时候才会有ErrorMessage数据。
* CmdMessage内的信息包含了controller操作的具体内容，可以清晰的查看到操作的具体资源以及数据等。


