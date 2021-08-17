package comm

import (
	"github.com/api7/apisix-seed/internal/utils"
)

var queryHeader = []string{"event", "entity", "service"}

type Query struct {
	header utils.Message
	args   utils.Message
}

func NewQuery(values []string, args map[string]string) (Query, error) {
	header, err := newHeader(values, queryHeader)
	if err != nil {
		return Query{}, err
	}

	argsMsg := make(utils.Message, 0, len(args))
	for key, value := range args {
		argsMsg.Add(key, value)
	}

	return Query{
		header: header,
		args:   argsMsg,
	}, nil
}

// Decode check and extract values from the query message
func (msg *Query) Decode() ([]string, map[string]string, error) {
	if err := headerCheck(msg.header, queryHeader); err != nil {
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
