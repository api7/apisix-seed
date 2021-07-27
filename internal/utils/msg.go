package utils

import (
	"fmt"

	"go.uber.org/zap/buffer"
)

const (
	EventAdd    = "add"
	EventUpdate = "update"
	EventDelete = "delete"
)

// kvPair contains a key and a string
type kvPair struct {
	Key   string
	Value string
}

// Message contains some kvPair
type Message []*kvPair

func (msg *Message) Add(key, value string) {
	kvPairs := []*kvPair(*msg)
	*msg = append(kvPairs, &kvPair{Key: key, Value: value})
}

func (msg *Message) String() string {
	msgString := buffer.Buffer{}
	for i, pair := range *msg {
		if i != 0 {
			msgString.AppendString("\n")
		}
		msgString.AppendString(fmt.Sprintf("key: %s, value: %s", pair.Key, pair.Value))
	}
	return msgString.String()
}
