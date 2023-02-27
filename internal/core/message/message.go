package message

import (
	"reflect"
)

type StoreEvent = int

const (
	// add or update config
	EventAdd StoreEvent = 0x01
	// delete config
	EventDelete StoreEvent = 0102
)

type Node struct {
	Host     string      `json:"host,omitempty"`
	Port     int         `json:"port,omitempty"`
	Weight   int         `json:"weight"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type Message struct {
	Key     string
	Value   string
	Version int64
	Action  StoreEvent
	a6Conf  A6Conf
}

func NewMessage(key string, value []byte, version int64, action StoreEvent, a6Type int) (*Message, error) {
	msg := &Message{
		Key:     key,
		Value:   string(value),
		Version: version,
		Action:  action,
	}
	if len(value) != 0 {
		a6, err := NewA6Conf(value, a6Type)
		if err != nil {
			return nil, err
		}
		msg.a6Conf = a6
	}
	return msg, nil
}

func (msg *Message) ServiceName() string {
	up := msg.a6Conf.GetUpstream()
	if up.ServiceName != "" {
		return up.ServiceName
	}
	return up.DupServiceName
}

func (msg *Message) DiscoveryType() string {
	up := msg.a6Conf.GetUpstream()
	if up.DiscoveryType != "" {
		return up.DiscoveryType
	}
	return up.DupDiscoveryType
}

func (msg *Message) DiscoveryArgs() map[string]interface{} {
	up := msg.a6Conf.GetUpstream()
	if up.DiscoveryArgs == nil {
		return nil
	}
	return map[string]interface{}{
		"namespace_id": up.DiscoveryArgs.NamespaceID,
		"group_name":   up.DiscoveryArgs.GroupName,
		"metadata":     up.DiscoveryArgs.Metadata,
	}
}

func (msg *Message) InjectNodes(nodes interface{}) {
	msg.a6Conf.Inject(nodes)
}

func (msg *Message) HasNodesAttr() bool {
	return msg.a6Conf.HasNodesAttr()
}

func (msg *Message) Marshal() ([]byte, error) {
	return msg.a6Conf.Marshal()
}

func ServiceFilter(msg *Message) bool {
	if msg.ServiceName() != "" && msg.DiscoveryType() != "" {
		return true
	}
	return false
}

func ServiceUpdate(msg, newMsg *Message) bool {
	if msg.ServiceName() != newMsg.ServiceName() || msg.DiscoveryType() != newMsg.DiscoveryType() {
		return false
	}

	// Two pointers are equal only when they are both nil
	args := msg.DiscoveryArgs()
	newArgs := newMsg.DiscoveryArgs()
	if args == nil && newArgs == nil {
		return false
	}
	if (args == nil && newArgs != nil) || (args != nil && newArgs == nil) {
		return true
	}
	if args["group_name"] != newArgs["group_name"] ||
		args["namespace_id"] != newArgs["namespace_id"] ||
		!reflect.DeepEqual(args["metadata"], newArgs["metadata"]) {
		return true
	}

	return false
}

func ServiceReplace(msg, newMsg *Message) bool {
	return msg.ServiceName() != newMsg.ServiceName() ||
		msg.DiscoveryType() != newMsg.DiscoveryType()
}
