# 应用资产接口设计

## 资源地址 /apis/linkingthing.com/ipam/v1/appassets

### 创建应用资产

#### 参数描述：
* appname: 应用名称
* domain: 域名
* semantic: 所属组织
* apptype: 应用类型
* business: 业务
* supportdoublenetwork: 双栈访问
* servertype: 服务器类型
* operatesupport: 运维人员
* phonenumber: 联系方式
* remark: 备注

#### 请求
curl --location --request POST 'https://localhost:58081/apis/linkingthing.com/ipam/v1/appassets' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTU0NDYyNTMsIlVzZXJOYW1lIjoiYWRtaW4ifQ.8AcL3Mlkg3wWOt4Pu3kqDaMnAfadNZbacQ_EpNgiYBc' \
--header 'Content-Type: application/json' \
--data-raw '{
    "appname": "myapp3",
    "domain": "www.test.com",
    "semantic": "中国 -> 四川 -> 成都",
    "apptype": "内部应用",
    "business": "test",
    "supportdoublenetwork": true,
    "supportdns64": true,
    "servertype": "物理服务器",
    "operatesupport": "liz2",
    "phonenumber": "111111",
    "remark": "fuyan"
}'

#### 返回
{
    "id": "b75cf857404ec9428082e639a2fe694f",
    "type": "appasset",
    "links": {
        "collection": "/apis/linkingthing.com/ipam/v1/appassets",
        "remove": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f",
        "self": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f",
        "update": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f"
    },
    "creationTimestamp": "2021-03-10T15:19:53+08:00",
    "deletionTimestamp": null,
    "appname": "myapp3",
    "domain": "www.test.com",
    "semantic": "中国 -> 四川 -> 成都",
    "apptype": "内部应用",
    "business": "test",
    "supportdoublenetwork": true,
    "supportdns64": true,
    "servertype": "物理服务器",
    "accesscount": 0,
    "operatesupport": "liz2",
    "phonenumber": "111111",
    "remark": "fuyan"
}

### 条件查询
可以根据参数查询应用资产

#### 参数描述：
* domain - 域名
* semantic - 所属组织
* business - 业务
* app_type - 应用类型
* operate_support - 运维人员
* phone_number - 联系方式

#### 请求
curl --location --request GET 'https://localhost:58081/apis/linkingthing.com/ipam/v1/appassets?phone_number=111111' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTU0NDYyNTMsIlVzZXJOYW1lIjoiYWRtaW4ifQ.8AcL3Mlkg3wWOt4Pu3kqDaMnAfadNZbacQ_EpNgiYBc'

#### 返回
{
    "type": "collection",
    "resourceType": "appasset",
    "links": {
        "self": "/apis/linkingthing.com/ipam/v1/appassets"
    },
    "pagination": {},
    "data": [
        {
            "id": "b75cf857404ec9428082e639a2fe694f",
            "type": "appasset",
            "links": {
                "collection": "/apis/linkingthing.com/ipam/v1/appassets",
                "remove": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f",
                "self": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f",
                "update": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f"
            },
            "creationTimestamp": "2021-03-10T15:19:53+08:00",
            "deletionTimestamp": null,
            "appname": "myapp3",
            "domain": "www.test.com",
            "semantic": "中国 -> 四川 -> 成都",
            "apptype": "内部应用",
            "business": "test",
            "supportdoublenetwork": false,
            "supportdns64": false,
            "servertype": "物理服务器",
            "accesscount": 0,
            "operatesupport": "liz2",
            "phonenumber": "111111",
            "remark": "fuyan"
        }
    ]
}

### 根据ID查询

#### 参数 ： 无

#### 请求
curl --location --request GET 'https://localhost:58081/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTU0NDYyNTMsIlVzZXJOYW1lIjoiYWRtaW4ifQ.8AcL3Mlkg3wWOt4Pu3kqDaMnAfadNZbacQ_EpNgiYBc'

#### 返回
{
    "id": "b75cf857404ec9428082e639a2fe694f",
    "type": "appasset",
    "links": {
        "collection": "/apis/linkingthing.com/ipam/v1/appassets",
        "remove": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f",
        "self": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f",
        "update": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f"
    },
    "creationTimestamp": "2021-03-10T15:19:53+08:00",
    "deletionTimestamp": null,
    "appname": "myapp3",
    "domain": "www.test.com",
    "semantic": "中国 -> 四川 -> 成都",
    "apptype": "内部应用",
    "business": "test",
    "supportdoublenetwork": false,
    "supportdns64": false,
    "servertype": "物理服务器",
    "accesscount": 0,
    "operatesupport": "liz2",
    "phonenumber": "111111",
    "remark": "fuyan"
}

### 修改接口

#### 参数

{
    "appname": "myapp4",
    "domain": "www.test.com1",
    "semantic": "中国 -> 四川 -> 成都1",
    "apptype": "内部应用",
    "business": "test1",
    "supportdoublenetwork": true,
    "supportdns64": true,
    "servertype": "物理服务器2",
    "operatesupport": "liz22",
    "phonenumber": "2222222",
    "remark": "fuyan2"
}

#### 参数描述：
* appname: 应用名称
* domain: 域名
* semantic: 所属组织
* apptype: 应用类型
* business: 业务
* supportdoublenetwork: 双栈访问
* servertype: 服务器类型
* operatesupport: 运维人员
* phonenumber: 联系方式
* remark: 备注

#### 请求
curl --location --request PUT 'https://localhost:58081/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTU0NDYyNTMsIlVzZXJOYW1lIjoiYWRtaW4ifQ.8AcL3Mlkg3wWOt4Pu3kqDaMnAfadNZbacQ_EpNgiYBc' \
--header 'Content-Type: application/json' \
--data-raw '{
    "appname": "myapp4",
    "domain": "www.test.com1",
    "semantic": "中国 -> 四川 -> 成都1",
    "apptype": "内部应用",
    "business": "test1",
    "supportdoublenetwork": true,
    "supportdns64": true,
    "servertype": "物理服务器2",
    "operatesupport": "liz22",
    "phonenumber": "2222222",
    "remark": "fuyan2"
}'

#### 返回
{
    "id": "b75cf857404ec9428082e639a2fe694f",
    "type": "appasset",
    "links": {
        "collection": "/apis/linkingthing.com/ipam/v1/appassets",
        "remove": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f",
        "self": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f",
        "update": "/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f"
    },
    "creationTimestamp": null,
    "deletionTimestamp": null,
    "appname": "myapp4",
    "domain": "www.test.com1",
    "semantic": "中国 -> 四川 -> 成都1",
    "apptype": "内部应用",
    "business": "test1",
    "supportdoublenetwork": true,
    "supportdns64": true,
    "servertype": "物理服务器2",
    "accesscount": 0,
    "operatesupport": "liz22",
    "phonenumber": "2222222",
    "remark": "fuyan2"
}

### 删除应用资产

#### 请求
curl --location --request DELETE 'https://localhost:58081/apis/linkingthing.com/ipam/v1/appassets/b75cf857404ec9428082e639a2fe694f' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTU0NDYyNTMsIlVzZXJOYW1lIjoiYWRtaW4ifQ.8AcL3Mlkg3wWOt4Pu3kqDaMnAfadNZbacQ_EpNgiYBc' \
--data-raw ''

#### 返回 无
