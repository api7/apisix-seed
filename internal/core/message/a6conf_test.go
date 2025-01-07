package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewA6Conf_Routes(t *testing.T) {
	testCases := []struct {
		desc  string
		value string
		err   string
	}{
		{
			desc: "normal",
			value: `{
    "uri": "/hh",
    "upstream": {
        "discovery_type": "nacos",
        "service_name": "APISIX-NACOS",
        "discovery_args": {
            "group_name": "DEFAULT_GROUP",
			"metadata":{
				"version":"v1",
				"is_gray":true
			}
        }
    }
}`,
		},
		{
			desc: "error conf",
			value: `{
    "uri": "/hh"
    "upstream": {
        "discovery_type": "nacos",
        "service_name": "APISIX-NACOS",
        "discovery_args": {
            "group_name": "DEFAULT_GROUP"
        }
    }
}`,
			err: `invalid character '"' after object key:value pair`,
		},
	}

	for _, v := range testCases {
		a6, err := NewA6Conf([]byte(v.value), A6RoutesConf)
		if v.err != "" {
			assert.Equal(t, v.err, err.Error(), v.desc)
		} else {
			assert.Nil(t, err, v.desc)
			assert.Equal(t, "nacos", a6.GetUpstream().DiscoveryType)
			assert.Equal(t, "APISIX-NACOS", a6.GetUpstream().ServiceName)
			assert.Equal(t, "v1", a6.GetUpstream().DiscoveryArgs.Metadata["version"])
			assert.Equal(t, true, a6.GetUpstream().DiscoveryArgs.Metadata["is_gray"])
		}

	}
}

func TestInject_Routes(t *testing.T) {
	givenA6Str := `{
    "uri": "/hh",
    "upstream": {
        "discovery_type": "nacos",
        "service_name": "APISIX-NACOS",
        "discovery_args": {
            "group_name": "DEFAULT_GROUP"
        }
    }
}`
	nodes := []*Node{
		{
			Host:   "192.168.1.1",
			Port:   80,
			Weight: 1,
		},
		{
			Host:   "192.168.1.2",
			Port:   80,
			Weight: 1,
		},
	}
	caseDesc := "sanity"
	a6, err := NewA6Conf([]byte(givenA6Str), A6RoutesConf)
	assert.Nil(t, err, caseDesc)
	a6.Inject(nodes)
	assert.Len(t, a6.GetUpstream().Nodes, 2)
}

func TestMarshal_Routes(t *testing.T) {
	givenA6Str := `{
    "status": 1,
    "id": "3",
    "uri": "/hh",
    "upstream": {
        "scheme": "http",
        "pass_host": "pass",
        "type": "roundrobin",
        "hash_on": "vars",
        "discovery_type": "nacos",
        "service_name": "APISIX-NACOS",
        "discovery_args": {
            "group_name": "DEFAULT_GROUP"
        }
    },
    "create_time": 1648871506,
    "priority": 0,
    "update_time": 1648871506
}`
	nodes := []*Node{
		{Host: "192.168.1.1", Port: 80, Weight: 1},
		{Host: "192.168.1.2", Port: 80, Weight: 1},
	}

	wantA6Str := `{
    "status": 1,
    "id": "3",
    "uri": "/hh",
    "upstream": {
        "scheme": "http",
        "pass_host": "pass",
        "type": "roundrobin",
        "hash_on": "vars",
        "nodes": [
            {
                "host": "192.168.1.1",
                "port": 80,
                "weight": 1
            },
            {
                "host": "192.168.1.2",
                "port": 80,
                "weight": 1
            }
        ]
    },
    "create_time": 1648871506,
    "priority": 0,
    "update_time": 1648871506
}`
	caseDesc := "sanity"
	a6, err := NewA6Conf([]byte(givenA6Str), A6RoutesConf)
	assert.Nil(t, err, caseDesc)

	a6.Inject(&nodes)
	ss, err := a6.Marshal()
	assert.Nil(t, err, caseDesc)

	assert.JSONEq(t, wantA6Str, string(ss))
}

func TestMarshal_Routes_With_Grpc_Port(t *testing.T) {
	givenA6Str := `{
    "status": 1,
    "id": "3",
    "uri": "/hh",
	 "labels": {
        "discovery_type": "nacos",
        "discovery_args.group_name": "DEFAULT_GROUP",
        "service_name": "test-service",
		"service_grpc_port":"10001"
    },
    "upstream": {
        "scheme": "http",
        "pass_host": "pass",
        "type": "roundrobin",
        "hash_on": "vars",
        "nodes": {
            "192.168.1.1:10001": 1
        }
    },
    "create_time": 1648871506,
    "priority": 0,
    "update_time": 1648871506
}`
	nodes := []*Node{
		{Host: "192.168.1.1", Port: 80, Weight: 1},
		{Host: "192.168.1.2", Port: 80, Weight: 1},
	}

	wantA6Str := `{
    "status": 1,
    "id": "3",
    "uri": "/hh",
	"labels": {
        "discovery_type": "nacos",
        "discovery_args.group_name": "DEFAULT_GROUP",
        "service_name": "test-service",
		"service_grpc_port":"10001"
    },
    "upstream": {
        "scheme": "http",
        "pass_host": "pass",
        "type": "roundrobin",
        "hash_on": "vars",
        "nodes": [
            {
                "host": "192.168.1.1",
                "port": 10001,
                "weight": 1
            },
            {
                "host": "192.168.1.2",
                "port": 10001,
                "weight": 1
            }
        ]
    },
    "create_time": 1648871506,
    "priority": 0,
    "update_time": 1648871506
}`
	caseDesc := "sanity"
	a6, err := NewA6Conf([]byte(givenA6Str), A6RoutesConf)
	assert.Nil(t, err, caseDesc)

	a6.Inject(nodes)
	ss, err := a6.Marshal()
	assert.Nil(t, err, caseDesc)

	assert.JSONEq(t, wantA6Str, string(ss))
}

func TestHasNodesAttr_Routes(t *testing.T) {
	tests := []struct {
		name  string
		a6Str string
		want  bool
	}{
		{
			name:  "without upstream",
			a6Str: `{"plugins":{"fault-injection":{"abort":{"http_status":200,"body":"fine"}}},"uri":"/status"}`,
			want:  false,
		},
		{
			name:  "has upstream without nodes",
			a6Str: `{"uri":"/hh","upstream":{"type":"roundrobin","discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			want:  false,
		},
		{
			name:  "normal",
			a6Str: `{"uri":"/hh","upstream":{"type":"roundrobin","nodes":[{"host":"192.168.1.1","port":80,"weight":1}]}}`,
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			routes, err := NewRoutes([]byte(tt.a6Str))
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, routes.HasNodesAttr(), "HasNodesAttr()")
		})
	}
}

func TestNewA6Conf_Services(t *testing.T) {
	testCases := []struct {
		desc  string
		value string
		err   string
	}{
		{
			desc: "normal",
			value: `{
    "enable_websocket": false,
    "upstream": {
        "discovery_type": "nacos",
        "service_name": "APISIX-NACOS",
        "discovery_args": {
            "group_name": "DEFAULT_GROUP"
        }
    }
}`,
		},
		{
			desc: "error conf",
			value: `{
    "enable_websocket": false
    "upstream": {
        "discovery_type": "nacos",
        "service_name": "APISIX-NACOS",
        "discovery_args": {
            "group_name": "DEFAULT_GROUP"
        }
    }
}`,
			err: `invalid character '"' after object key:value pair`,
		},
	}

	for _, v := range testCases {
		a6, err := NewA6Conf([]byte(v.value), A6ServicesConf)
		if v.err != "" {
			assert.Equal(t, v.err, err.Error(), v.desc)
		} else {
			assert.Nil(t, err, v.desc)
			assert.Equal(t, "nacos", a6.GetUpstream().DiscoveryType)
			assert.Equal(t, "APISIX-NACOS", a6.GetUpstream().ServiceName)
		}

	}
}

func TestInject_Services(t *testing.T) {
	givenA6Str := `{
    "enable_websocket": false,
    "upstream": {
        "discovery_type": "nacos",
        "service_name": "APISIX-NACOS",
        "discovery_args": {
            "group_name": "DEFAULT_GROUP"
        }
    }
}`
	nodes := []*Node{
		{
			Host:   "192.168.1.1",
			Port:   80,
			Weight: 1,
		},
		{
			Host:   "192.168.1.2",
			Port:   80,
			Weight: 1,
		},
	}
	caseDesc := "sanity"
	a6, err := NewA6Conf([]byte(givenA6Str), A6ServicesConf)
	assert.Nil(t, err, caseDesc)
	a6.Inject(nodes)
	assert.Len(t, a6.GetUpstream().Nodes, 2)
}

func TestMarshal_Services(t *testing.T) {
	givenA6Str := `{
    "enable_websocket": false,
    "upstream": {
        "scheme": "http",
        "pass_host": "pass",
        "type": "roundrobin",
        "hash_on": "vars",
        "discovery_type": "nacos",
        "service_name": "APISIX-NACOS",
        "discovery_args": {
            "group_name": "DEFAULT_GROUP"
        }
    },
    "create_time": 1648871506,
    "update_time": 1648871506
}`
	nodes := []*Node{
		{Host: "192.168.1.1", Port: 80, Weight: 1},
		{Host: "192.168.1.2", Port: 80, Weight: 1},
	}

	wantA6Str := `{
    "enable_websocket": false,
    "upstream": {
        "scheme": "http",
        "pass_host": "pass",
        "type": "roundrobin",
        "hash_on": "vars",
        "_discovery_type": "nacos",
        "_service_name": "APISIX-NACOS",
        "discovery_args": {
            "group_name": "DEFAULT_GROUP"
        },
        "nodes": [
            {
                "host": "192.168.1.1",
                "port": 80,
                "weight": 1
            },
            {
                "host": "192.168.1.2",
                "port": 80,
                "weight": 1
            }
        ]
    },
    "create_time": 1648871506,
    "update_time": 1648871506
}`
	caseDesc := "sanity"
	a6, err := NewA6Conf([]byte(givenA6Str), A6RoutesConf)
	assert.Nil(t, err, caseDesc)

	a6.Inject(&nodes)
	ss, err := a6.Marshal()
	assert.Nil(t, err, caseDesc)

	assert.JSONEq(t, wantA6Str, string(ss))
}

func TestHasNodesAttr_Services(t *testing.T) {
	tests := []struct {
		name  string
		a6Str string
		want  bool
	}{
		{
			name:  "without upstream",
			a6Str: `{"plugins":{"limit-count":{"count":2,"time_window":60,"rejected_code":503,"key":"remote_addr"}}}`,
			want:  false,
		},
		{
			name:  "has upstream without nodes",
			a6Str: `{"plugins":{"limit-count":{"count":2,"time_window":60,"rejected_code":503,"key":"remote_addr"}},"upstream":{"type":"roundrobin","discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			want:  false,
		},
		{
			name:  "normal",
			a6Str: `{"plugins":{"limit-count":{"count":2,"time_window":60,"rejected_code":503,"key":"remote_addr"}},"upstream":{"type":"roundrobin","nodes":{"127.0.0.1:1980":1}}}`,
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			services, err := NewServices([]byte(tt.a6Str))
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, services.HasNodesAttr(), "HasNodesAttr()")
		})
	}
}

func TestNewA6Conf_Upstreams(t *testing.T) {
	testCases := []struct {
		desc  string
		value string
		err   string
	}{
		{
			desc: "normal",
			value: `{
    "discovery_type": "nacos",
	"service_name": "APISIX-NACOS",
	"discovery_args": {
		"group_name": "DEFAULT_GROUP"
	}
}`,
		},
		{
			desc: "error conf",
			value: `{
    "discovery_type": "nacos"
	"service_name": "APISIX-NACOS",
	"discovery_args": {
		"group_name": "DEFAULT_GROUP"
	}
}`,
			err: `invalid character '"' after object key:value pair`,
		},
	}

	for _, v := range testCases {
		a6, err := NewA6Conf([]byte(v.value), A6UpstreamsConf)
		if v.err != "" {
			assert.Equal(t, v.err, err.Error(), v.desc)
		} else {
			assert.Nil(t, err, v.desc)
			assert.Equal(t, "nacos", a6.GetUpstream().DiscoveryType)
			assert.Equal(t, "APISIX-NACOS", a6.GetUpstream().ServiceName)
		}

	}
}

func TestInject_Upstreams(t *testing.T) {
	givenA6Str := `{
    "discovery_type": "nacos",
	"service_name": "APISIX-NACOS",
	"discovery_args": {
		"group_name": "DEFAULT_GROUP"
	}
}`
	nodes := []*Node{
		{
			Host:   "192.168.1.1",
			Port:   80,
			Weight: 1,
		},
		{
			Host:   "192.168.1.2",
			Port:   80,
			Weight: 1,
		},
	}
	caseDesc := "sanity"
	a6, err := NewA6Conf([]byte(givenA6Str), A6UpstreamsConf)
	assert.Nil(t, err, caseDesc)
	a6.Inject(nodes)
	assert.Len(t, a6.GetUpstream().Nodes, 2)
}

func TestMarshal_Upstreams(t *testing.T) {
	givenA6Str := `{
    "status":1,
    "id":"3",
    "scheme":"http",
    "pass_host":"pass",
    "type":"roundrobin",
    "hash_on":"vars",
    "discovery_type":"nacos",
    "service_name":"APISIX-NACOS",
    "discovery_args":{
        "group_name":"DEFAULT_GROUP",
		"metadata":{
			"version":"v1"
		}
    },
    "create_time":1648871506,
    "update_time":1648871506
}`
	nodes := []*Node{
		{Host: "192.168.1.1", Port: 80, Weight: 1},
		{Host: "192.168.1.2", Port: 80, Weight: 1},
	}

	wantA6Str := `{
    "status": 1,
    "id": "3",
    "scheme": "http",
	"pass_host": "pass",
	"type": "roundrobin",
	"hash_on": "vars",
	"_discovery_type": "nacos",
	"_service_name": "APISIX-NACOS",
	"discovery_args": {
		"group_name": "DEFAULT_GROUP",
		"metadata":{
			"version":"v1"
		}
	},
	"nodes": [
		{
			"host": "192.168.1.1",
			"port": 80,
			"weight": 1
		},
		{
			"host": "192.168.1.2",
			"port": 80,
			"weight": 1
		}
	],
    "create_time": 1648871506,
    "update_time": 1648871506
}`
	caseDesc := "sanity"
	a6, err := NewA6Conf([]byte(givenA6Str), A6UpstreamsConf)
	assert.Nil(t, err, caseDesc)

	a6.Inject(&nodes)
	ss, err := a6.Marshal()
	assert.Nil(t, err, caseDesc)

	assert.JSONEq(t, wantA6Str, string(ss))
}

func TestHasNodesAttr_Upstreams(t *testing.T) {
	tests := []struct {
		name  string
		a6Str string
		want  bool
	}{
		{
			name:  "upstream without nodes",
			a6Str: `{"type":"roundrobin","discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}`,
			want:  false,
		},
		{
			name:  "normal",
			a6Str: `{"type":"roundrobin","nodes":{"127.0.0.1:1980":1}}`,
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ups, err := NewUpstreams([]byte(tt.a6Str))
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, ups.HasNodesAttr(), "HasNodesAttr()")
		})
	}
}
