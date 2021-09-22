package comm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuery(t *testing.T) {
	tests := []struct {
		caseDesc   string
		giveValues []string
		giveArgs   map[string]string
	}{
		{
			caseDesc:   "Test Query Encode without extra arguments",
			giveValues: []string{"add", "upstream;1", "test"},
			giveArgs:   nil,
		},
		{
			caseDesc:   "Test Query Encode with arguments",
			giveValues: []string{"update", "service;1", "test"},
			giveArgs: map[string]string{
				"namespace_id": "test_ns",
			},
		},
	}

	for _, tc := range tests {
		query, err := NewQuery(tc.giveValues, tc.giveArgs)
		assert.Nil(t, err, tc.caseDesc)

		values, args, err := query.Decode()
		assert.Nil(t, err)
		assert.Equal(t, tc.giveValues, values, tc.caseDesc)
		assert.Equal(t, tc.giveArgs, args, tc.caseDesc)
	}
}
