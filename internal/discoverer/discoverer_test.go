package discoverer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceEncodeWatch(t *testing.T) {
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
					{Host: "127.0.0.1:80", Weight: 10},
					{Host: "127.0.0.1:8080", Weight: 20},
				},
				entities: map[string]struct{}{
					"upstream;2": {},
					"upstream;1": {},
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
key: weight, value: 20`},
	}

	for _, tc := range tests {
		msg, err := tc.giveService.NewNotifyMessage()
		assert.Nil(t, err)
		assert.True(t, msg.String() == tc.wantMsg, tc.caseDesc)
	}
}
