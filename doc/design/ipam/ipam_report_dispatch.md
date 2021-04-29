# 下发上报

## 资源设计

### 下发配置资源
名称：ipdispatchconfigs

属性: enableDispatch - 启用开关

查询集合接口：GET https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs

参数：无

示例：

curl --location --request GET 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--data-raw ''

返回:

{
    "type": "collection",
    "resourceType": "ipdispatchconfig",
    "links": {
        "self": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs"
    },
    "pagination": {},
    "data": [
        {
            "id": "ipdispatchconfigid",
            "type": "ipdispatchconfig",
            "links": {
                "collection": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs",
                "dispatchclients": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients",
                "self": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid",
                "update": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid"
            },
            "creationTimestamp": "2021-02-28T23:08:49-05:00",
            "deletionTimestamp": null,
            "enableDispatch": false
        }
    ]
}

主键查询接口：GET https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid

参数：无

示例：

curl --location --request GET 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--data-raw ''

返回

{
    "id": "ipdispatchconfigid",
    "type": "ipdispatchconfig",
    "links": {
        "collection": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs",
        "dispatchclients": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients",
        "self": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid",
        "update": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid"
    },
    "creationTimestamp": "2021-02-28T23:08:49-05:00",
    "deletionTimestamp": null,
    "enableDispatch": false
}

更新接口： PUT https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid

参数： {"enableDispatch": false}

示例：

curl --location --request PUT 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--header 'Content-Type: application/json' \
--data-raw '{"enableDispatch": false}'

返回：

{
    "id": "ipdispatchconfigid",
    "type": "ipdispatchconfig",
    "links": {
        "collection": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs",
        "dispatchclients": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients",
        "self": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid",
        "update": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid"
    },
    "creationTimestamp": null,
    "deletionTimestamp": null,
    "enableDispatch": false
}


### 上报配置资源

名称： ipreportconfigs

属性： 
    enableReport - 启用开关
    reportServerAddr - ip地址 

集合查询接口：GET https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipreportconfigs

参数： 无

示例：

curl --location --request GET 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipreportconfigs' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs'

返回：

{
    "type": "collection",
    "resourceType": "ipreportconfig",
    "links": {
        "self": "/apis/linkingthing.com/ipam/v1/ipreportconfigs"
    },
    "pagination": {},
    "data": [
        {
            "id": "ipreportconfigid",
            "type": "ipreportconfig",
            "links": {
                "collection": "/apis/linkingthing.com/ipam/v1/ipreportconfigs",
                "self": "/apis/linkingthing.com/ipam/v1/ipreportconfigs/ipreportconfigid",
                "update": "/apis/linkingthing.com/ipam/v1/ipreportconfigs/ipreportconfigid"
            },
            "creationTimestamp": "2021-02-28T23:04:51-05:00",
            "deletionTimestamp": null,
            "enableReport": false,
            "reportServerAddr": ""
        }
    ]
}

主键查询接口: GET https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipreportconfigs/ipreportconfigid

参数：无

示例

curl --location --request GET 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipreportconfigs/ipreportconfigid' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs'

返回

{
    "id": "ipreportconfigid",
    "type": "ipreportconfig",
    "links": {
        "collection": "/apis/linkingthing.com/ipam/v1/ipreportconfigs",
        "self": "/apis/linkingthing.com/ipam/v1/ipreportconfigs/ipreportconfigid",
        "update": "/apis/linkingthing.com/ipam/v1/ipreportconfigs/ipreportconfigid"
    },
    "creationTimestamp": "2021-02-28T23:04:51-05:00",
    "deletionTimestamp": null,
    "enableReport": false,
    "reportServerAddr": ""
}

更新接口：PUT https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipreportconfigs/ipreportconfigid

参数：{"enableReport": true, "reportServerAddr": "10.0.0.156"}

示例：

curl --location --request PUT 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipreportconfigs/ipreportconfigid' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--header 'Content-Type: application/json' \
--data-raw '{"enableReport": true, "reportServerAddr": "10.0.0.156"}'

返回

{
    "id": "ipreportconfigid",
    "type": "ipreportconfig",
    "links": {
        "collection": "/apis/linkingthing.com/ipam/v1/ipreportconfigs",
        "self": "/apis/linkingthing.com/ipam/v1/ipreportconfigs/ipreportconfigid",
        "update": "/apis/linkingthing.com/ipam/v1/ipreportconfigs/ipreportconfigid"
    },
    "creationTimestamp": null,
    "deletionTimestamp": null,
    "enableReport": true,
    "reportServerAddr": "10.0.0.156"
}

### 下发客户资源

名称： dispatchclients

属性： 
    name - 名称
    clientaddr - ip地址
    remark - 备注

集合查询接口： GET https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients

参数： 
{
    "name: "test",
    "clientaddr": "10.0.0.1"
}

示例

curl --location --request GET 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--header 'Content-Type: text/plain' \
--data-raw '{
    "name: "2",
    "clientaddr": "10.0.0.2"
}'

返回

{
    "type": "collection",
    "resourceType": "dispatchclient",
    "links": {
        "self": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients"
    },
    "pagination": {},
    "data": [
        {
            "id": "fd55bb4d4064ce9d80cd7c5307f9c904",
            "type": "dispatchclient",
            "links": {
                "collection": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients",
                "remove": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904",
                "self": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904",
                "update": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904"
            },
            "creationTimestamp": "2021-02-28T23:32:44-05:00",
            "deletionTimestamp": null,
            "name": "2",
            "clientAddr": "10.0.0.2"
        }
    ]
}

主键查询接口: GET https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904

参数：无

示例

curl --location --request GET 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--data-raw ''

返回

{
    "id": "fd55bb4d4064ce9d80cd7c5307f9c904",
    "type": "dispatchclient",
    "links": {
        "collection": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients",
        "remove": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904",
        "self": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904",
        "update": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904"
    },
    "creationTimestamp": "2021-02-28T23:32:44-05:00",
    "deletionTimestamp": null,
    "name": "2",
    "clientAddr": "10.0.0.2"
}

创建接口： POST https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients

参数：
{
    "name": "2",
    "clientAddr": "10.0.0.2",
    "remark": "222"
}

示例

curl --location --request POST 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name": "2",
    "clientAddr": "10.0.0.2"
}'

返回

{
    "id": "fd55bb4d4064ce9d80cd7c5307f9c904",
    "type": "dispatchclient",
    "links": {
        "collection": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients",
        "remove": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904",
        "self": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904",
        "update": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904"
    },
    "creationTimestamp": "2021-02-28T23:32:44-05:00",
    "deletionTimestamp": null,
    "name": "2",
    "clientAddr": "10.0.0.2"
}

更新接口： PUT https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904

参数：
{
    "name": "7",
    "clientAddr": "10.0.0.7"
}

示例

curl --location --request PUT 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name": "7",
    "clientAddr": "10.0.0.7"
}'

返回

{
    "id": "fd55bb4d4064ce9d80cd7c5307f9c904",
    "type": "dispatchclient",
    "links": {
        "collection": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients",
        "remove": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904",
        "self": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904",
        "update": "/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904"
    },
    "creationTimestamp": null,
    "deletionTimestamp": null,
    "name": "7",
    "clientAddr": "10.0.0.7"
}

导入接口：POST https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients?action=importcsv

参数： {"name":"dispatchClient-2021-02-24 01:54:03.csv"}

示例

curl --location --request POST 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients?action=importcsv' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--header 'Content-Type: text/plain' \
--data-raw '{"name":"dispatchClient-2021-02-24 01:54:03.csv"}'

返回

[
    {
        "id": "985a393a406ca2df80cbb8641dd56a60",
        "creationTimestamp": "2021-03-01T00:16:23-05:00",
        "deletionTimestamp": null,
        "name": "1",
        "clientAddr": "10.0.0.1"
    },
    {
        "id": "ea2cbac4404ba5798003a9f939d3b436",
        "creationTimestamp": "2021-03-01T00:16:23-05:00",
        "deletionTimestamp": null,
        "name": "2",
        "clientAddr": "10.0.0.2"
    },
    {
        "id": "eaecc1a5408f18c98024d12c0cc3a8b1",
        "creationTimestamp": "2021-03-01T00:16:23-05:00",
        "deletionTimestamp": null,
        "name": "3",
        "clientAddr": "10.0.0.3"
    },
    {
        "id": "6799fa9140a266a180e6a34e6c8584aa",
        "creationTimestamp": "2021-03-01T00:16:23-05:00",
        "deletionTimestamp": null,
        "name": "4",
        "clientAddr": "10.0.0.4"
    },
    {
        "id": "aefe861d4073912e80145860694f4a5d",
        "creationTimestamp": "2021-03-01T00:16:23-05:00",
        "deletionTimestamp": null,
        "name": "5",
        "clientAddr": "10.0.0.5"
    }
]

导出接口：POST https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients?action=exportcsv

参数： 无

示例

curl --location --request POST 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients?action=exportcsv' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--data-raw ''

返回

{
    "path": "/public/dispatchClient-2021-03-01 00:19:21.csv"
}


导出模板接口：POST https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients?action=exportcsvtemplate

参数 ： 无

示例

curl --location --request POST 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients?action=exportcsvtemplate' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--data-raw ''

返回

{
    "path": "/public/dc-template.csv"
}

删除接口 DELETE https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/fd55bb4d4064ce9d80cd7c5307f9c904

参数： 无

示例

curl --location --request DELETE 'https://10.0.0.156:58081/apis/linkingthing.com/ipam/v1/ipdispatchconfigs/ipdispatchconfigid/dispatchclients/135993a640f619cd8069ce93477141ef' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ2NTc5MDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.QTFAHGc8CnlS4qMkt_EcfAypClxTxE7pHyVwrul2shs' \
--data-raw ''

返回： 空