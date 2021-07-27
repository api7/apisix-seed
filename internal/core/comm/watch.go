package comm

import (
	"errors"

	"github.com/api7/apisix-seed/internal/utils"
	"go.uber.org/zap/buffer"
)

var watchHeader = [2]string{"event", "service"}

type Watch struct {
	header   utils.Message
	entities utils.Message
	nodes    utils.Message
}

func NewWatchHeader(values []string) (utils.Message, error) {
	if len(values) != len(watchHeader) {
		return nil, errors.New("incorrect watch header values")
	}

	msg := make(utils.Message, 0, len(watchHeader))
	for idx, key := range watchHeader {
		msg.Add(key, values[idx])
	}
	return msg, nil
}

func NewWatch(header, entities, nodes utils.Message) Watch {
	return Watch{
		header:   header,
		entities: entities,
		nodes:    nodes,
	}
}

func (msg *Watch) String() string {
	msgString := buffer.Buffer{}

	msgs := []utils.Message{msg.header, msg.entities, msg.nodes}
	for i, msg := range msgs {
		str := msg.String()
		if i != 0 {
			msgString.AppendString("\n")
		}
		msgString.AppendString(str)
	}
	return msgString.String()
}
