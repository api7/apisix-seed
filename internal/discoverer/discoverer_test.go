package discoverer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceEncodeWatch(t *testing.T) {
	tests := []struct {
		caseDesc    string
		giveService Service
		wantMsg     []string
	}{
		{
			caseDesc: "Test Watch Decode",
			giveService: Service{
				name: "test",
				nodes: []Node{
					{host: "127.0.0.1:80", weight: 10},
					{host: "127.0.0.1:8080", weight: 20},
				},
				entities: map[string]struct{}{
					"upstream;1": {},
					"upstream;2": {},
				},
				args: nil,
			},
			wantMsg: []string{`key: event, value: update
key: service, value: test
key: entity, value: upstream;1
key: entity, value: upstream;2
key: node, value: 127.0.0.1:80
key: weight, value: 10
key: node, value: 127.0.0.1:8080
key: weight, value: 20`, `key: event, value: update
key: service, value: test
key: entity, value: upstream;2
key: entity, value: upstream;1
key: node, value: 127.0.0.1:80
key: weight, value: 10
key: node, value: 127.0.0.1:8080
key: weight, value: 20`}},
	}

	for _, tc := range tests {
		watch, err := tc.giveService.EncodeWatch()
		assert.Nil(t, err)
		ret := false
		for _, msg := range tc.wantMsg {
			if watch.String() == msg {
				ret = true
				break
			}
		}
		assert.True(t, ret, tc.caseDesc)
	}
}
