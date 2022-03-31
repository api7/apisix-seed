package discoverer

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/api7/apisix-seed/internal/log"
	"github.com/api7/apisix-seed/internal/utils"
	"github.com/go-zookeeper/zk"
	"golang.org/x/net/context"
)

func init() {
	Discoveries["zookeeper"] = NewZookeeperDiscoverer
}

type ZookeeperService struct {
	Name         string
	BindEntities []string
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

	msgCh chan *comm.Message
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

func (zd *ZookeeperDiscoverer) Query(query *comm.Query) error {
	values, _, err := query.Decode()
	if err != nil {
		return err
	}

	event, entity, serviceName := values[0], values[1], values[2]

	switch event {
	case utils.EventAdd:
		err = zd.fetchService(serviceName, []string{entity})
	case utils.EventDelete:
		err = zd.removeService(serviceName)
	}

	return err
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

// fetchService fetch service watch and send message notify
func (zd *ZookeeperDiscoverer) fetchService(serviceName string, entities []string) error {
	zkService, ok := zd.zkWatchServices.Load(serviceName)

	if !ok {
		var err error
		zkService, err = zd.newZookeeperClient(serviceName)
		if err != nil {
			return err
		}

		zd.addWatchService(zkService.(*ZookeeperService))
	}

	for _, entity := range entities {
		entryExists := false
		for _, bindEntry := range zkService.(*ZookeeperService).BindEntities {
			if entity == bindEntry {
				entryExists = true
				break
			}
		}

		if !entryExists {
			zkService.(*ZookeeperService).BindEntities = append(zkService.(*ZookeeperService).BindEntities, entity)
		}
	}

	serviceInfo, _, err := zd.zkConn.Get(zkService.(*ZookeeperService).WatchPath)
	if err != nil {
		return err
	}

	node := Node{}
	err = json.Unmarshal(serviceInfo, &node)
	if err != nil {
		return err
	}

	zd.sendMessage(zkService.(*ZookeeperService), []Node{node})

	return nil
}

// removeService remove service watch and send message notify
func (zd *ZookeeperDiscoverer) removeService(serviceName string) error {
	zkService, ok := zd.zkWatchServices.Load(serviceName)
	if !ok {
		return errors.New("Zookeeper service: " + serviceName + " undefined")
	}

	zd.sendMessage(zkService.(*ZookeeperService), make([]Node, 0))
	zd.removeWatchService(zkService.(*ZookeeperService))

	return nil
}

// sendMessage send message notify
func (zd *ZookeeperDiscoverer) sendMessage(zkService *ZookeeperService, nodes []Node) {
	messageService := &Service{
		name:     zkService.Name,
		nodes:    nodes,
		entities: make(map[string]struct{}),
	}

	for _, entry := range zkService.BindEntities {
		messageService.entities[entry] = struct{}{}
	}

	msg, err := messageService.NewNotifyMessage()
	if err != nil {
		log.Errorf("Zookeeper send message fail, err: %s", err)
		return
	}

	zd.msgCh <- msg
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
		msgCh:             make(chan *comm.Message, 10),
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
		BindEntities: make([]string, 0),
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
					err = zd.fetchService(serviceName, a6Entity.([]string))
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
				err = zd.removeService(service.Name)
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
