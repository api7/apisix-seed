package comm

import (
	"github.com/api7/apisix-seed/internal/utils"
	"go.uber.org/zap/buffer"
)

var updateHeader = []string{"event", "service"}

type Update struct {
	header  utils.Message
	oldArgs utils.Message
	newArgs utils.Message
}

func NewUpdate(values []string, oldArgs, newArgs map[string]string) (Update, error) {
	header, err := newHeader(values, updateHeader)
	if err != nil {
		return Update{}, err
	}

	oldArgsMsg := make(utils.Message, 0, len(oldArgs))
	for key, value := range oldArgs {
		oldArgsMsg.Add(key, value)
	}
	newArgsMsg := make(utils.Message, 0, len(newArgs))
	for key, value := range newArgs {
		newArgsMsg.Add(key, value)
	}

	return Update{
		header:  header,
		oldArgs: oldArgsMsg,
		newArgs: newArgsMsg,
	}, nil
}

// Decode check and extract values from the update message
func (msg *Update) Decode() ([]string, map[string]string, map[string]string, error) {
	if err := headerCheck(msg.header, updateHeader); err != nil {
		return nil, nil, nil, err
	}

	msgValues := make([]string, len(msg.header))
	for idx, pair := range msg.header {
		msgValues[idx] = pair.Value
	}

	// If additional service discovery arguments exist
	var msgOldArgs map[string]string
	if len(msg.oldArgs) != 0 {
		msgOldArgs = make(map[string]string, len(msg.oldArgs))
		for _, pair := range msg.oldArgs {
			msgOldArgs[pair.Key] = pair.Value
		}
	}
	var msgNewArgs map[string]string
	if len(msg.newArgs) != 0 {
		msgNewArgs = make(map[string]string, len(msg.newArgs))
		for _, pair := range msg.newArgs {
			msgNewArgs[pair.Key] = pair.Value
		}
	}

	return msgValues, msgOldArgs, msgNewArgs, nil
}

func (msg *Update) String() string {
	msgString := buffer.Buffer{}

	msgs := []utils.Message{msg.header, msg.oldArgs, msg.newArgs}
	for i, msg := range msgs {
		str := msg.String()
		if i != 0 {
			msgString.AppendString("\n")
		}
		msgString.AppendString(str)
	}
	return msgString.String()
}
