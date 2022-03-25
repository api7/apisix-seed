package discoverer

import (
	"encoding/json"
	"fmt"
	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/api7/apisix-seed/internal/log"
	"github.com/api7/apisix-seed/internal/utils"
	"github.com/go-zookeeper/zk"
	"golang.org/x/net/context"
	"sort"
	"sync"
	"time"
)

func init() {
	Discoveries["zookeeper"] = NewZookeeperDiscoverer
}

type ZookeeperService struct {
	Name         string
	BindEntries  []string
	WatchConn    *zk.Conn
	RootPath     string
	WatchPath    string
	WatchContext context.Context
	WatchCancel  context.CancelFunc
	WatchMutex   sync.Mutex
}

type ZookeeperDiscoverer struct {
	timeout int
	weight  int

	zkConfig   *conf.Zookeeper
	zkServices map[string]*ZookeeperService
	eventMutex sync.Mutex

	msgCh chan *comm.Message
}

func (zd *ZookeeperDiscoverer) Stop() {
	for _, service := range zd.zkServices {
		zd.unsubscribe(service)
	}
	close(zd.msgCh)
}

func (zd *ZookeeperDiscoverer) Query(query *comm.Query) error {
	values, _, err := query.Decode()
	if err != nil {
		return err
	}

	event, entity, serviceName := values[0], values[1], values[2]

	switch event {
	case utils.EventAdd:
		zd.eventMutex.Lock()
		defer zd.eventMutex.Unlock()
		err = zd.fetchService(serviceName, []string{entity})
	case utils.EventDelete:
		zd.eventMutex.Lock()
		defer zd.eventMutex.Unlock()
		err = zd.removeService(serviceName)
	}

	if err != nil {
		return err
	}

	return nil
}

func (zd *ZookeeperDiscoverer) Update(update *comm.Update) error {
	// TODO:
	// Zookeeper does not need to introduce parameter information like nacos,
	// so the Update interface cannot be triggered on the framework.
	return nil
}

func (zd *ZookeeperDiscoverer) Watch() chan *comm.Message {
	return zd.msgCh
}

func (zd *ZookeeperDiscoverer) fetchService(serviceName string, entries []string) error {
	zkService, ok := zd.zkServices[serviceName]
	if !ok {
		var err error
		zkService, err = zd.newZookeeperClient(serviceName)
		if err != nil {
			return err
		}

		zd.zkServices[serviceName] = zkService

		go zd.subscribe(zkService)
	}

	for _, entry := range entries {
		if !inArray(entry, zkService.BindEntries) {
			zkService.BindEntries = append(zkService.BindEntries, entry)
		}
	}

	serviceInfo, _, err := zkService.WatchConn.Get(zkService.WatchPath)
	if err != nil {
		return err
	}

	node := Node{}
	err = json.Unmarshal(serviceInfo, &node)
	if err != nil {
		return err
	}

	zd.sendMessage(zkService, []Node{node})

	return nil
}

func (zd *ZookeeperDiscoverer) removeService(serviceName string) error {
	zkService, ok := zd.zkServices[serviceName]
	if ok {
		zd.unsubscribe(zkService)
	}

	zd.sendMessage(zkService, []Node{})

	return nil
}

func (zd *ZookeeperDiscoverer) sendMessage(zkService *ZookeeperService, nodes []Node) {
	messageService := &Service{
		name:     zkService.Name,
		nodes:    nodes,
		entities: make(map[string]struct{}),
	}
	for _, entry := range zkService.BindEntries {
		messageService.entities[entry] = struct{}{}
	}

	msg, err := messageService.NewNotifyMessage()
	if err != nil {
		log.Infof("Zookeeper send message fail, err: %s", err)
		return
	}

	zd.msgCh <- msg
}

func NewZookeeperDiscoverer(disConfig interface{}) (Discoverer, error) {
	config := disConfig.(*conf.Zookeeper)

	discoverer := ZookeeperDiscoverer{
		msgCh:      make(chan *comm.Message, 10),
		zkConfig:   config,
		zkServices: make(map[string]*ZookeeperService),
		eventMutex: sync.Mutex{},
	}

	return &discoverer, nil
}

func (zd *ZookeeperDiscoverer) newZookeeperClient(serviceName string) (*ZookeeperService, error) {
	conn, _, err := zk.Connect(zd.zkConfig.Hosts, time.Second*time.Duration(zd.zkConfig.Timeout))
	if err != nil {
		return nil, err
	}

	err = zd.initZookeeperRoot(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	watchPath := zd.zkConfig.Prefix + "/" + serviceName
	service := &ZookeeperService{
		Name:         serviceName,
		BindEntries:  make([]string, 0),
		WatchConn:    conn,
		RootPath:     zd.zkConfig.Prefix,
		WatchPath:    watchPath,
		WatchContext: ctx,
		WatchCancel:  cancel,
		WatchMutex:   sync.Mutex{},
	}

	return service, nil
}

func (zd *ZookeeperDiscoverer) initZookeeperRoot(zkClient *zk.Conn) error {
	ok, _, err := zkClient.Exists(zd.zkConfig.Prefix)
	if err != nil {
		return err
	}

	if !ok {
		_, err = zkClient.Create(zd.zkConfig.Prefix, []byte(""), 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			return err
		}
	}

	return nil
}

func (zd *ZookeeperDiscoverer) subscribe(service *ZookeeperService) {
	for {
		_, _, event, err := service.WatchConn.GetW(service.WatchPath)
		if err != nil {
			log.Infof("subscribe service: %s, err: %s", service.WatchPath, err)
		}

		_, _, childEvent, err := service.WatchConn.ChildrenW(service.RootPath)
		if err != nil {
			log.Infof("subscribe service root: %s, err: %s", service.RootPath, err)
		}

		select {
		case <-service.WatchContext.Done():
			fmt.Printf("subscribe service: %s cancel\n", service.WatchPath)
			log.Infof("subscribe service: %s cancel, err: %s", service.WatchPath, service.WatchContext.Err())
			return
		case e := <-event:
			switch e.Type {
			case zk.EventNodeDataChanged:
				err = zd.fetchService(service.Name, service.BindEntries)
				if err != nil {
					log.Infof("subscribe service: %s fail, err: %s", service.WatchPath, err)
				}
			case zk.EventNodeDeleted:
				err = zd.removeService(service.Name)
				if err != nil {
					log.Infof("subscribe service: %s remove fail", err)
				}
			}
		case e := <-childEvent:
			log.Infof("subscribe service root event: %s", e.Type.String())
		}
	}
}

func (zd *ZookeeperDiscoverer) unsubscribe(service *ZookeeperService) {
	service.WatchMutex.Lock()
	defer service.WatchMutex.Unlock()
	service.WatchCancel()
	service.WatchConn.Close()
	delete(zd.zkServices, service.Name)
	log.Infof("unsubscribe service: %s", service.Name)
}

func inArray(target string, array []string) bool {
	sort.Strings(array)
	index := sort.SearchStrings(array, target)
	if index < len(array) && array[index] == target {
		return true
	}
	return false
}
