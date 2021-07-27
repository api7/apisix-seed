package discoverer

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/api7/apisix-seed/internal/utils"
)

var (
	queryChecks = [3]string{"event", "entity", "service"}
	Discoveries = make(map[string]Discover)
)

type Discover func(disConfig interface{}) Discoverer

// Discoverer defines the component that interact nacos, consul and so on
type Discoverer interface {
	Stop()
	Query(utils.Message) error
	Watch() chan utils.Message
}

type Node struct {
	host   string
	weight int
}

type Service struct {
	name     string
	nodes    []Node
	entities []string
	args     map[string]string
}

// queryDecode check and extract values from the query message
func queryDecode(msg utils.Message) ([]string, map[string]string, error) {
	msgLength := len(msg)
	if msgLength < 3 {
		return nil, nil, errors.New("incorrect query message format")
	}

	// Check the required parts of the query message
	msgValues := make([]string, 3)
	for i := 0; i < 3; i++ {
		if msg[i].Key != queryChecks[i] {
			err := fmt.Sprintf("incorrect query part %d: give %s, require %s", i+1, msg[i].Key, queryChecks[i])
			return nil, nil, errors.New(err)
		}
		msgValues[i] = msg[i].Value
	}

	// Check query event
	switch event := msgValues[0]; event {
	case utils.EventAdd, utils.EventUpdate, utils.EventDelete:
	default:
		err := fmt.Sprintf("incorrect query event: %s", event)
		return nil, nil, errors.New(err)
	}

	// If additional service discovery arguments exist
	var msgArgs map[string]string
	if msgLength > 3 {
		msgArgs = make(map[string]string, msgLength-3)
		for i := 3; i < msgLength; i++ {
			msgArgs[msg[i].Key] = msg[i].Value
		}
	}

	return msgValues, msgArgs, nil
}

// watchEncode encodes a service to the watch message
func watchEncode(service Service) utils.Message {
	msg := make(utils.Message, 0, 2+len(service.entities)+2*len(service.nodes))
	msg.Add("event", utils.EventUpdate)
	msg.Add("service", service.name)
	for _, entity := range service.entities {
		msg.Add("entity", entity)
	}
	for _, node := range service.nodes {
		msg.Add("node", node.host)
		msg.Add("weight", strconv.Itoa(node.weight))
	}

	return msg
}
