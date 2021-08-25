package storer

import (
	"errors"
	"fmt"

	"github.com/api7/apisix-seed/internal/utils"
)

var eventHeader = []string{"event", "key", "value"}

type Event struct {
	header utils.Message
}

type Watch struct {
	Events   []Event
	Error    error
	Canceled bool
}

func NewWatch(canceled bool) Watch {
	var err error = nil
	if canceled {
		err = fmt.Errorf("watch channel canceled")
	}

	return Watch{
		Events:   make([]Event, 0),
		Error:    err,
		Canceled: canceled,
	}
}

func (msg *Watch) Add(event, key, value string) error {
	// Check watch event
	switch event {
	case utils.EventAdd, utils.EventUpdate, utils.EventDelete:
	default:
		err := fmt.Sprintf("incorrect event: %s", event)
		return errors.New(err)
	}

	h := make(utils.Message, len(eventHeader))
	vals := []string{event, key, value}

	for i := range eventHeader {
		h.Add(eventHeader[i], vals[i])
	}

	msg.Events = append(msg.Events, Event{h})
	return nil
}

// Decode check and extract values from the watch message
func (msg *Watch) Decode() ([][]string, error) {
	if len(msg.Events) == 0 {
		return nil, errors.New("incorrect watch content")
	}

	msgValues := make([][]string, len(msg.Events))
	for i, event := range msg.Events {
		msgValues[i] = make([]string, len(eventHeader))
		for j, pair := range event.header {
			msgValues[i][j] = pair.Value
		}
	}

	return msgValues, nil
}
