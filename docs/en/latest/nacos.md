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

# Installation

## Deploy Nacos

Quickly deploy Nacos using the Nacos Docker image:
```bash
docker run --name nacos-quick -e MODE=standalone -p 8848:8848 -d nacos/nacos-server:1.4.1
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

# Features

## Metadata

APISIX-SEED supports the grouping of services according to metadata, let's look at an example

Register 2 instances within nacos:

```
curl -X POST 'http://127.0.0.1:8848/nacos/v1/ns/instance?serviceName=httpbin&ip=127.0.0.1&port=8080&metadata=%7B%22version%22:%22v1%22%7D&ephemeral=false'

curl -X POST 'http://127.0.0.1:8848/nacos/v1/ns/instance?serviceName=httpbin&ip=127.0.0.1&port=8081&metadata=%7B%22version%22:%22v2%22%7D&ephemeral=false'

```

Where the metadata information for instance 127.0.0.1:8080 is {"version": "v1"}

And the metadata information for instance 127.0.0.1:8081 is {"version": "v2"}

Configure APISIX to only fetch upstream instances with version = v1

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

Check the upstream

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

As you can see, only the upstream instance 127.0.0.1:8080 with version = v1 is fetched
