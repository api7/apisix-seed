---
title: nacos
keywords:
  - APISIX
  - Nacos
  - apisix-seed
description: This document contains information about how to use Nacos as service registry in Apache APISIX via apisix-seed.
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


## Deploy Nacos

Quickly deploy Nacos using the Nacos Docker image:
```bash
docker run --name nacos-quick -e MODE=standalone -p 8848:8848 -d nacos/nacos-server:2.0.2
```

## Install APISIX-Seed

Download and build APISIX-Seed:
```bash
git clone https://github.com/api7/apisix-seed.git
cd apisix-seed
make build && make install
```

The default configuration file is in `/usr/local/apisix-seed/conf/conf.yaml` with the following contents:
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
You can easily understand each configuration item, we will not explain it additionally.

Start APISIX-Seed:
```bash
APISIX_SEED_WORKDIR=/usr/local/apisix-seed /usr/local/apisix-seed/apisix-seed
```

## Register the upstream service

Start the httpbin service via Docker:
```bash
docker run -d -p 8080:80 --rm kennethreitz/httpbin
```

Register the service to Nacos:
```bash
curl -X POST 'http://127.0.0.1:8848/nacos/v1/ns/instance?serviceName=httpbin&ip=127.0.0.1&port=8080'
```

## Verify in Apache APISIX

Start Apache APISIX with default configuration:
```bash
apisix start
```

Create a Route through the Admin API interface of Apache APISIX:
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

Send a request to confirm whether service discovery is in effect:
```bash
curl http://127.0.0.1:9080/get -H 'Host: httpbin'
```
