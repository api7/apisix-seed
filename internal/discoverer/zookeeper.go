package discoverer

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/api7/apisix-seed/internal/core/message"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/log"
	"github.com/go-zookeeper/zk"
	"golang.org/x/net/context"
)

func init() {
	Discoveries["zookeeper"] = NewZookeeperDiscoverer
}

type ZookeeperService struct {
	Name         string
	mutex        *sync.Mutex
	BindEntities map[string]*message.Message
	WatchPath    string
	WatchContext context.Context
	WatchCancel  context.CancelFunc
}

type ZookeeperDiscoverer struct {
	zkConfig          *conf.Zookeeper
	zkConn            *zk.Conn
	zkWatchServices   sync.Map
	zkUnWatchServices sync.Map
	zkUnWatchContext  context.Context
	zkUnWatchCancel   context.CancelFunc

	msgCh chan *message.Message
}

func (zd *ZookeeperDiscoverer) Stop() {
	zd.zkWatchServices.Range(func(key, value interface{}) bool {
		zd.removeWatchService(value.(*ZookeeperService))
		return true
	})
	close(zd.msgCh)
	zd.zkConn.Close()
	zd.zkUnWatchCancel()
}

func (zd *ZookeeperDiscoverer) Query(msg *message.Message) error {
	return zd.fetchService(msg.ServiceName(), map[string]*message.Message{msg.Key: msg})
}

func (zd *ZookeeperDiscoverer) Update(oldMsg, msg *message.Message) error {
	zkService, ok := zd.zkWatchServices.Load(oldMsg.ServiceName())
	if !ok {
		return nil
	}
	service := zkService.(*ZookeeperService)
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if _, ok = service.BindEntities[oldMsg.Key]; ok {
		service.BindEntities[oldMsg.Key].Version = msg.Version
	}

	return nil
}

func (zd *ZookeeperDiscoverer) Delete(msg *message.Message) error {
	return zd.removeService(msg.ServiceName(), false)
}

func (zd *ZookeeperDiscoverer) Watch() chan *message.Message {
	return zd.msgCh
}

// fetchService fetch service watch and send message notify
func (zd *ZookeeperDiscoverer) fetchService(serviceName string, a6conf map[string]*message.Message) error {
	var service *ZookeeperService
	zkService, ok := zd.zkWatchServices.Load(serviceName)

	if ok {
		service = zkService.(*ZookeeperService)
	} else {
		var err error
		service, err = zd.newZookeeperClient(serviceName)
		if err != nil {
			return err
		}

		zd.addWatchService(service)
	}

	service.mutex.Lock()
	for k, msg := range a6conf {
		if _, ok = service.BindEntities[k]; !ok {
			service.BindEntities[k] = msg
		} else {
			service.BindEntities[k].Version = msg.Version
		}
	}
	service.mutex.Unlock()

	serviceInfo, _, err := zd.zkConn.Get(service.WatchPath)
	if err != nil {
		return err
	}

	var nodes []*message.Node
	err = json.Unmarshal(serviceInfo, &nodes)
	if err != nil {
		return err
	}

	zd.sendMessage(service, nodes)

	return nil
}

// removeService remove service watch and send message notify
func (zd *ZookeeperDiscoverer) removeService(serviceName string, isRewrite bool) error {
	zkService, ok := zd.zkWatchServices.Load(serviceName)
	if !ok {
		return errors.New("Zookeeper service: " + serviceName + " undefined")
	}

	if isRewrite {
		zd.sendMessage(zkService.(*ZookeeperService), make([]*message.Node, 0))
	}

	zd.removeWatchService(zkService.(*ZookeeperService))

	return nil
}

// sendMessage send message notify
func (zd *ZookeeperDiscoverer) sendMessage(zkService *ZookeeperService, nodes []*message.Node) {
	for _, msg := range zkService.BindEntities {
		msg.InjectNodes(nodes)
		zd.msgCh <- msg
	}
}

// NewZookeeperDiscoverer generate zookeeper discoverer instance
func NewZookeeperDiscoverer(disConfig interface{}) (Discoverer, error) {
	config := disConfig.(*conf.Zookeeper)

	conn, _, err := zk.Connect(config.Hosts, time.Second*time.Duration(config.Timeout))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	discoverer := ZookeeperDiscoverer{
		msgCh:             make(chan *message.Message, 10),
		zkConfig:          config,
		zkConn:            conn,
		zkWatchServices:   sync.Map{},
		zkUnWatchServices: sync.Map{},
		zkUnWatchContext:  ctx,
		zkUnWatchCancel:   cancel,
	}

	err = discoverer.initZookeeperRoot()
	if err != nil {
		return nil, err
	}

	go discoverer.watchServicePrefix()

	return &discoverer, nil
}

// initZookeeperRoot generate zookeeper root path
func (zd *ZookeeperDiscoverer) initZookeeperRoot() error {
	ok, _, err := zd.zkConn.Exists(zd.zkConfig.Prefix)
	if err != nil {
		return err
	}

	if !ok {
		_, err = zd.zkConn.Create(zd.zkConfig.Prefix, []byte(""), 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			return err
		}
	}

	return nil
}

// newZookeeperClient generate zookeeper client
func (zd *ZookeeperDiscoverer) newZookeeperClient(serviceName string) (*ZookeeperService, error) {
	ctx, cancel := context.WithCancel(context.Background())
	watchPath := zd.zkConfig.Prefix + "/" + serviceName
	service := &ZookeeperService{
		Name:         serviceName,
		mutex:        &sync.Mutex{},
		BindEntities: make(map[string]*message.Message),
		WatchPath:    watchPath,
		WatchContext: ctx,
		WatchCancel:  cancel,
	}

	return service, nil
}

// watchServicePrefix watch service prefix change, update unwatch service
func (zd *ZookeeperDiscoverer) watchServicePrefix() {
	for {
		_, _, event, err := zd.zkConn.ChildrenW(zd.zkConfig.Prefix)
		if err != nil {
			log.Errorf("watch service prefix: %s fail, err: %s", zd.zkConfig.Prefix, err)
			return
		}

		select {
		case <-zd.zkUnWatchContext.Done():
			return
		case <-event:
			var serviceNames []string
			serviceNames, _, err = zd.zkConn.Children(zd.zkConfig.Prefix)
			if err != nil {
				log.Errorf("fetch service prefix: %s fail, err: %s", zd.zkConfig.Prefix, err)
				continue
			}

			for _, serviceName := range serviceNames {
				a6Entity, ok := zd.zkUnWatchServices.Load(serviceName)
				if ok {
					err = zd.fetchService(serviceName, a6Entity.(map[string]*message.Message))
					if err != nil {
						log.Errorf("fetch service: %s fail, err: %s", serviceName, err)
					}
				}
			}
		}
	}
}

// watchService watch service change
func (zd *ZookeeperDiscoverer) watchService(service *ZookeeperService) {
	for {
		_, _, event, err := zd.zkConn.GetW(service.WatchPath)
		if err != nil {
			log.Errorf("watch service: %s fail, err: %s", service.WatchPath, err)
			zd.removeWatchService(service)
			return
		}

		select {
		case <-service.WatchContext.Done():
			log.Infof("watch service: %s cancel, err: %s", service.WatchPath, service.WatchContext.Err())
			return
		case e := <-event:
			switch e.Type {
			case zk.EventNodeDataChanged:
				err = zd.fetchService(service.Name, service.BindEntities)
				if err != nil {
					log.Errorf("fetch service: %s fail, err: %s", service.WatchPath, err)
				}
			case zk.EventNodeDeleted:
				err = zd.removeService(service.Name, true)
				if err != nil {
					log.Errorf("remove service: %s remove fail", err)
				}
			}
		}
	}
}

// addWatchService remove watch service
func (zd *ZookeeperDiscoverer) removeWatchService(service *ZookeeperService) {
	service.WatchCancel()
	zd.zkWatchServices.Delete(service.Name)
	zd.zkUnWatchServices.LoadOrStore(service.Name, service.BindEntities)
	log.Infof("stop watch service: %s", service.Name)
}

// addWatchService add watch service
func (zd *ZookeeperDiscoverer) addWatchService(service *ZookeeperService) {
	zd.zkWatchServices.LoadOrStore(service.Name, service)
	zd.zkUnWatchServices.Delete(service.Name)
	go zd.watchService(service)
	log.Infof("start watch service: %s", service.Name)
}
