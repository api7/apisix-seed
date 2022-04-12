package message

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
	Key    string
	Value  string
	Action StoreEvent
	a6Conf *A6Conf
}

func NewMessage(key string, value []byte, action StoreEvent) (*Message, error) {
	msg := &Message{
		Key:    key,
		Value:  string(value),
		Action: action,
	}
	if len(value) != 0 {
		a6, err := NewA6Conf(value)
		if err != nil {
			return nil, err
		}
		msg.a6Conf = a6
	}
	return msg, nil
}

func (msg *Message) ServiceName() string {
	up := msg.a6Conf.Upstream
	if up.ServiceName != "" {
		return up.ServiceName
	}
	return up.DupServiceName
}

func (msg *Message) DiscoveryType() string {
	up := msg.a6Conf.Upstream
	if up.DiscoveryType != "" {
		return up.DiscoveryType
	}
	return up.DupDiscoveryType
}

func (msg *Message) DiscoveryArgs() map[string]string {
	up := msg.a6Conf.Upstream
	if up.DiscoveryArgs == nil {
		return nil
	}
	return map[string]string{
		"namespace_id": up.DiscoveryArgs.NamespaceID,
		"group_name":   up.DiscoveryArgs.GroupName,
	}
}

func (msg *Message) InjectNodes(nodes interface{}) {
	msg.a6Conf.Inject(nodes)
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
		args["namespace_id"] != newArgs["namespace_id"] {
		return true
	}

	return false
}

func ServiceReplace(msg, newMsg *Message) bool {
	return msg.ServiceName() != newMsg.ServiceName() ||
		msg.DiscoveryType() != newMsg.DiscoveryType()
}
