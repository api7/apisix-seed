package entity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodesFormat(t *testing.T) {
	// route data saved in ETCD
	routeStr := `{
		"uris": ["/*"],
		"upstream": {
			"type": "roundrobin",
			"nodes": [{
				"host": "127.0.0.1",
				"port": 80,
				"weight": 0
			}]
		}
	}`

	// bind struct
	var route Route
	err := Unmarshal([]byte(routeStr), &route)
	assert.Nil(t, err)

	// nodes format
	nodes := NodesFormat(route.Upstream.Nodes)

	// json encode for client
	res, err := json.Marshal(nodes)
	assert.Nil(t, err)
	jsonStr := string(res)
	assert.Contains(t, jsonStr, `"weight":0`)
	assert.Contains(t, jsonStr, `"port":80`)
	assert.Contains(t, jsonStr, `"host":"127.0.0.1"`)
}

func TestNodesFormatStruct(t *testing.T) {
	// route data saved in ETCD
	var route Route
	route.Upstream = &UpstreamDef{}
	var nodes = []*Node{{Host: "127.0.0.1", Port: 80, Weight: 0}}
	route.Upstream.Nodes = nodes

	// nodes format
	formattedNodes := NodesFormat(route.Upstream.Nodes)

	// json encode for client
	res, err := json.Marshal(formattedNodes)
	assert.Nil(t, err)
	jsonStr := string(res)
	assert.Contains(t, jsonStr, `"weight":0`)
	assert.Contains(t, jsonStr, `"port":80`)
	assert.Contains(t, jsonStr, `"host":"127.0.0.1"`)
}

func TestNodesFormatMap(t *testing.T) {
	// route data saved in ETCD
	routeStr := `{
		"uris": ["/*"],
		"upstream": {
			"type": "roundrobin",
			"nodes": {"127.0.0.1:8080": 0}
		}
	}`

	// bind struct
	var route Route
	err := Unmarshal([]byte(routeStr), &route)
	assert.Nil(t, err)

	// nodes format
	nodes := NodesFormat(route.Upstream.Nodes)

	// json encode for client
	res, err := json.Marshal(nodes)
	assert.Nil(t, err)
	jsonStr := string(res)
	assert.Contains(t, jsonStr, `"weight":0`)
	assert.Contains(t, jsonStr, `"port":8080`)
	assert.Contains(t, jsonStr, `"host":"127.0.0.1"`)
}

func TestNodesFormatEmptyStruct(t *testing.T) {
	// route data saved in ETCD
	routeStr := `{
		"uris": ["/*"],
		"upstream": {
			"type": "roundrobin",
			"nodes": []
		}
	}`

	// bind struct
	var route Route
	err := Unmarshal([]byte(routeStr), &route)
	assert.Nil(t, err)

	// nodes format
	nodes := NodesFormat(route.Upstream.Nodes)

	// json encode for client
	res, err := json.Marshal(nodes)
	assert.Nil(t, err)
	jsonStr := string(res)
	assert.Contains(t, jsonStr, `[]`)
}

func TestNodesFormatEmptyMap(t *testing.T) {
	// route data saved in ETCD
	routeStr := `{
		"uris": ["/*"],
		"upstream": {
			"type": "roundrobin",
			"nodes": {}
		}
	}`

	// bind struct
	var route Route
	err := Unmarshal([]byte(routeStr), &route)
	assert.Nil(t, err)

	// nodes format
	nodes := NodesFormat(route.Upstream.Nodes)

	// json encode for client
	res, err := json.Marshal(nodes)
	assert.Nil(t, err)
	jsonStr := string(res)
	assert.Contains(t, jsonStr, `[]`)
}

func TestNodesFormatNoNodes(t *testing.T) {
	// route data saved in ETCD
	routeStr := `{
		"uris": ["/*"],
		"upstream": {
			"type": "roundrobin",
			"service_name": "USER-SERVICE",
			"discovery_type": "eureka"
		}
	}`

	// bind struct
	var route Route
	err := Unmarshal([]byte(routeStr), &route)
	assert.Nil(t, err)

	// nodes format
	nodes := NodesFormat(route.Upstream.Nodes)

	// json encode for client
	res, err := json.Marshal(nodes)
	assert.Nil(t, err)
	jsonStr := string(res)
	assert.Contains(t, jsonStr, `null`)
}
