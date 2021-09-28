package discoverer

import (
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/api7/apisix-seed/internal/log"
	"github.com/api7/apisix-seed/internal/utils"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func init() {
	Discoveries["nacos"] = NewNacosDiscoverer
}

func serviceID(service string, args map[string]string) string {
	serviceId := fmt.Sprintf("%s@%s@%s", args["namespace_id"], args["group_name"], service)
	return serviceId
}

type NacosDiscoverer struct {
	timeout uint64
	weight  int
	// nacos server configs, grouping by authentication information
	ServerConfigs map[string][]constant.ServerConfig
	// nacos naming clients, grouping by authentication information
	namingClients map[string][]naming_client.INamingClient

	params map[string]*vo.SubscribeParam
	cache  map[string]*Service
	mu     sync.Mutex

	crc hash.Hash32

	watchCh chan *comm.Watch
}

func NewNacosDiscoverer(disConfig interface{}) (Discoverer, error) {
	config := disConfig.(*conf.Nacos)

	serverConfigs := make(map[string][]constant.ServerConfig)
	for _, host := range config.Host {
		u, err := url.Parse(host)
		if err != nil {
			log.Errorf("parse url fail: %s", err)
			return nil, err
		}

		port := 8848 // nacos default port
		if portStr := u.Port(); len(portStr) != 0 {
			port, _ = strconv.Atoi(portStr)
		}

		auth := u.User.String()
		serverConfigs[auth] = append(serverConfigs[auth], constant.ServerConfig{
			IpAddr:      u.Hostname(),
			Port:        uint64(port),
			Scheme:      u.Scheme,
			ContextPath: config.Prefix,
		})
	}

	timeout := config.Timeout
	discoverer := NacosDiscoverer{
		// compatible with past timeout configurations
		timeout:       uint64(timeout.Connect + timeout.Read + timeout.Send),
		weight:        config.Weight,
		ServerConfigs: serverConfigs,
		namingClients: make(map[string][]naming_client.INamingClient),
		params:        make(map[string]*vo.SubscribeParam),
		cache:         make(map[string]*Service),
		mu:            sync.Mutex{},
		crc:           crc32.NewIEEE(),
		watchCh:       make(chan *comm.Watch, 10),
	}
	return &discoverer, nil
}

func (d *NacosDiscoverer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	close(d.watchCh)

	// Unsubscribe all services
	for _, service := range d.cache {
		d.unsubscribe(service)
	}
}

func (d *NacosDiscoverer) Query(query *comm.Query) error {
	values, args, err := query.Decode()
	if err != nil {
		return err
	}

	event, entity, service := values[0], values[1], values[2]
	serviceId := serviceID(service, args)

	d.mu.Lock()
	switch event {
	case utils.EventAdd:
		ok := false
		var cacheService *Service
		if cacheService, ok = d.cache[serviceId]; ok {
			// cache information is already available
			cacheService.entities[entity] = struct{}{}
		} else {
			// fetch new service information
			d.mu.Unlock()

			nodes, err := d.fetch(service, args)
			if err != nil {
				return err
			}

			cacheService = &Service{
				name:     service,
				nodes:    nodes,
				entities: map[string]struct{}{entity: {}},
				args:     args,
			}

			d.mu.Lock()
			d.cache[serviceId] = cacheService
		}

		d.mu.Unlock()
		watch, err := cacheService.EncodeWatch()
		if err != nil {
			return err
		}
		d.watchCh <- watch
	case utils.EventDelete:
		defer d.mu.Unlock()

		if cacheService, ok := d.cache[serviceId]; ok {
			entities := cacheService.entities
			delete(entities, entity)

			// When a service is not used, it needs to be unsubscribed
			if len(entities) == 0 {
				d.unsubscribe(cacheService)
				delete(d.cache, serviceId)
			}
		}
	}
	return nil
}

func (d *NacosDiscoverer) Update(update *comm.Update) error {
	values, oldArgs, newArgs, err := update.Decode()
	if err != nil {
		return err
	}

	event, service := values[0], values[1]
	log.Infof("Nacos update service %s", service)
	if event != utils.EventUpdate {
		log.Error("incorrect update event")
		return errors.New("incorrect update event")
	}
	serviceId := serviceID(service, oldArgs)
	newServiceId := serviceID(service, newArgs)

	d.mu.Lock()
	if cacheService, ok := d.cache[serviceId]; ok {
		d.unsubscribe(cacheService)
		d.mu.Unlock()

		nodes, err := d.fetch(service, newArgs)
		if err != nil {
			return err
		}
		cacheService.nodes = nodes
		cacheService.args = newArgs

		d.mu.Lock()
		delete(d.cache, serviceId)
		d.cache[newServiceId] = cacheService
		d.mu.Unlock()

		watch, err := cacheService.EncodeWatch()
		if err != nil {
			return err
		}
		d.watchCh <- watch
	}

	return nil
}

func (d *NacosDiscoverer) Watch() chan *comm.Watch {
	return d.watchCh
}

func (d *NacosDiscoverer) fetch(service string, args map[string]string) ([]Node, error) {
	// if the namespace client has not yet been created
	namespace := args["namespace_id"]
	if _, ok := d.namingClients[namespace]; !ok {
		err := d.newClient(namespace)
		if err != nil {
			return nil, err
		}
	}

	serviceId := serviceID(service, args)
	client := d.namingClients[namespace][d.hash(serviceId, namespace)]

	serviceInfo, err := client.GetService(vo.GetServiceParam{
		ServiceName: service,
		GroupName:   args["group_name"],
	})
	if err != nil {
		log.Errorf("Nacos get service[%s] error: %s", service, err)
		return nil, err
	}

	// watch the new service
	if err := d.subscribe(service, args, client); err != nil {
		log.Errorf("Nacos subscribe service[%s] error: %s", service, err)
		return nil, err
	}

	nodes := make([]Node, len(serviceInfo.Hosts))
	for i, host := range serviceInfo.Hosts {
		address := fmt.Sprintf("%s:%d", host.Ip, host.Port)
		weight := int(host.Weight)
		if weight == 0 {
			weight = d.weight
		}

		nodes[i] = Node{
			host:   address,
			weight: weight,
		}
	}

	return nodes, nil
}

func (d *NacosDiscoverer) watch(serviceId string) func([]model.SubscribeService, error) {
	watch := func(services []model.SubscribeService, err error) {
		nodes := make([]Node, len(services))
		for i, inst := range services {
			address := fmt.Sprintf("%s:%d", inst.Ip, inst.Port)
			weight := int(inst.Weight)
			if weight == 0 {
				weight = d.weight
			}

			nodes[i] = Node{
				host:   address,
				weight: weight,
			}
		}

		d.mu.Lock()
		cacheService := d.cache[serviceId]
		cacheService.nodes = nodes
		d.mu.Unlock()

		watch, _ := cacheService.EncodeWatch()
		d.watchCh <- watch
	}

	return watch
}

func (d *NacosDiscoverer) subscribe(service string, args map[string]string, client naming_client.INamingClient) error {
	log.Infof("Nacos subscribe service %s", service)
	serviceId := serviceID(service, args)
	param := &vo.SubscribeParam{
		ServiceName:       service,
		GroupName:         args["group_name"],
		SubscribeCallback: d.watch(serviceId),
	}

	// TODO: retry if failed to Subscribe
	err := client.Subscribe(param)
	if err == nil {
		d.mu.Lock()
		d.params[serviceId] = param
		d.mu.Unlock()
	}
	return err
}

func (d *NacosDiscoverer) unsubscribe(service *Service) {
	log.Infof("Nacos unsubscribe service %s", service.name)
	serviceId := serviceID(service.name, service.args)
	param := d.params[serviceId]

	namespace := service.args["namespace_id"]
	client := d.namingClients[namespace][d.hash(serviceId, namespace)]
	// the nacos unsubscribe function returns only nil
	// so ignore the error handling
	_ = client.Unsubscribe(param)
	delete(d.params, serviceId)
}

func (d *NacosDiscoverer) newClient(namespace string) error {
	newClients := make([]naming_client.INamingClient, 0, len(d.ServerConfigs))
	for auth, serverConfigs := range d.ServerConfigs {
		var username, password string
		if len(auth) != 0 {
			strs := strings.Split(auth, ":")
			if l := len(strs); l == 1 {
				username = strs[0]
			} else if l == 2 {
				username, password = strs[0], strs[1]
			} else {
				log.Error("incorrect auth information")
				return errors.New("incorrect auth information")
			}
		}

		clientConfig := constant.ClientConfig{
			TimeoutMs:           d.timeout,
			NamespaceId:         namespace,
			Username:            username,
			Password:            password,
			NotLoadCacheAtStart: true,
		}
		client, err := clients.NewNamingClient(
			vo.NacosClientParam{
				ClientConfig:  &clientConfig,
				ServerConfigs: serverConfigs,
			},
		)
		if err != nil {
			log.Errorf("Nacos new client error: %s", err)
			return err
		}
		newClients = append(newClients, client)
	}

	d.namingClients[namespace] = newClients
	log.Info("Successfully create a new Nacos client")
	return nil
}

// hash distributes the serviceId to different clients using CRC32
func (d *NacosDiscoverer) hash(serviceId, namespace string) int {
	d.crc.Reset()
	_, _ = d.crc.Write([]byte(serviceId))
	return int(d.crc.Sum32()) % len(d.namingClients[namespace])
}
