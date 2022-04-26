# APISIX-Seed for Apache APISIX
[![Go Report Card](https://goreportcard.com/badge/github.com/api7/apisix-seed)](https://goreportcard.com/report/github.com/api7/apisix-seed)
[![Build Status](https://github.com/api7/apisix-seed/workflows/unit-test-ci/badge.svg?branch=main)](https://github.com/api7/apisix-seed/actions)
[![Codecov](https://codecov.io/gh/api7/apisix-seed/branch/main/graph/badge.svg)](https://codecov.io/gh/api7/apisix-seed)

Do service discovery for Apache APISIX on the Control Plane

# What's APISIX-Seed
Apache APISIX is a dynamic, real-time, high-performance API gateway.

In terms of architecture design, Apache APISIX is divided into two parts: data plane and control plane. The data plane is Apache APISIX itself, which is the component of the traffic proxy and offers many full-featured plugins covering areas such as authentication, security, traffic control, serverless, analytics & monitoring, transformations and logging.
The control plane is mainly used to manage routing, and implement the configuration center through etcd.

For cloud-native gateways, it is necessary to dynamically obtain the latest service instance information (service discovery) through the service registry. Currently, Apache APISIX already supports this feature in the data plane.

This project is a component of Apache APISIX to implement service discovery in the control plane. It supports cluster deployment. At present, we have supported zookeeper and nacos. We will also support more service registries.

The following figure is the topology diagram of APISIX-Seed deployment.

![apisix-seed overview](docs/assets/images/apisix-seed%20overview.png)

# Why APISIX-Seed
- Network topology becomes simpler

> Apache APISIX does not need to maintain a network connection with each registry, and only needs to pay attention to the configuration information in Etcd. This will greatly simplify the network topology.

- Total data volume about upstream service becomes smaller
> Due to the characteristics of the registry, Apache APISIX may store the full amount of registry service data in the worker, such as consul_kv. By introducing APISIX-Seed, each process of Apache APISIX will not need to additionally cache upstream service-related information

- Easier to manage
> Service discovery configuration needs to be configured once per APISIX instance. By introducing APISIX-Seed, Apache APISIX will be indifferent to the configuration changes of the service registry

# How it works
We use the go language to implement APISIX-Seed. The flow diagram:

![apisix-seed flow diagram](docs/assets/images/apisix-seed%20workflow.png)

APISIX-Seed completes data exchange by watching the changes of etcd and service registry at the same time.

The process is as follows:

- Apache APISIX registers an upstream and specifies the service discovery type to etcd.
- APISIX-Seed watches the resource changes of Apache APISIX in etcd and filters the discovery type and obtains the service name.
- APISIX-Seed binds the service to the etcd resource and starts watching the service in the service registry.
- The client registers the service in the service registry.
- APISIX-Seed gets the service changes in the service registry.
- APISIX-Seed queries the bound etcd resource information through the service name, and writes the updated service node to etcd.
- The Apache APISIX worker watches etcd changes and refreshes the service node information to the memory.
