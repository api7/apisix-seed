package comm

import (
	"fmt"
	"strconv"

	"github.com/api7/apisix-seed/internal/utils"
	"go.uber.org/zap/buffer"
)

var msgHeader = []string{"event", "service"}

type Message struct {
	header   utils.Message
	entities utils.Message
	nodes    utils.Message
}

func NewMessage(values []string, entities, nodes utils.Message) (Message, error) {
	header, err := newHeader(values, msgHeader)
	if err != nil {
		return Message{}, err
	}

	return Message{
		header:   header,
		entities: entities,
		nodes:    nodes,
	}, nil
}

func (msg *Message) Decode() ([]string, []string, map[string]float64, error) {
	if err := headerCheck(msg.header, msgHeader); err != nil {
		return nil, nil, nil, err
	}

	msgValues := make([]string, len(msg.header))
	for idx, pair := range msg.header {
		msgValues[idx] = pair.Value
	}

	var msgEntities []string
	if n := len(msg.entities); n != 0 {
		msgEntities = make([]string, n)
		for idx, pair := range msg.entities {
			msgEntities[idx] = pair.Value
		}
	}

	var msgNodes map[string]float64
	if n := len(msg.nodes); n != 0 {
		msgNodes = make(map[string]float64, n/2)
		for i := 0; i < len(msg.nodes); i += 2 {
			weight, err := strconv.ParseFloat(msg.nodes[i+1].Value, 64)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("wrong weight format")
			}

			msgNodes[msg.nodes[i].Value] = weight
		}
	}

	return msgValues, msgEntities, msgNodes, nil
}

func (msg *Message) String() string {
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
