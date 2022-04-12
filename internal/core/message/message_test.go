package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage(t *testing.T) {
	givenA6Str := `{
    "status": 1,
    "id": "3",
    "uri": "/hh",
    "upstream": {
        "scheme": "http",
        "pass_host": "pass",
        "type": "roundrobin",
        "hash_on": "vars",
        "_discovery_type": "nacos",
        "service_name": "APISIX-NACOS",
        "discovery_args": {
            "group_name": "DEFAULT_GROUP"
        }
    },
    "create_time": 1648871506,
    "priority": 0,
    "update_time": 1648871506
}`
	givenKey := "/apisix/routes/1"
	givenAction := EventAdd
	caseDesc := "normal"

	msg, err := NewMessage(givenKey, []byte(givenA6Str), givenAction)
	assert.Nil(t, err, caseDesc)

	assert.Equal(t, givenKey, msg.Key, caseDesc)
	assert.Equal(t, givenAction, msg.Action, caseDesc)
	assert.Equal(t, "nacos", msg.DiscoveryType(), caseDesc)
	assert.Equal(t, "APISIX-NACOS", msg.ServiceName(), caseDesc)
	assert.Equal(t, "DEFAULT_GROUP", msg.DiscoveryArgs()["group_name"], caseDesc)

	msg.InjectNodes([]*Node{
		{Host: "1.1.31.1", Port: 80, Weight: 1},
	})

	_, err = msg.Marshal()
	assert.Nil(t, err, caseDesc)
}

func TestServiceFilter(t *testing.T) {
	testCases := []struct {
		desc  string
		key   string
		value string
		ret   bool
	}{
		{
			desc:  "normal",
			key:   "/apisix/routes/a",
			value: `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			ret:   true,
		},
		{
			desc:  "no service_name",
			key:   "/apisix/routes/b",
			value: `{"uri":"/hh","upstream":{"discovery_type":"nacos","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			ret:   false,
		},
	}
	for _, tc := range testCases {
		msg, err := NewMessage(tc.key, []byte(tc.value), EventAdd)
		assert.Nil(t, err, tc.desc)
		assert.Equal(t, tc.ret, ServiceFilter(msg))
	}
}

func TestServiceReplace(t *testing.T) {
	testCases := []struct {
		desc     string
		key      string
		value    string
		newValue string
		ret      bool
	}{
		{
			desc:     "a6 conf no change",
			key:      "/apisix/routes/a",
			value:    `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			newValue: `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			ret:      false,
		},
		{
			desc:     "service_name changed",
			key:      "/apisix/routes/b",
			value:    `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			newValue: `{"uri":"/hh","upstream":{"discovery_type":"zk","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			ret:      false,
		},
		{
			desc:     "args changed",
			key:      "/apisix/routes/b",
			value:    `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			newValue: `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"NEW-DEFAULT_GROUP"}}}`,
			ret:      true,
		},
	}
	for _, tc := range testCases {
		msg, err := NewMessage(tc.key, []byte(tc.value), EventAdd)
		assert.Nil(t, err, tc.desc)
		newMsg, err := NewMessage(tc.key, []byte(tc.newValue), EventAdd)
		assert.Nil(t, err, tc.desc)
		assert.Equal(t, tc.ret, ServiceUpdate(msg, newMsg), tc.desc)
	}
}

func TestServiceUpdate(t *testing.T) {
	testCases := []struct {
		desc     string
		key      string
		value    string
		newValue string
		ret      bool
	}{
		{
			desc:     "a6 conf no change",
			key:      "/apisix/routes/a",
			value:    `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			newValue: `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			ret:      false,
		},
		{
			desc:     "service_name changed",
			key:      "/apisix/routes/b",
			value:    `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			newValue: `{"uri":"/hh","upstream":{"discovery_type":"zk","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			ret:      true,
		},
		{
			desc:     "args changed",
			key:      "/apisix/routes/b",
			value:    `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"DEFAULT_GROUP"}}}`,
			newValue: `{"uri":"/hh","upstream":{"discovery_type":"nacos","service_name":"APISIX-NACOS","discovery_args":{"group_name":"NEW-DEFAULT_GROUP"}}}`,
			ret:      false,
		},
	}
	for _, tc := range testCases {
		msg, err := NewMessage(tc.key, []byte(tc.value), EventAdd)
		assert.Nil(t, err, tc.desc)
		newMsg, err := NewMessage(tc.key, []byte(tc.newValue), EventAdd)
		assert.Nil(t, err, tc.desc)
		assert.Equal(t, tc.ret, ServiceReplace(msg, newMsg), tc.desc)
	}
}
