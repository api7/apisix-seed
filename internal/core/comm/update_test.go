package comm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdate(t *testing.T) {
	tests := []struct {
		caseDesc    string
		giveValues  []string
		giveOldArgs map[string]string
		giveNewArgs map[string]string
	}{
		{
			caseDesc:    "Test Update Encode without old args",
			giveValues:  []string{"update", "test"},
			giveOldArgs: nil,
			giveNewArgs: map[string]string{
				"namespace_id": "test_ns",
			},
		},
		{
			caseDesc:   "Test Update Encode with old args",
			giveValues: []string{"update", "test"},
			giveOldArgs: map[string]string{
				"namespace_id": "old_ns",
			},
			giveNewArgs: map[string]string{
				"namespace_id": "test_ns",
			},
		},
	}

	for _, tc := range tests {
		update, err := NewUpdate(tc.giveValues, tc.giveOldArgs, tc.giveNewArgs)
		assert.Nil(t, err, tc.caseDesc)

		values, oldArgs, newArgs, err := update.Decode()
		assert.Nil(t, err, tc.caseDesc)
		assert.Equal(t, tc.giveValues, values, tc.caseDesc)
		assert.Equal(t, tc.giveOldArgs, oldArgs, tc.caseDesc)
		assert.Equal(t, tc.giveNewArgs, newArgs, tc.caseDesc)
	}
}
