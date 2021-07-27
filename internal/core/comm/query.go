package comm

import (
	"errors"
	"fmt"

	"github.com/api7/apisix-seed/internal/utils"
)

var queryHeader = [3]string{"event", "entity", "service"}

type Query struct {
	header utils.Message
	args   utils.Message
}

func headerCheck(header utils.Message) error {
	if len(header) < len(queryHeader) {
		return errors.New("incorrect query message format")
	}

	// Check the required parts of the query message
	for idx, check := range queryHeader {
		if header[idx].Key != check {
			err := fmt.Sprintf("incorrect query part %d: give %s, require %s", idx+1, header[idx].Key, check)
			return errors.New(err)
		}
	}

	// Check query event
	switch event := header[0].Value; event {
	case utils.EventAdd, utils.EventUpdate, utils.EventDelete:
	default:
		err := fmt.Sprintf("incorrect query event: %s", event)
		return errors.New(err)
	}

	return nil
}

// Decode check and extract values from the query message
func (msg *Query) Decode() ([]string, map[string]string, error) {
	if err := headerCheck(msg.header); err != nil {
		return nil, nil, err
	}

	msgValues := make([]string, len(msg.header))
	for idx, pair := range msg.header {
		msgValues[idx] = pair.Value
	}

	// If additional service discovery arguments exist
	var msgArgs map[string]string
	if len(msg.args) != 0 {
		msgArgs = make(map[string]string, len(msg.args))
		for _, pair := range msg.args {
			msgArgs[pair.Key] = pair.Value
		}
	}

	return msgValues, msgArgs, nil
}
