package entity

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpstream(t *testing.T) {
	upstreamStr := `{"type":"roundrobin"}`

	var upstream Upstream
	err := Unmarshal([]byte(upstreamStr), &upstream)
	assert.Nil(t, err)

	jsonStr, err := Marshal(&upstream)
	assert.Nil(t, err)
	assert.Equal(t, upstreamStr, string(jsonStr), "Test unnamed attribute")

	upstream.ServiceName = "TEST-SERVICE"
	upstream.DiscoveryType = "nacos"

	embedElm(reflect.ValueOf(&upstream), upstream.All)
	expectedAll := map[string]interface{}{
		"discovery_type": "nacos",
		"service_name":   "TEST-SERVICE",
		"type":           "roundrobin",
	}
	assert.True(t, reflect.DeepEqual(upstream.All, expectedAll), "Test latest attribute")

	upstream.DiscoveryArgs = &UpstreamArg{}
	upstream.DiscoveryArgs.NamespaceID = "TEST-NAMESPACE"
	upstream.DiscoveryArgs.GroupName = "TEST-GROUP"

	embedElm(reflect.ValueOf(&upstream), upstream.All)
	expectedArg := map[string]interface{}{
		"namespace_id": "TEST-NAMESPACE",
		"group_name":   "TEST-GROUP",
	}
	assert.True(t, reflect.DeepEqual(upstream.All["discovery_args"], expectedArg), "Test latest attribute")
}

func TestRoute(t *testing.T) {
	routeStr := `{"uris":["/*"]}`

	var route Route
	err := Unmarshal([]byte(routeStr), &route)
	assert.Nil(t, err)

	jsonStr, err := Marshal(&route)
	assert.Nil(t, err)
	assert.Equal(t, routeStr, string(jsonStr), "Test unnamed attribute")

	route.Upstream = &UpstreamDef{}
	route.Upstream.ServiceName = "TEST-SERVICE"
	route.Upstream.DiscoveryType = "nacos"

	nodes := []*Node{
		{Host: "test.com"},
	}
	route.SetNodes(nodes)

	embedElm(reflect.ValueOf(&route), route.All)
	expectedAll := map[string]interface{}{
		"uris": []interface{}{"/*"},
		"upstream": map[string]interface{}{
			"discovery_type": "nacos",
			"service_name":   "TEST-SERVICE",
			"nodes":          nodes,
		},
	}
	assert.True(t, reflect.DeepEqual(route.All, expectedAll), "Test latest attribute")
}

func TestService(t *testing.T) {
	serviceStr := `{"name":"TEST"}`

	var service Service
	err := Unmarshal([]byte(serviceStr), &service)
	assert.Nil(t, err)

	jsonStr, err := Marshal(&service)
	assert.Nil(t, err)
	assert.Equal(t, serviceStr, string(jsonStr), "Test unnamed attribute")

	service.Upstream = &UpstreamDef{}
	service.Upstream.ServiceName = "TEST-SERVICE"
	service.Upstream.DiscoveryType = "nacos"

	nodes := []*Node{
		{Host: "test.com"},
	}
	service.SetNodes(nodes)

	embedElm(reflect.ValueOf(&service), service.All)
	expectedAll := map[string]interface{}{
		"name": "TEST",
		"upstream": map[string]interface{}{
			"discovery_type": "nacos",
			"service_name":   "TEST-SERVICE",
			"nodes":          nodes,
		},
	}
	assert.True(t, reflect.DeepEqual(service.All, expectedAll), "Test latest attribute")
}
