---
title: nacos
keywords:
  - APISIX
  - Nacos
  - apisix-seed
description: 本篇文档介绍了如何通过 apisix-seed 在 Apache APISIX 中使用 Nacos 做服务发现。
---

<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->

# 安装
## 部署 Nacos

使用 Nacos Docker 镜像快速部署 Nacos:
```bash
docker run --name nacos-quick -e MODE=standalone -p 8848:8848 -d nacos/nacos-server:1.4.1
```

## 安装 APISIX-Seed

下载并构建 APISIX-Seed:
```bash
git clone https://github.com/api7/apisix-seed.git
cd apisix-seed
make build && make install
```

默认配置文件在 `/usr/local/apisix-seed/conf/conf.yaml` 中，内容如下：
```yaml
etcd:                            # APISIX etcd Configure
  host:
    - "http://127.0.0.1:2379"
  prefix: /apisix
  timeout: 30

discovery:                       # service discovery center
  nacos:
    host:                        # it's possible to define multiple nacos hosts addresses of the same nacos cluster.
      - "http://127.0.0.1:8848"
    prefix: /nacos
    weight: 100                  # default weight for node
    timeout:
      connect: 2000              # default 2000ms
      send: 2000                 # default 2000ms
      read: 5000                 # default 5000ms
```
每个配置项大家可以很容易的理解，我们不再赘述。

启动 APISIX-Seed:
```bash
APISIX_SEED_WORKDIR=/usr/local/apisix-seed /usr/local/apisix-seed/apisix-seed
```

## 注册上游服务

通过 Docker 启动 httpbin 服务:
```bash
docker run -d -p 8080:80 --rm kennethreitz/httpbin
```

将服务注册到 Nacos:
```bash
curl -X POST 'http://127.0.0.1:8848/nacos/v1/ns/instance?serviceName=httpbin&ip=127.0.0.1&port=8080'
```

## 在 Apache APISIX 中验证

使用默认配置启动 Apache APISIX:
```bash
apisix start
```

通过 Apache APISIX 的 Admin API 接口创建路由:
```bash
curl http://127.0.0.1:9180/apisix/admin/routes/1 -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1' -X PUT -i -d '
{
    "uri": "/*",
    "hosts": [
        "httpbin"
    ],
    "upstream": {
        "discovery_type": "nacos",
        "service_name": "httpbin",
        "type": "roundrobin"
    }
}'
```

发送请求确认服务发现是否生效:
```bash
curl http://127.0.0.1:9080/get -H 'Host: httpbin'
```

# 功能介绍
## 支持 metadata

APISIX-SEED 支持根据 metadata 来对服务进行分组，让我们来看一个例子

在 nacos 内部注册 2 个实例:

```
curl -X POST 'http://127.0.0.1:8848/nacos/v1/ns/instance?serviceName=httpbin&ip=127.0.0.1&port=8080&metadata=%7B%22version%22:%22v1%22%7D&ephemeral=false'

curl -X POST 'http://127.0.0.1:8848/nacos/v1/ns/instance?serviceName=httpbin&ip=127.0.0.1&port=8081&metadata=%7B%22version%22:%22v2%22%7D&ephemeral=false'

```

其中实例 127.0.0.1:8080 的 metadata 信息为 {"version":"v1"}，

实例 127.0.0.1:8081 的 metadata 信息为 {"version":"v2"}

在 APISIX 中配置只获取 version = v1 的上游实例

```
curl http://127.0.0.1:9180/apisix/admin/routes/1 -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1' -X PUT -i -d '
{
    "uri": "/*",
    "hosts": [
        "httpbin"
    ],
    "upstream": {
        "discovery_type": "nacos",
        "service_name": "httpbin",
        "type": "roundrobin",
        "discovery_args": {
          "metadata": {
            "version": "v1"
          }
        }
    }
}'
```

查询

```
curl http://127.0.0.1:9180/apisix/admin/routes/1 -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1'

{
  "value": {
    "hosts": [
      "httpbin"
    ],
    "update_time": 1677482525,
    "upstream": {
      "hash_on": "vars",
      "nodes": [
        {
          "weight": 1,
          "port": 8080,
          "host": "127.0.0.1"
        }
      ],
      "discovery_args": {
        "metadata": {
          "version": "v1"
        }
      },
      "_service_name": "httpbin",
      "_discovery_type": "nacos",
      "pass_host": "pass",
      "scheme": "http",
      "type": "roundrobin"
    },
    "id": "1",
    "status": 1,
    "create_time": 1677482525,
    "uri": "/*",
    "priority": 0
  },
  "key": "/apisix/routes/1",
  "createdIndex": 557,
  "modifiedIndex": 559
}
```

可以看到，只获取到了 version = v1 的上游实例 127.0.0.1:8080
