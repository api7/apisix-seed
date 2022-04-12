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

type A6Conf struct {
	Upstream Upstream               `json:"upstream"`
	All      map[string]interface{} `json:"-"`
}

func NewA6Conf(value []byte) (*A6Conf, error) {
	a6 := &A6Conf{
		All: make(map[string]interface{}),
	}
	err := unmarshal(value, a6)
	if err != nil {
		return nil, err
	}
	return a6, nil
}

func unmarshal(data []byte, v *A6Conf) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &v.All)
	if err != nil {
		return err
	}

	return nil
}

func (a6 *A6Conf) Inject(nodes interface{}) {
	a6.Upstream.Nodes = nodes
}

func (a6 *A6Conf) Marshal() ([]byte, error) {
	a6.embedElm(reflect.ValueOf(a6), a6.All)

	return json.Marshal(a6.All)
}

// Embed the latest value into `all` map
func (a6 *A6Conf) embedElm(v reflect.Value, all map[string]interface{}) {
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
				a6.embedElm(val, all)
			} else {
				if _, ok := all[tagName]; !ok {
					all[tagName] = make(map[string]interface{})
				}
				a6.embedElm(val, all[tagName].(map[string]interface{}))
			}
		} else {
			all[tagName] = val.Interface()
		}
	}
}
