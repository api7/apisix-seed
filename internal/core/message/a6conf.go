package message

import (
	"encoding/json"
	"reflect"
	"strings"
)

type UpstreamArg struct {
	NamespaceID string `json:"namespace_id,omitempty"`
	GroupName   string `json:"group_name,omitempty"`
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

		val := v.FieldByName(fieldName)
		// ignore members without set values
		if val.IsZero() {
			continue
		}

		if fieldName == "DiscoveryType" || fieldName == "ServiceName" {
			all["_"+tagName] = val.Interface()
			delete(all, tagName)
			continue
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
	All map[string]interface{} `json:"-"`
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

func NewUpstreams(value []byte) (A6Conf, error) {
	ups := &Upstreams{
		All: make(map[string]interface{}),
	}
	err := unmarshal(value, ups)
	if err != nil {
		return nil, err
	}
	return ups, nil
}

type Routes struct {
	Upstream Upstream               `json:"upstream"`
	All      map[string]interface{} `json:"-"`
}

func (routes *Routes) GetAll() *map[string]interface{} {
	return &routes.All
}

func (routes *Routes) Marshal() ([]byte, error) {
	embedElm(reflect.ValueOf(routes), routes.All)

	return json.Marshal(routes.All)
}

func (routes *Routes) Inject(nodes interface{}) {
	routes.Upstream.Nodes = nodes
}

func (routes *Routes) GetUpstream() Upstream {
	return routes.Upstream
}

func NewRoutes(value []byte) (A6Conf, error) {
	routes := &Routes{
		All: make(map[string]interface{}),
	}
	err := unmarshal(value, routes)
	if err != nil {
		return nil, err
	}
	return routes, nil
}

type Services struct {
	Upstream Upstream               `json:"upstream"`
	All      map[string]interface{} `json:"-"`
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

func NewServices(value []byte) (A6Conf, error) {
	services := &Services{
		All: make(map[string]interface{}),
	}
	err := unmarshal(value, services)
	if err != nil {
		return nil, err
	}
	return services, nil
}
