package storer

import (
	"errors"
	"fmt"

	"github.com/api7/apisix-seed/internal/log"
	"github.com/api7/apisix-seed/internal/utils"
	"go.uber.org/zap/buffer"
)

var eventHeader = []string{"event", "key", "value"}

type Event struct {
	header utils.Message
}

type StoreEvent struct {
	Events   []Event
	Error    error
	Canceled bool
}

func NewStoreEvent(canceled bool) StoreEvent {
	var err error = nil
	if canceled {
		err = fmt.Errorf("StoreEvent channel canceled")
	}

	return StoreEvent{
		Events:   make([]Event, 0),
		Error:    err,
		Canceled: canceled,
	}
}

func (msg *StoreEvent) Add(event, key, value string) error {
	// Check store event
	switch event {
	case utils.EventAdd, utils.EventUpdate, utils.EventDelete:
	default:
		err := fmt.Sprintf("incorrect event: %s", event)
		log.Error(err)
		return errors.New(err)
	}

	h := make(utils.Message, 0, len(eventHeader))
	vals := []string{event, key, value}

	for i := range eventHeader {
		h.Add(eventHeader[i], vals[i])
	}

	msg.Events = append(msg.Events, Event{h})
	return nil
}

// Decode check and extract values from the watch message
func (msg *StoreEvent) Decode() ([][]string, error) {
	if len(msg.Events) == 0 {
		return nil, errors.New("incorrect watch content")
	}

	msgValues := make([][]string, len(msg.Events))
	for i, event := range msg.Events { //event: [header, header, header], header-> [{key:event, value:add}, {key:key, value:/apisix/routes/9},{key:value, value:data}]
		msgValues[i] = make([]string, len(eventHeader))
		for j, pair := range event.header {
			msgValues[i][j] = pair.Value
		}
	}

	return msgValues, nil
}

func (msg *StoreEvent) String() string {
	msgString := buffer.Buffer{}

	for i, ev := range msg.Events {
		msg := ev.header
		str := msg.String()
		if i != 0 {
			msgString.AppendString("\n")
		}
		msgString.AppendString(str)
	}
	return msgString.String()
}
