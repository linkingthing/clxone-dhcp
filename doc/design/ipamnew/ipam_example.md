* semantic
```text
List:
curl --location --request GET 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semantics' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3MjMzMDQsIlVzZXJOYW1lIjoiYWRtaW4ifQ.xojJM-wnjDeYPCXFgpX4sPE-b3lkwA0eRjysPtXqi0U'

Create:
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semantics' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3MjMzMDQsIlVzZXJOYW1lIjoiYWRtaW4ifQ.xojJM-wnjDeYPCXFgpX4sPE-b3lkwA0eRjysPtXqi0U' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name": "北京连星科技",
    "parentId": "root",
    "rootId": "root"
}'

Put:
curl --location --request PUT 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semantics/2083d41340e35d2c80a0ef44ecb0800f' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3MjMzMDQsIlVzZXJOYW1lIjoiYWRtaW4ifQ.xojJM-wnjDeYPCXFgpX4sPE-b3lkwA0eRjysPtXqi0U' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name": "北京连星科技集团",
    "parentId": "root",
    "rootId": "root"
}'

Delete:
curl --location --request DELETE 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semantics/2083d41340e35d2c80a0ef44ecb0800f' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3MjMzMDQsIlVzZXJOYW1lIjoiYWRtaW4ifQ.xojJM-wnjDeYPCXFgpX4sPE-b3lkwA0eRjysPtXqi0U' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name": "北京连星科技集团",
    "parentId": "root",
    "rootId": "root"
}'

action:list_tree
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semantics/ead5bb32402a648880574a6ab83e9a27?action=list_tree' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3MjM5NDgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.PRd-b7OBHoSZEfaZudUoWdIMlzNFumtE5p17VHp0vms' \
--header 'Content-Type: application/json' 

action:create_subnode
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semantics/ead5bb32402a648880574a6ab83e9a27?action=create_subnode' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3MjY3NjgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.z9EUkYzxiCndo97oKvlK0h9bDH0rbm3eh9MLrxSyGqk' \
--header 'Content-Type: application/json' \
--data-raw '{
    "semanticId": "ead5bb32402a648880574a6ab83e9a27",
    "semanticNodes": [
        {
            "name": "成都连星",
            "parentId": "ead5bb32402a648880574a6ab83e9a27",
            "rootId": "ead5bb32402a648880574a6ab83e9a27"
        },
        {
            "name": "重庆连星",
            "parentId": "ead5bb32402a648880574a6ab83e9a27",
            "rootId": "ead5bb32402a648880574a6ab83e9a27"
        }
    ]
}'

```

* networktype
```text
List:
curl --location --request GET 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networktypes' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQwNDUwNzgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.gsHhJCXzChobOVsLdQNALlclsRunWDL9auG6mRthhR4' \
--header 'Content-Type: application/json'

Get:
curl --location --request GET 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networktypes/management' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQwNDUwNzgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.gsHhJCXzChobOVsLdQNALlclsRunWDL9auG6mRthhR4'

Create:
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networktypes' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQwNDUwNzgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.gsHhJCXzChobOVsLdQNALlclsRunWDL9auG6mRthhR4' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name": "test",
    "custom": true,
    "comment": "just test"
}'

Update:
curl --location --request PUT 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networktypes/test' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQwNDU1NDQsIlVzZXJOYW1lIjoiYWRtaW4ifQ.hUX_czX2FS1WFJLFU1XrLsL3Ezp0Q2ui5xCzDw2FIzY' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name": "test",
    "custom": true,
    "comment": "just test 123"
}'

Delete:
curl --location --request DELETE 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networktypes/test' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQwNDU1NDQsIlVzZXJOYW1lIjoiYWRtaW4ifQ.hUX_czX2FS1WFJLFU1XrLsL3Ezp0Q2ui5xCzDw2FIzY'


```

* networkv6
```text
List:
curl --location --request GET 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networkv6s' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3MjIwMzEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.C7GxhJZ48vtu6ekPiJD-c_VojsaGELl2YX9zEXxl4_I'

Create:
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networkv6s' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3MjgxMzUsIlVzZXJOYW1lIjoiYWRtaW4ifQ.fDbCCWaqbQcy_YjueSwEb_TKJKLNdCfq0srB2aYump0' \
--header 'Content-Type: application/json' \
--data-raw '{
    "prefix": "2001::/32",
    "name": "小试牛刀",
    "semanticName": "北京连星科技",
    "semanticId": "ead5bb32402a648880574a6ab83e9a27",
    "networkType": "office",
    "createMode": "manual",
    "business": "test",
    "comment": ""
}'

Update:
curl --location --request PUT 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networkv6s/0f43f17f4006a407800a877a5c76998c' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3MjgxMzUsIlVzZXJOYW1lIjoiYWRtaW4ifQ.fDbCCWaqbQcy_YjueSwEb_TKJKLNdCfq0srB2aYump0' \
--header 'Content-Type: application/json' \
--data-raw '{
    "prefix": "2001::/32",
    "name": "小试牛刀123",
    "semanticName": "北京连星科技",
    "semanticId": "ead5bb32402a648880574a6ab83e9a27",
    "networkType": "office",
    "createMode": "manual",
    "business": "test",
    "comment": ""
}'

Delete:
curl --location --request DELETE 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networkv6s/9c18b91540753d1180a48fd3f9b07a96' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3MjkxODgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.evZsZvU87qfrUoPvzm_nBnG2-zJKSaEo4KZ-W1h8vJE' \
--header 'Content-Type: application/json'

Get:
curl --location --request GET 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networkv6s/ddee76a5405efc6780d480158a6ece4b' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3MjkzNDcsIlVzZXJOYW1lIjoiYWRtaW4ifQ.apXN0UZaYFu4cZ926ja7azFQvu9cFZTJ_FtoXmfHork'

```

* networkv4
```text
List:
curl --location --request GET 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networkv4s/' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3Mjk3NjYsIlVzZXJOYW1lIjoiYWRtaW4ifQ.DWwNixEZWGY40jzLSqAa6gwTe-oVjGqtDxaY3LOKoEg'

Create:
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networkv4s' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3Mjk4ODIsIlVzZXJOYW1lIjoiYWRtaW4ifQ.OZDbRgL5QYoZyxvrC8QBtA5znJz2TOjzfZ3GIyCH1Ec' \
--header 'Content-Type: application/json' \
--data-raw '{
    "prefix": "10.0.0.0/24",
    "name": "小试牛刀",
    "semanticName": "北京连星科技",
    "semanticId": "ead5bb32402a648880574a6ab83e9a27",
    "networkType": "office",
    "createMode": "manual",
    "business": "test",
    "comment": ""
}'

Update:
curl --location --request PUT 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networkv4s/b13f9d2e4052cd6480d561e8a47143cb' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3Mjk4ODIsIlVzZXJOYW1lIjoiYWRtaW4ifQ.OZDbRgL5QYoZyxvrC8QBtA5znJz2TOjzfZ3GIyCH1Ec' \
--header 'Content-Type: application/json' \
--data-raw '{
    "prefix": "10.0.0.0/24",
    "name": "小试牛刀33",
    "semanticName": "北京连星科技",
    "semanticId": "ead5bb32402a648880574a6ab83e9a27",
    "networkType": "office",
    "createMode": "manual",
    "business": "test",
    "comment": ""
}'

Delete:
curl --location --request DELETE 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networkv4s/b13f9d2e4052cd6480d561e8a47143cb' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3Mjk4ODIsIlVzZXJOYW1lIjoiYWRtaW4ifQ.OZDbRgL5QYoZyxvrC8QBtA5znJz2TOjzfZ3GIyCH1Ec' \
--header 'Content-Type: application/json'

Get:
curl --location --request GET 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/networkv4s/98b7a8a0402c0ef680570c5d4d448af8' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3Mjk4ODIsIlVzZXJOYW1lIjoiYWRtaW4ifQ.OZDbRgL5QYoZyxvrC8QBtA5znJz2TOjzfZ3GIyCH1Ec'

```

* semanticinfo
```text
List:
curl --location --request GET 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semanticinfos' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3ODM3ODQsIlVzZXJOYW1lIjoiYWRtaW4ifQ.8IcS7kTmvlCWr5sj-3kWUrNPsihoTGsHBwZKJnHJYqA'

action:list_semantic_info
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semanticinfos/ead5bb32402a648880574a6ab83e9a27?action=list_semantic_info' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3ODQyMTIsIlVzZXJOYW1lIjoiYWRtaW4ifQ.s1NYX68Wrg1QRho3dffgcSp0oRQr00b3fQ4txtg7loE'

action:list_v4
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semanticinfos/ead5bb32402a648880574a6ab83e9a27?action=list_v4' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3ODQ5NDAsIlVzZXJOYW1lIjoiYWRtaW4ifQ.ABb7f_tav3oT6g2XlqwUB3BmYaVqhzGhQQ3LUwwxJe4' \
--header 'Content-Type: application/json' \
--data-raw '{
    "semanticId": "ead5bb32402a648880574a6ab83e9a27"
}'

action:list_v6
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semanticinfos/ead5bb32402a648880574a6ab83e9a27?action=list_v6' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3ODQ5NDAsIlVzZXJOYW1lIjoiYWRtaW4ifQ.ABb7f_tav3oT6g2XlqwUB3BmYaVqhzGhQQ3LUwwxJe4' \
--header 'Content-Type: application/json' \
--data-raw '{
    "semanticId": "ead5bb32402a648880574a6ab83e9a27"
}'

action:search_semantic
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semanticinfos/ead5bb32402a648880574a6ab83e9a27?action=search_semantic' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3ODU0MTcsIlVzZXJOYW1lIjoiYWRtaW4ifQ.7wmtbl9i-uHUojzTjoMgvJAJ90Er5SFu30K191e-5kE' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name": "重庆"
}'

```

* plan
```text
List:
curl --location --request GET 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/plans' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3ODU0MTcsIlVzZXJOYW1lIjoiYWRtaW4ifQ.7wmtbl9i-uHUojzTjoMgvJAJ90Er5SFu30K191e-5kE' \
--header 'Content-Type: application/json'

Create:
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/plans' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3OTE2NjcsIlVzZXJOYW1lIjoiYWRtaW4ifQ.0OYRe0ktBgfS6gJqQjiSSEqhzuDDNsGiWZKzeW6toJM' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name": "生产网",
    "semantic": "ead5bb32402a648880574a6ab83e9a27",
    "prefixes": ["2002::/32"]
}'

Update:
curl --location --request PUT 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/plans/5d35351f40a2d3ce8083a802a8c03eb0' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3OTI0NDEsIlVzZXJOYW1lIjoiYWRtaW4ifQ.8QHE54t3SxORNxk1Idtn4HjPKRsxLz1zmhJRhZw2jkA' \
--header 'Content-Type: application/json' \
--data-raw '{
    "name": "生产网123",
    "semantic": "ead5bb32402a648880574a6ab83e9a27",
    "prefixes": ["2002::/32"]
}'

Delete:
curl --location --request DELETE 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/plans/5d35351f40a2d3ce8083a802a8c03eb0' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3OTMwMzUsIlVzZXJOYW1lIjoiYWRtaW4ifQ.jTgOyxO3BBUNkFz3O8TWtPrWpNawtWZ8Qhk71N8qZ5Y' \
--header 'Content-Type: application/json'

action:list_plan_tree
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/plans/7cf8fe4e407532c280523e9684f46fb9?action=list_plan_tree' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM3OTM5NTQsIlVzZXJOYW1lIjoiYWRtaW4ifQ.8JgnkceoKTMfDmvXqChLivR4IJLH-dE8lE-zADowPEs' \
--header 'Content-Type: application/json'

action:update_semantic_plan
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/plans/7cf8fe4e407532c280523e9684f46fb9?action=update_semantic_plan' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM4MDU3OTMsIlVzZXJOYW1lIjoiYWRtaW4ifQ.hQzDjOHA6uFMi27fzyaGFAot178oxtWZdKU_QYHDtxw' \
--header 'Content-Type: application/json' \
--data-raw '{
    "nodeId": "ead5bb32402a648880574a6ab83e9a27",
    "bitWidth": 3,
    "subSemanticPrefixCount": 1,
    "prefixBeginValue": "2000"
}'

action:plan_prefix_v6
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/plans/7cf8fe4e407532c280523e9684f46fb9?action=plan_prefix_v6' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM4MDkzOTgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.A9oEvdzSE9-0h_3etKT_TF055x11icgcRSN-a2uxsnE' \
--header 'Content-Type: application/json' \
--data-raw '{
    "prefixs": [
        "2002::/32"
    ],
    "parentId": "ead5bb32402a648880574a6ab83e9a27",
    "subNodeIds": [
        "5722c7ce40f83f8880d9cb73c427f01d",
        "fc61971a40073cdc801636f39ab05800"
    ]
}'

action:update_prefix_v6
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/plans/7cf8fe4e407532c280523e9684f46fb9?action=update_prefix_v6' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM4MDkzOTgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.A9oEvdzSE9-0h_3etKT_TF055x11icgcRSN-a2uxsnE' \
--header 'Content-Type: application/json' \
--data-raw '{
    "prefixs": [
        "2002::/32"
    ],
    "prefixCounts": [
        2
    ],
    "parentId": "ead5bb32402a648880574a6ab83e9a27",
    "subNodeId": "fc61971a40073cdc801636f39ab05800"
}'

action:clean_prefix_v6
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/plans/7cf8fe4e407532c280523e9684f46fb9?action=clean_prefix_v6' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTM4MDkzOTgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.A9oEvdzSE9-0h_3etKT_TF055x11icgcRSN-a2uxsnE' \
--header 'Content-Type: application/json' \
--data-raw '{
    "prefixs": [
        "2002::/32"
    ],
    "parentId": "ead5bb32402a648880574a6ab83e9a27",
    "subNodeIds": [
        "5722c7ce40f83f8880d9cb73c427f01d",
        "fc61971a40073cdc801636f39ab05800"
    ]
}'

```

* dispatch
```
disptach-config:
List:
curl --location --request GET 'https://10.0.0.202:58081/apis/linkingthing.com/ipam/v1/dispatchconfigs' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ0MTA0ODgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.cQvzBadc-iuPHhCAABIYFsRH6IxUIjsslCWWAoermqk'

Get:
curl --location --request GET 'https://10.0.0.202:58081/apis/linkingthing.com/ipam/v1/dispatchconfigs/dispatchConfigDefaultId' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ0MTA0ODgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.cQvzBadc-iuPHhCAABIYFsRH6IxUIjsslCWWAoermqk'

Update:
curl --location --request PUT 'https://10.0.0.202:58081/apis/linkingthing.com/ipam/v1/dispatchconfigs/dispatchConfigDefaultId' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ0MTA0ODgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.cQvzBadc-iuPHhCAABIYFsRH6IxUIjsslCWWAoermqk' \
--header 'Content-Type: application/json' \
--data-raw '{
    "enableDispatch": true,
    "enableReport": false,
    "reportServerAddr": "",
    "dispatchClients": [
        {
            "dispatchConfig": "dispatchConfigDefaultId",
            "name": "10.0.0.202",
            "clientAddr": "10.0.0.202"
        }
    ]
}'

action:dispatch_forward
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semanticinfos/7c87d4f6400f325a80fcd1a5322b3fe4?action=dispatch_forward' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ0MTA0ODgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.cQvzBadc-iuPHhCAABIYFsRH6IxUIjsslCWWAoermqk' \
--header 'Content-Type: application/json' \
--data-raw '{
    "remoteAddr": "10.0.0.202",
    "semanticInfos": [
        {
            "id": "e9f8662d40c33c0380894383f74cc2f7"
        }
    ]
}'

action:repeal_forward
curl --location --request POST 'https://127.0.0.1:58081/apis/linkingthing.com/ipam/v1/semanticinfos/0ecc6acf40c30d3d808c4c8a6c2e814d?action=repeal_forward' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ0MTA0ODgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.cQvzBadc-iuPHhCAABIYFsRH6IxUIjsslCWWAoermqk' \
--header 'Content-Type: application/json' \
--data-raw '{
    "remoteAddr": "10.0.0.202",
    "semanticInfos": [
{
            "id": "e0dcee36400c11b680fd761438d8b161"
        }
    ]
}'

action:report_forward
curl --location --request POST 'https://10.0.0.202:58081/apis/linkingthing.com/ipam/v1/semanticinfos/e9f8662d40c33c0380894383f74cc2f7?action=report_forward' \
--header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MTQ0MTA0ODgsIlVzZXJOYW1lIjoiYWRtaW4ifQ.cQvzBadc-iuPHhCAABIYFsRH6IxUIjsslCWWAoermqk' \
--header 'Content-Type: application/json' \
--data-raw '{
    "remoteAddr": "10.0.0.202",
    "semanticInfos": [
        {
            "id": "e9f8662d40c33c0380894383f74cc2f7"
        }
    ]
}'

```




