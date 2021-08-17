package comm

import (
	"fmt"
	"testing"

	"github.com/api7/apisix-seed/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestHeaderCheck(t *testing.T) {
	tests := []struct {
		caseDesc string
		giveMsg  [][2]string
		wantErr  error
	}{
		{
			caseDesc: "Test Wrong Format: empty message",
			giveMsg:  [][2]string{},
			wantErr:  fmt.Errorf("incorrect message header format"),
		},
		{
			caseDesc: "Test Incorrect Part: incorrect event",
			giveMsg: [][2]string{
				{"action", "add"},
				{"entity", "upstream;1"},
				{"service", "test"},
			},
			wantErr: fmt.Errorf("incorrect header part 1: give action, require event"),
		},
		{
			caseDesc: "Test Incorrect Part: incorrect entity",
			giveMsg: [][2]string{
				{"event", "add"},
				{"entities", "upstream;1"},
				{"service", "test"},
			},
			wantErr: fmt.Errorf("incorrect header part 2: give entities, require entity"),
		},
		{
			caseDesc: "Test Incorrect Part: incorrect service",
			giveMsg: [][2]string{
				{"event", "add"},
				{"entity", "upstream;1"},
				{"services", "test"},
			},
			wantErr: fmt.Errorf("incorrect header part 3: give services, require service"),
		},
		{
			caseDesc: "Test Incorrect Event",
			giveMsg: [][2]string{
				{"event", "remove"},
				{"entity", "upstream;1"},
				{"service", "test"},
			},
			wantErr: fmt.Errorf("incorrect header event: remove"),
		},
	}

	for _, tc := range tests {
		msg := make(utils.Message, 0, len(tc.giveMsg))
		for _, pair := range tc.giveMsg {
			msg.Add(pair[0], pair[1])
		}
		err := headerCheck(msg, queryHeader)
		assert.True(t, tc.wantErr.Error() == err.Error(), tc.caseDesc)
	}
}
