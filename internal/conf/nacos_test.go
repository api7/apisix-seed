package conf

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ReadFile(t *testing.T, file string) []byte {
	wd, _ := os.Getwd()
	dir := wd[:strings.Index(wd, "internal")]
	path := filepath.Join(dir, "test/testdata/nacos_conf/", file)
	bs, err := ioutil.ReadFile(path)
	assert.Nil(t, err)
	return bs
}

func TestNacosValidator(t *testing.T) {
	tests := []struct {
		caseDesc        string
		givePath        string
		wantValidateErr []error
	}{
		{
			caseDesc: "Test Required Host: empty content",
			givePath: "empty.yaml",
			wantValidateErr: []error{
				fmt.Errorf("Host: Invalid type. Expected: array, given: null"),
			},
		},
		{
			caseDesc: "Test Required Host: empty array",
			givePath: "empty_host.yaml",
			wantValidateErr: []error{
				fmt.Errorf("Host: Array must have at least 1 items"),
			},
		},
		{
			caseDesc: "Test pattern match",
			givePath: "pattern.yaml",
			wantValidateErr: []error{
				fmt.Errorf("Host.0: Does not match pattern '^http(s)?:\\/\\/[a-zA-Z0-9-_.:]+$'\nPrefix: Does not match pattern '^[\\/a-zA-Z0-9-_.]*$'"),
				fmt.Errorf("Prefix: Does not match pattern '^[\\/a-zA-Z0-9-_.]*$'\nHost.0: Does not match pattern '^http(s)?:\\/\\/[a-zA-Z0-9-_.:]+$'"),
			},
		},
		{
			caseDesc: "Test minimum",
			givePath: "minimum.yaml",
			wantValidateErr: []error{
				fmt.Errorf("Weight: Must be greater than or equal to 1\nTimeout.Connect: Must be greater than or equal to 1"),
				fmt.Errorf("Timeout.Connect: Must be greater than or equal to 1\nWeight: Must be greater than or equal to 1"),
			},
		},
	}

	for _, tc := range tests {
		bc := ReadFile(t, tc.givePath)
		_, err := nacosBuilder(bc)
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
		caseDesc  string
		givePath  string
		wantNacos *Nacos
	}{
		{
			caseDesc: "Test Builder: default value",
			givePath: "default_value.yaml",
			wantNacos: &Nacos{
				Host:      []string{"http://127.0.0.1:8848"},
				Namespace: "public",
				Weight:    100,
				Timeout: timeout{
					Connect: 2000,
					Send:    2000,
					Read:    5000,
				},
			},
		},
		{
			caseDesc: "Test Builder: set value",
			givePath: "set_value.yaml",
			wantNacos: &Nacos{
				Host:      []string{"http://127.0.0.1:8858"},
				Prefix:    "/nacos/v2/",
				Namespace: "pub",
				Username:  "apisix",
				Password:  "apisix-seed",
				Weight:    10,
				Timeout: timeout{
					Connect: 200,
					Send:    200,
					Read:    500,
				},
			},
		},
	}

	for _, tc := range tests {
		bc := ReadFile(t, tc.givePath)
		nacos, err := nacosBuilder(bc)
		assert.Nil(t, err)
		ret := reflect.DeepEqual(nacos, tc.wantNacos)
		assert.True(t, ret, tc.caseDesc)
	}
}
