package conf

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestNacosValidator(t *testing.T) {
	tests := []struct {
		caseDesc        string
		giveContent     string
		wantValidateErr []error
	}{
		{
			caseDesc:    "Test Required Host: empty content",
			giveContent: ``,
			wantValidateErr: []error{
				fmt.Errorf("Host: Invalid type. Expected: array, given: null"),
			},
		},
		{
			caseDesc: "Test Required Host: empty array",
			giveContent: `
host: []
`,
			wantValidateErr: []error{
				fmt.Errorf("Host: Array must have at least 1 items"),
			},
		},
		{
			caseDesc: "Test pattern match",
			giveContent: `
host:
  - http:#
prefix: /nacos/#1
`,
			wantValidateErr: []error{
				fmt.Errorf("Host.0: Does not match pattern '^http(s)?:\\/\\/[a-zA-Z0-9-_.:\\@]+$'\nPrefix: Does not match pattern '^[\\/a-zA-Z0-9-_.]+$'"),
				fmt.Errorf("Prefix: Does not match pattern '^[\\/a-zA-Z0-9-_.]+$'\nHost.0: Does not match pattern '^http(s)?:\\/\\/[a-zA-Z0-9-_.:\\@]+$'"),
			},
		},
		{
			caseDesc: "Test minimum",
			giveContent: `
host:
  - "http://127.0.0.1:8848"
weight: -1
timeout:
  connect: -1
`,
			wantValidateErr: []error{
				fmt.Errorf("Weight: Must be greater than or equal to 1\nTimeout.Connect: Must be greater than or equal to 1"),
				fmt.Errorf("Timeout.Connect: Must be greater than or equal to 1\nWeight: Must be greater than or equal to 1"),
			},
		},
	}

	for _, tc := range tests {
		_, err := nacosBuilder([]byte(tc.giveContent))
		ret := false
		for _, wantErr := range tc.wantValidateErr {
			if wantErr.Error() == err.Error() {
				ret = true
				break
			}
		}
		assert.True(t, ret, tc.caseDesc)
	}
}

func TestNacosBuilder(t *testing.T) {
	tests := []struct {
		caseDesc    string
		giveContent string
		wantNacos   *Nacos
	}{
		{
			caseDesc: "Test Builder: default value",
			giveContent: `
host:
  - "http://127.0.0.1:8848"
`,
			wantNacos: &Nacos{
				Host:   []string{"http://127.0.0.1:8848"},
				Prefix: "/nacos/v1/",
				Weight: 100,
				Timeout: timeout{
					Connect: 2000,
					Send:    2000,
					Read:    5000,
				},
			},
		},
		{
			caseDesc: "Test Builder: set value",
			giveContent: `
host:
  - "http://127.0.0.1:8858"
prefix: /nacos/v2/ 
weight: 10 
timeout: 
  connect: 200
  send: 200
  read: 500
`,
			wantNacos: &Nacos{
				Host:   []string{"http://127.0.0.1:8858"},
				Prefix: "/nacos/v2/",
				Weight: 10,
				Timeout: timeout{
					Connect: 200,
					Send:    200,
					Read:    500,
				},
			},
		},
	}

	for _, tc := range tests {
		nacos, err := nacosBuilder([]byte(tc.giveContent))
		assert.Nil(t, err)
		ret := reflect.DeepEqual(nacos, tc.wantNacos)
		assert.True(t, ret, tc.caseDesc)
	}
}
