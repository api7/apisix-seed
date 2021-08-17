package comm

import (
	"reflect"
	"testing"

	"github.com/api7/apisix-seed/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestQueryDecode(t *testing.T) {
	tests := []struct {
		caseDesc   string
		giveHeader [][2]string
		giveBody   [][2]string
		wantValues []string
		wantArgs   map[string]string
	}{
		{
			caseDesc: "Test Query Encode without extra arguments",
			giveHeader: [][2]string{
				{"event", "add"},
				{"entity", "upstream;1"},
				{"service", "test"},
			},
			giveBody:   nil,
			wantValues: []string{"add", "upstream;1", "test"},
			wantArgs:   nil,
		},
		{
			caseDesc: "Test Query Encode with arguments",
			giveHeader: [][2]string{
				{"event", "update"},
				{"entity", "service;1"},
				{"service", "test"},
			},
			giveBody: [][2]string{
				{"namespace_id", "test_ns"},
			},
			wantValues: []string{"update", "service;1", "test"},
			wantArgs: map[string]string{
				"namespace_id": "test_ns",
			},
		},
	}

	for _, tc := range tests {
		header := make(utils.Message, 0, len(tc.giveHeader))
		for _, pair := range tc.giveHeader {
			header.Add(pair[0], pair[1])
		}
		body := make(utils.Message, 0, len(tc.giveBody))
		for _, pair := range tc.giveBody {
			body.Add(pair[0], pair[1])
		}
		query := Query{header, body}
		values, args, err := query.Decode()
		assert.Nil(t, err)
		assert.True(t, reflect.DeepEqual(values, tc.wantValues), tc.caseDesc)
		assert.True(t, reflect.DeepEqual(args, tc.wantArgs), tc.caseDesc)
	}
}
