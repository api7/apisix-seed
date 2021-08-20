package comm

import (
	"errors"
	"fmt"

	"github.com/api7/apisix-seed/internal/utils"
)

func headerCheck(header utils.Message, needHeader []string) error {
	if len(header) < len(needHeader) {
		return errors.New("incorrect message header format")
	}

	// Check the required parts of the query message
	for idx, check := range needHeader {
		if header[idx].Key != check {
			err := fmt.Sprintf("incorrect header part %d: give %s, require %s", idx+1, header[idx].Key, check)
			return errors.New(err)
		}
	}

	// Check query event
	switch event := header[0].Value; event {
	case utils.EventAdd, utils.EventUpdate, utils.EventDelete:
	default:
		err := fmt.Sprintf("incorrect header event: %s", event)
		return errors.New(err)
	}

	return nil
}

func newHeader(values, needHeader []string) (utils.Message, error) {
	if len(values) != len(needHeader) {
		return nil, errors.New("incorrect header values")
	}

	msg := make(utils.Message, 0, len(needHeader))
	for idx, key := range needHeader {
		msg.Add(key, values[idx])
	}
	return msg, nil
}
