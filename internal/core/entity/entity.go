package entity

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

type Aller interface {
	GetAll() *map[string]interface{}
}

func Unmarshal(data []byte, v Aller) error {
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

func Marshal(v Aller) ([]byte, error) {
	all := *v.GetAll()
	embedElm(reflect.ValueOf(v.(interface{})), all)

	return json.Marshal(all)
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

type BaseInfo struct {
	ID         string `json:"id"`
	CreateTime int64  `json:"create_time,omitempty"`
	UpdateTime int64  `json:"update_time,omitempty"`
}

func (info *BaseInfo) GetBaseInfo() *BaseInfo {
	return info
}

func (info *BaseInfo) Updating(storedInfo *BaseInfo) {
	info.ID = storedInfo.ID
	info.CreateTime = storedInfo.CreateTime
	info.UpdateTime = time.Now().Unix()
}

type BaseInfoSetter interface {
	GetBaseInfo() *BaseInfo
}

type Node struct {
	Host     string      `json:"host,omitempty"`
	Port     int         `json:"port,omitempty"`
	Weight   int         `json:"weight"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type NodesSetter interface {
	SetNodes([]*Node)
}

type UpstreamArg struct {
	NamespaceID string `json:"namespace_id,omitempty"`
	GroupName   string `json:"group_name,omitempty"`
}

type UpstreamDef struct {
	Nodes            interface{}  `json:"nodes,omitempty"`
	DiscoveryType    string       `json:"discovery_type,omitempty"`
	DupDiscoveryType string       `json:"_discovery_type,omitempty"`
	DiscoveryArgs    *UpstreamArg `json:"discovery_args,omitempty"`
	DupServiceName   string       `json:"_service_name,omitempty"`
	ServiceName      string       `json:"service_name,omitempty"`
}

func (u *UpstreamDef) SetNodes(nodes []*Node) {
	u.Nodes = NodesFormat(nodes)
}

func (u *UpstreamDef) GetArgs() map[string]string {
	if u.DiscoveryArgs == nil {
		return nil
	}

	args := make(map[string]string, 2)
	args["namespace_id"] = u.DiscoveryArgs.NamespaceID
	args["group_name"] = u.DiscoveryArgs.GroupName

	return args
}

func (u *UpstreamDef) GetServiceName() string {
	if u.ServiceName != "" {
		return u.ServiceName
	}
	return u.DupServiceName
}

func (u *UpstreamDef) GetDiscoveryType() string {
	if u.DiscoveryType != "" {
		return u.DiscoveryType
	}
	return u.DupDiscoveryType
}

type Entity interface {
	Extract() (string, string, map[string]string)
	GetDiscoveryType() string
	KeyPath(basePath string) string
	Type() string
}

type Upstream struct {
	BaseInfo
	UpstreamDef
	All map[string]interface{} `json:"-"`
}

func (u *Upstream) GetAll() *map[string]interface{} {
	return &u.All
}

func (u *Upstream) SetNodes(nodes []*Node) {
	(u.UpstreamDef).SetNodes(nodes)
}

func (u *Upstream) Extract() (string, string, map[string]string) {
	id, service := u.ID, u.GetServiceName()
	args := (u.UpstreamDef).GetArgs()

	return id, service, args
}

func (u *Upstream) GetDiscoveryType() string {
	return (u.UpstreamDef).GetDiscoveryType()
}

func (u *Upstream) KeyPath(basePath string) string {
	return basePath + "/" + u.ID
}

func (u *Upstream) Type() string {
	return "upstreams"
}

type Route struct {
	BaseInfo
	Upstream *UpstreamDef           `json:"upstream,omitempty"`
	All      map[string]interface{} `json:"-"`
}

func (r *Route) GetAll() *map[string]interface{} {
	return &r.All
}

func (r *Route) SetNodes(nodes []*Node) {
	r.Upstream.SetNodes(nodes)
}

func (r *Route) Extract() (string, string, map[string]string) {
	id, service := r.ID, r.Upstream.GetServiceName()
	args := r.Upstream.GetArgs()

	return id, service, args
}

func (r *Route) GetDiscoveryType() string {
	return r.Upstream.GetDiscoveryType()
}

func (r *Route) KeyPath(basePath string) string {
	return basePath + "/" + r.ID
}

func (r *Route) Type() string {
	return "routes"
}

type Service struct {
	BaseInfo
	Upstream *UpstreamDef           `json:"upstream,omitempty"`
	All      map[string]interface{} `json:"-"`
}

func (s *Service) GetAll() *map[string]interface{} {
	return &s.All
}

func (s *Service) SetNodes(nodes []*Node) {
	s.Upstream.SetNodes(nodes)
}

func (s *Service) Extract() (string, string, map[string]string) {
	id, service := s.ID, s.Upstream.GetServiceName()
	args := s.Upstream.GetArgs()

	return id, service, args
}

func (s *Service) GetDiscoveryType() string {
	return s.Upstream.GetDiscoveryType()
}

func (s *Service) KeyPath(basePath string) string {
	return basePath + "/" + s.ID
}

func (*Service) Type() string {
	return "services"
}
