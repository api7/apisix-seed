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
            "group_name": "DEFAULT_GROUP"
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

func TestMarshal(t *testing.T) {
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

func TestMarshal_Upstreams(t *testing.T) {
	givenA6Str := `{
    "status": 1,
    "id": "3",
    "scheme": "http",
	"pass_host": "pass",
	"type": "roundrobin",
	"hash_on": "vars",
	"discovery_type": "nacos",
	"service_name": "APISIX-NACOS",
	"discovery_args": {
		"group_name": "DEFAULT_GROUP"
	},
    "create_time": 1648871506,
    "update_time": 1648871506
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
