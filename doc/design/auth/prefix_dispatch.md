# Prefix_Dispatch
地址规划下发上报角色控制，目的用于子系统获取token以及下发上报使用。除了地址规划其他无任何权限。

## 设计
1.ddi-role.json新增一个角色:DISPATCH，配置如下:
```text
{
      "role": "DISPATCH",
      "baseAuthority": [],
      "dnsAuthority": [],
      "dhcpAuthority": [
        {
          "resource": "plan",
          "operations": [
            "ACTION",
            "POST"
          ]
        }
      ]
    }
```
该角色只拥有plan操作权限。并且角色账号密码为默认初始值:dispatch|123456。md5（f5e9b944416b4f63fce2319e563a4088）以后提供给前端调用请求GetToken

2.新增GetToken方法，用于分级系统获取token，通过拿到token然后去请求操作。例如：  
B/apis/linkingthing.com/ipam/v1/plans?action=dispatch  
B/apis/linkingthing.com/ipam/v1/plans?action=repeal  
这样通过前端请求B分级系统直接进行下发以及撤销下发操作。

3.授权  
3.1 10.0.0.2启动子系统，并且初始化生成dispatch|123456,角色为DISPATCH的用户。  
3.2 管理系统10.0.0.1配置启动下发服务，并且配置下发服务器地址，这里与子系统A，IP为10.0.0.2。  
3.3 管理系统选择:语义节点名为上海，前缀为2008::/64进行下发，选择子系统A。  
3.4 客户端请求10.0.0.2服务器调用GetToken获取token。  
3.5 客户端请求10.0.0.2服务器调用/apis/linkingthing.com/ipam/v1/plans?action=dispatch进行下发操作。
3.6 10.0.0.2子系统收到dispatch请求，通过token校验账号。满足user以及DISPATCH角色方可通过。
3.7 客户登陆10.0.0.2子系统，只能看到前缀为2008::/64的语义，并且对该语义进行操作。  
3.8 客户更新系统联动，启动上报服务，并且填上上级系统的IP:10.0.0.1。
3.9 客户上报规划结果，请求到10.0.0.1，接口为：apis/linkingthing.com/ipam/v1/plans?action=report。

4.鉴权  
4.1 GetToken鉴定是否为默认角色以及请求信息校验。
4.2 DISPATCH只拥有plan-ACTION-dispatch|repeal|report权限。
4.3 子系统不能删除下发的语义plan。