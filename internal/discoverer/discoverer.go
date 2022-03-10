package discoverer

import (
	"sort"
	"strconv"

	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/api7/apisix-seed/internal/utils"
)

type NewDiscoverFunc func(disConfig interface{}) (Discoverer, error)

// Discoverer defines the component that interact nacos, consul and so on
type Discoverer interface {
	Stop()
	Query(*comm.Query) error
	Update(*comm.Update) error
	Watch() chan *comm.Message
}

// Node defines the upstream machine information
type Node struct {
	host   string
	weight int
}

// Service defines the service information for discoverer
type Service struct {
	name     string
	nodes    []Node              // nodes are the upstream machines of the service
	entities map[string]struct{} // entities are the upstreams/services/routes that use the service
	args     map[string]string
}

func (s *Service) encodeNodes() utils.Message {
	msg := make(utils.Message, 0, 2*len(s.nodes))
	for _, node := range s.nodes {
		msg.Add("node", node.host)
		msg.Add("weight", strconv.Itoa(node.weight))
	}
	return msg
}

func (s *Service) encodeEntities() utils.Message {
	msg := make(utils.Message, 0, len(s.entities))
	for entity := range s.entities {
		msg.Add("entity", entity)
	}
	sort.Slice(msg, func(i, j int) bool {
		return msg[i].Value < msg[j].Value
	})

	return msg
}

// NewCommonMessage encodes a service to the watch message
// message:
///{
//	header: [event->add, serviceName->APISIX_NACOS]
//	entities: [/apisix/routes/1, /apisix/service/1, /apisix/upstream/1]
//	nodes: [127.0.0.1:10000:1, 127.0.0.1:10001:1]
//}
func (s *Service) NewNotifyMessage() (*comm.Message, error) {
	headerVals := []string{utils.EventUpdate, s.name}
	entities := s.encodeEntities()
	nodes := s.encodeNodes()

	msg, err := comm.NewMessage(headerVals, entities, nodes)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
