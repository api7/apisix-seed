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


## 部署 Nacos

使用 Nacos Docker 镜像快速部署 Nacos:
```bash
docker run --name nacos-quick -e MODE=standalone -p 8848:8848 -d nacos/nacos-server:2.0.2
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
curl http://127.0.0.1:9080/apisix/admin/routes/1 -H 'X-API-KEY: edd1c9f034335f136f87ad84b625c8f1' -X PUT -i -d '
{
    "uris": "/*",
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
