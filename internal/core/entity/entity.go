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

func (info *BaseInfo) KeyCompat(key string) {
	if info.ID == "" && key != "" {
		info.ID = key
	}
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
	Nodes         interface{}  `json:"nodes,omitempty"`
	DiscoveryType string       `json:"discovery_type,omitempty"`
	DiscoveryArgs *UpstreamArg `json:"discovery_args,omitempty"`
	ServiceName   string       `json:"service_name,omitempty"`
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

type Queer interface {
	Extract() (string, string, map[string]string)
	GetType() string
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
	id, service := u.ID, u.ServiceName
	args := (u.UpstreamDef).GetArgs()

	return id, service, args
}

func (u *Upstream) GetType() string {
	return u.DiscoveryType
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
	id, service := r.ID, r.Upstream.ServiceName
	args := r.Upstream.GetArgs()

	return id, service, args
}

func (r *Route) GetType() string {
	return r.Upstream.DiscoveryType
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
	id, service := s.ID, s.Upstream.ServiceName
	args := s.Upstream.GetArgs()

	return id, service, args
}

func (s *Service) GetType() string {
	return s.Upstream.DiscoveryType
}
