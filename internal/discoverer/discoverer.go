package discoverer

import (
	"github.com/api7/apisix-seed/internal/core/message"
)

type NewDiscoverFunc func(disConfig interface{}) (Discoverer, error)

// Discoverer defines the component that interact nacos, consul and so on
type Discoverer interface {
	Stop()
	Query(*message.Message) error
	Update(*message.Message, *message.Message) error
	Delete(*message.Message) error
	Watch() chan *message.Message
}
