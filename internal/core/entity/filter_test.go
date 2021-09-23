package entity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceFilter(t *testing.T) {
	tests := []struct {
		caseDesc   string
		giveRoute  string
		wantResult bool
	}{
		{
			caseDesc: "Test Filter: existing discovery",
			giveRoute: `{
	"uris": ["/*"],	
	"upstream": {	
		"type": "roundrobin",	
		"nodes": [],
		"service_name": "test",
		"discovery_type": "nacos"
	}
}`,
			wantResult: true,
		},
		{
			caseDesc: "Test Filter: non-existing discovery",
			giveRoute: `{
	"uris": ["/*"],	
	"upstream": {	
		"type": "roundrobin",	
		"nodes": {"127.0.0.1:8080": 0}
	}
}`,
			wantResult: false,
		},
	}

	for _, tc := range tests {
		var route Route
		err := json.Unmarshal([]byte(tc.giveRoute), &route)
		assert.Nil(t, err)

		assert.Equal(t, tc.wantResult, ServiceFilter(&route), tc.caseDesc)
	}
}

func TestServiceUpdate(t *testing.T) {
	tests := []struct {
		caseDesc     string
		giveRoute    string
		giveNewRoute string
		wantResult   bool
	}{
		{
			caseDesc: "Test Update: no update",
			giveRoute: `{
	"upstream": {	
		"service_name": "test",
		"discovery_type": "nacos"
	}
}`,
			giveNewRoute: `{
	"upstream": {	
		"service_name": "test",
		"discovery_type": "nacos"
	}
}`,
			wantResult: false,
		},
		{
			caseDesc: "Test Update: service name update",
			giveRoute: `{
	"upstream": {	
		"service_name": "test",
		"discovery_type": "nacos"
	}
}`,
			giveNewRoute: `{
	"upstream": {	
		"service_name": "test1",
		"discovery_type": "nacos"
	}
}`,
			wantResult: false,
		},
		{
			caseDesc: "Test Update: args update",
			giveRoute: `{
	"upstream": {	
		"service_name": "test",
		"discovery_type": "nacos"
	}
}`,
			giveNewRoute: `{
	"upstream": {	
		"service_name": "test",
		"discovery_type": "nacos",
		"discovery_args": {
			"group_name": "test_group"
		}
	}
}`,
			wantResult: true,
		},
	}

	for _, tc := range tests {
		var route Route
		err := json.Unmarshal([]byte(tc.giveRoute), &route)
		assert.Nil(t, err)

		var newRoute Route
		err = json.Unmarshal([]byte(tc.giveNewRoute), &newRoute)
		assert.Nil(t, err)

		assert.Equal(t, tc.wantResult, ServiceUpdate(&route, &newRoute), tc.caseDesc)
	}
}

func TestServiceReplace(t *testing.T) {
	tests := []struct {
		caseDesc     string
		giveRoute    string
		giveNewRoute string
		wantResult   bool
	}{
		{
			caseDesc: "Test Replace: no replace",
			giveRoute: `{
	"upstream": {	
		"service_name": "test",
		"discovery_type": "nacos"
	}
}`,
			giveNewRoute: `{
	"upstream": {	
		"service_name": "test",
		"discovery_type": "nacos"
	}
}`,
			wantResult: false,
		},

		{
			caseDesc: "Test Replace: service name update",
			giveRoute: `{
	"upstream": {	
		"service_name": "test",
		"discovery_type": "nacos"
	}
}`,
			giveNewRoute: `{
	"upstream": {	
		"service_name": "test1",
		"discovery_type": "nacos"
	}
}`,
			wantResult: true,
		},
		{
			caseDesc: "Test Replace: discovery type update",
			giveRoute: `{
	"upstream": {	
		"service_name": "test",
		"discovery_type": "nacos"
	}
}`,
			giveNewRoute: `{
	"upstream": {	
		"service_name": "test",
		"discovery_type": "consul"
	}
}`,
			wantResult: true,
		},
	}

	for _, tc := range tests {
		var route Route
		err := json.Unmarshal([]byte(tc.giveRoute), &route)
		assert.Nil(t, err)

		var newRoute Route
		err = json.Unmarshal([]byte(tc.giveNewRoute), &newRoute)
		assert.Nil(t, err)

		assert.Equal(t, tc.wantResult, ServiceReplace(&route, &newRoute), tc.caseDesc)
	}
}
