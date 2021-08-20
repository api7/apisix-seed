package comm

import (
	"github.com/api7/apisix-seed/internal/utils"
	"go.uber.org/zap/buffer"
)

var watchHeader = []string{"event", "service"}

type Watch struct {
	header   utils.Message
	entities utils.Message
	nodes    utils.Message
}

func NewWatch(values []string, entities, nodes utils.Message) (Watch, error) {
	header, err := newHeader(values, watchHeader)
	if err != nil {
		return Watch{}, err
	}

	return Watch{
		header:   header,
		entities: entities,
		nodes:    nodes,
	}, nil
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
