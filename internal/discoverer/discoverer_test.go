package discoverer

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/api7/apisix-seed/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestQueryCheck(t *testing.T) {
	tests := []struct {
		caseDesc string
		giveMsg  [][2]string
		wantErr  error
	}{
		{
			caseDesc: "Test Wrong Format: empty message",
			giveMsg:  [][2]string{},
			wantErr:  fmt.Errorf("incorrect query message format"),
		},
		{
			caseDesc: "Test Incorrect Part: incorrect event",
			giveMsg: [][2]string{
				{"action", "add"},
				{"entity", "upstream;1"},
				{"service", "test"},
			},
			wantErr: fmt.Errorf("incorrect query part 1: give action, require event"),
		},
		{
			caseDesc: "Test Incorrect Part: incorrect entity",
			giveMsg: [][2]string{
				{"event", "add"},
				{"entities", "upstream;1"},
				{"service", "test"},
			},
			wantErr: fmt.Errorf("incorrect query part 2: give entities, require entity"),
		},
		{
			caseDesc: "Test Incorrect Part: incorrect service",
			giveMsg: [][2]string{
				{"event", "add"},
				{"entity", "upstream;1"},
				{"services", "test"},
			},
			wantErr: fmt.Errorf("incorrect query part 3: give services, require service"),
		},
		{
			caseDesc: "Test Incorrect Event",
			giveMsg: [][2]string{
				{"event", "remove"},
				{"entity", "upstream;1"},
				{"service", "test"},
			},
			wantErr: fmt.Errorf("incorrect query event: remove"),
		},
	}

	for _, tc := range tests {
		msg := make(utils.Message, 0, len(tc.giveMsg))
		for _, pair := range tc.giveMsg {
			msg.Add(pair[0], pair[1])
		}
		_, _, err := queryDecode(msg)
		assert.True(t, tc.wantErr.Error() == err.Error(), tc.caseDesc)
	}
}

func TestQueryEncode(t *testing.T) {
	tests := []struct {
		caseDesc   string
		giveMsg    [][2]string
		wantValues []string
		wantArgs   map[string]string
	}{
		{
			caseDesc: "Test Query Encode without extra arguments",
			giveMsg: [][2]string{
				{"event", "add"},
				{"entity", "upstream;1"},
				{"service", "test"},
			},
			wantValues: []string{"add", "upstream;1", "test"},
			wantArgs:   nil,
		},
		{
			caseDesc: "Test Query Encode with arguments",
			giveMsg: [][2]string{
				{"event", "add"},
				{"entity", "upstream;1"},
				{"service", "test"},
				{"namespace_id", "test_ns"},
			},
			wantValues: []string{"add", "upstream;1", "test"},
			wantArgs: map[string]string{
				"namespace_id": "test_ns",
			},
		},
	}

	for _, tc := range tests {
		msg := make(utils.Message, 0, len(tc.giveMsg))
		for _, pair := range tc.giveMsg {
			msg.Add(pair[0], pair[1])
		}
		values, args, err := queryDecode(msg)
		assert.Nil(t, err)
		assert.True(t, reflect.DeepEqual(values, tc.wantValues), tc.caseDesc)
		assert.True(t, reflect.DeepEqual(args, tc.wantArgs), tc.caseDesc)
	}
}

func TestWatchEncode(t *testing.T) {
	tests := []struct {
		caseDesc    string
		giveService Service
		wantMsg     string
	}{
		{
			caseDesc: "Test Watch Decode",
			giveService: Service{
				name: "test",
				nodes: []Node{
					{host: "127.0.0.1:80", weight: 10},
					{host: "127.0.0.1:8080", weight: 20},
				},
				entities: []string{
					"upstream;1",
					"upstream;2",
				},
				args: nil,
			},
			wantMsg: `key: event, value: update
key: service, value: test
key: entity, value: upstream;1
key: entity, value: upstream;2
key: node, value: 127.0.0.1:80
key: weight, value: 10
key: node, value: 127.0.0.1:8080
key: weight, value: 20`,
		},
	}

	for _, tc := range tests {
		msg := watchEncode(tc.giveService)
		assert.True(t, msg.String() == tc.wantMsg, tc.caseDesc)
	}
}
