package message

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/api7/gopkg/pkg/log"
)

// Sample route config
// "labels":
// {
// 	"discovery_args.group_name": "group_name",
// 	"discovery_args.namespace_id": "test_name",
// 	"discovery_type": "nacos",
// 	"service_name": "test-service",
// 	"service_grpc_port": "10001"
// },

type Labels struct {
	DiscoveryType            string `json:"discovery_type,omitempty"`
	ServiceName              string `json:"service_name,omitempty"`
	DiscoveryArgsNamespaceID string `json:"discovery_args.namespace_id,omitempty"`
	DiscoveryArgsGroupName   string `json:"discovery_args.group_name,omitempty"`
	ServiceGrpcPort          string `json:"service_grpc_port,omitempty"`
}

type UpstreamArg struct {
	NamespaceID string                 `json:"namespace_id,omitempty"`
	GroupName   string                 `json:"group_name,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type Upstream struct {
	Nodes            interface{}  `json:"nodes,omitempty"`
	DiscoveryType    string       `json:"discovery_type,omitempty"`
	DupDiscoveryType string       `json:"_discovery_type,omitempty"`
	DiscoveryArgs    *UpstreamArg `json:"discovery_args,omitempty"`
	DupServiceName   string       `json:"_service_name,omitempty"`
	ServiceName      string       `json:"service_name,omitempty"`
}

const (
	A6RoutesConf    = 0
	A6UpstreamsConf = 1
	A6ServicesConf  = 2
)

func ToA6Type(prefix string) int {
	if strings.HasSuffix(prefix, "routes") {
		return A6RoutesConf
	}
	if strings.HasSuffix(prefix, "upstreams") {
		return A6UpstreamsConf
	}
	if strings.HasSuffix(prefix, "services") {
		return A6ServicesConf
	}
	return A6RoutesConf
}

type A6Conf interface {
	GetAll() *map[string]interface{}
	Inject(nodes interface{})
	Marshal() ([]byte, error)
	GetUpstream() Upstream
	HasNodesAttr() bool
}

func NewA6Conf(value []byte, a6Type int) (A6Conf, error) {
	switch a6Type {
	case A6RoutesConf:
		return NewRoutes(value)
	case A6UpstreamsConf:
		return NewUpstreams(value)
	case A6ServicesConf:
		return NewServices(value)
	default:
		return NewRoutes(value)
	}
}

func unmarshal(data []byte, v A6Conf) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, v.GetAll())
	if err != nil {
		return err
	}

	return nil
}

// Embed the latest value into `all` map
func embedElm(v reflect.Value, all map[string]interface{}) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	typ := v.Type()

	fieldNum := typ.NumField()
	for i := 0; i < fieldNum; i++ {
		field := typ.Field(i)
		fieldName := field.Name
		tagName := strings.TrimSuffix(field.Tag.Get("json"), ",omitempty")

		if fieldName == "All" && tagName == "-" {
			continue
		}

		if fieldName == "hasNodesAttr" {
			continue
		}

		val := v.FieldByName(fieldName)
		// ignore members without set values
		if val.IsZero() {
			continue
		}

		if fieldName == "DiscoveryType" || fieldName == "ServiceName" || fieldName == "DiscoveryArgs" {
			name := typ.Name()
			if name == "Upstream" {
				// all["_"+tagName] = val.Interface()
				delete(all, tagName)
				continue
			}
		}

		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		if val.Kind() == reflect.Struct {
			// handle struct embedding
			if field.Anonymous {
				embedElm(val, all)
			} else {
				if _, ok := all[tagName]; !ok {
					all[tagName] = make(map[string]interface{})
				}
				embedElm(val, all[tagName].(map[string]interface{}))
			}
		} else {
			all[tagName] = val.Interface()
		}
	}
}

type Upstreams struct {
	Upstream
	hasNodesAttr bool
	All          map[string]interface{} `json:"-"`
}

func (ups *Upstreams) GetAll() *map[string]interface{} {
	return &ups.All
}

func (ups *Upstreams) Marshal() ([]byte, error) {
	embedElm(reflect.ValueOf(ups), ups.All)

	return json.Marshal(ups.All)
}

func (ups *Upstreams) Inject(nodes interface{}) {
	ups.Nodes = nodes
}

func (ups *Upstreams) GetUpstream() Upstream {
	return ups.Upstream
}

func (ups *Upstreams) HasNodesAttr() bool {
	return ups.hasNodesAttr
}

func NewUpstreams(value []byte) (A6Conf, error) {
	ups := &Upstreams{
		All: make(map[string]interface{}),
	}
	err := unmarshal(value, ups)
	if err != nil {
		return nil, err
	}

	//We have to save the state of the nodes property after serializing the original data,
	// because it may be changed in subsequent logic
	if ups.Nodes != nil {
		ups.hasNodesAttr = true
	}

	return ups, nil
}

// "labels": {
// 	"service_name":"aquaman-user",
// 	"discovery_type":"nacos",
// 	"service_grpc_port":"10001"
// },

type Routes struct {
	Labels       Labels                 `json:"labels"`
	Upstream     Upstream               `json:"upstream"`
	All          map[string]interface{} `json:"-"`
	hasNodesAttr bool
}

func (routes *Routes) GetAll() *map[string]interface{} {
	return &routes.All
}

func (routes *Routes) Marshal() ([]byte, error) {
	// If grpc port is configured, modify all nodes' port to grpc port
	if routes.Labels.ServiceGrpcPort != "" && routes.Upstream.Nodes != nil {
		grpcPort, err := strconv.ParseInt(routes.Labels.ServiceGrpcPort, 10, 64)
		if err != nil {
			log.Errorf("invalid grpc port configuration: failed to parse port %s to integer for route %s: %s", routes.Labels.ServiceGrpcPort, routes.All["id"], err)
			return nil, fmt.Errorf("invalid grpc port configuration: failed to parse port %s to integer for route %s: %s", routes.Labels.ServiceGrpcPort, routes.All["id"], err)
		}
		if nodes, ok := routes.Upstream.Nodes.([]*Node); ok {
			nodesCopy := make([]*Node, len(nodes))
			for i, n := range nodes {
				nodesCopy[i] = &Node{
					Host:   n.Host,
					Weight: n.Weight,
					Port:   int(grpcPort),
				}
				log.Infof("updated gRPC port to %d for node %s in route %s", grpcPort, n.Host, routes.All["id"])
			}
			routes.Upstream.Nodes = nodesCopy
		}
	}

	embedElm(reflect.ValueOf(routes), routes.All)

	// routes.All["labels"] = routes.Labels
	bytes, error := json.Marshal(routes.All)
	// print("a6conf marshal 2=====", string(bytes))
	return bytes, error
}

func (routes *Routes) Inject(nodes interface{}) {
	routes.Upstream.Nodes = nodes
}

func (routes *Routes) GetUpstream() Upstream {
	return routes.Upstream
}

func (routes *Routes) HasNodesAttr() bool {
	return routes.hasNodesAttr
}

func NewRoutes(value []byte) (A6Conf, error) {
	routes := &Routes{
		All: make(map[string]interface{}),
	}

	// println("===", string(value))

	err := unmarshal(value, routes)
	if err != nil {
		return nil, err
	}

	if id, ok := routes.All["id"].(string); ok {
		if routes.Labels.ServiceName != "" {
			println("upstream nodes in route id: ", id, " will be synced with service: ", routes.Labels.ServiceName)
			routes.Upstream.ServiceName = routes.Labels.ServiceName
			routes.Upstream.DiscoveryType = routes.Labels.DiscoveryType
			routes.Upstream.DiscoveryArgs = &UpstreamArg{
				NamespaceID: routes.Labels.DiscoveryArgsNamespaceID,
				GroupName:   routes.Labels.DiscoveryArgsGroupName,
			}
		}
	}

	if routes.Upstream.Nodes != nil {
		routes.hasNodesAttr = true
	}

	return routes, nil
}

type Services struct {
	Upstream     Upstream               `json:"upstream"`
	All          map[string]interface{} `json:"-"`
	hasNodesAttr bool
}

func (services *Services) GetAll() *map[string]interface{} {
	return &services.All
}

func (services *Services) Marshal() ([]byte, error) {
	embedElm(reflect.ValueOf(services), services.All)

	return json.Marshal(services.All)
}

func (services *Services) Inject(nodes interface{}) {
	services.Upstream.Nodes = nodes
}

func (services *Services) GetUpstream() Upstream {
	return services.Upstream
}

func (services *Services) HasNodesAttr() bool {
	return services.hasNodesAttr
}

func NewServices(value []byte) (A6Conf, error) {
	services := &Services{
		All: make(map[string]interface{}),
	}
	err := unmarshal(value, services)
	if err != nil {
		return nil, err
	}

	if services.Upstream.Nodes != nil {
		services.hasNodesAttr = true
	}

	return services, nil
}
