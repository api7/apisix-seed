package discoverer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/model"

	"github.com/api7/apisix-seed/internal/utils"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/vo"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
)

func init() {
	Discoveries["nacos"] = NewNacosDiscoverer
}

type NacosDiscoverer struct {
	weight int
	// nacos client default config
	ClientConfig constant.ClientConfig
	// nacos server configs
	ServerConfigs []constant.ServerConfig
	// nacos naming clients
	namingClients map[string]naming_client.INamingClient

	params map[string]*vo.SubscribeParam
	cache  map[string]Service
	mu     sync.Mutex

	watchCh chan *comm.Watch
}

func NewNacosDiscoverer(disConfig interface{}) (Discoverer, error) {
	config := disConfig.(*conf.Nacos)
	timeout := config.Timeout

	clientConfig := constant.ClientConfig{
		// compatible with past timeout configurations
		TimeoutMs:           uint64(timeout.Connect + timeout.Read + timeout.Send),
		NamespaceId:         config.Namespace,
		Username:            config.Username,
		Password:            config.Password,
		NotLoadCacheAtStart: true,
	}

	serverConfigs := make([]constant.ServerConfig, len(config.Host))
	for i, host := range config.Host {
		schemaEnd, ipEnd := strings.Index(host, "://"), strings.LastIndex(host, ":")

		port := 80 // default port
		if schemaEnd != ipEnd {
			port, _ = strconv.Atoi(host[ipEnd+1:])
		} else {
			ipEnd = len(host)
		}

		serverConfigs[i] = constant.ServerConfig{
			IpAddr:      host[schemaEnd+len("://") : ipEnd],
			Port:        uint64(port),
			Scheme:      host[:schemaEnd],
			ContextPath: config.Prefix,
		}
	}

	namingClients := make(map[string]naming_client.INamingClient, 1)
	defaultClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, err
	}
	namingClients[config.Namespace] = defaultClient

	discoverer := NacosDiscoverer{
		weight:        config.Weight,
		ClientConfig:  clientConfig,
		ServerConfigs: serverConfigs,
		namingClients: namingClients,
		params:        make(map[string]*vo.SubscribeParam),
		cache:         make(map[string]Service),
		mu:            sync.Mutex{},
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
	d.defaultNamespace(&args)

	event, entity, service := values[0], values[1], values[2]
	serviceId := fmt.Sprintf("%s@%s@%s", args["namespace"], args["group"], service)

	d.mu.Lock()
	switch event {
	case utils.EventAdd:
		ok := false
		var cacheService Service
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

			cacheService = Service{
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
	d.defaultNamespace(&oldArgs)
	d.defaultNamespace(&newArgs)

	event, service := values[0], values[1]
	if event != utils.EventUpdate {
		return errors.New("incorrect update event")
	}
	serviceId := fmt.Sprintf("%s@%s@%s", oldArgs["namespace"], oldArgs["group"], service)
	newServiceId := fmt.Sprintf("%s@%s@%s", newArgs["namespace"], newArgs["group"], service)

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
	namespace := args["namespace"]
	if _, ok := d.namingClients[namespace]; !ok {
		clientConfig := d.ClientConfig
		clientConfig.NamespaceId = namespace
		client, err := clients.NewNamingClient(
			vo.NacosClientParam{
				ClientConfig:  &clientConfig,
				ServerConfigs: d.ServerConfigs,
			},
		)
		if err != nil {
			return nil, err
		}
		d.namingClients[namespace] = client
	}

	client := d.namingClients[namespace]
	serviceInfo, err := client.GetService(vo.GetServiceParam{
		ServiceName: service,
		GroupName:   args["group"],
	})
	if err != nil {
		return nil, err
	}

	// watch the new service
	if err := d.subscribe(service, args); err != nil {
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
		d.cache[serviceId] = cacheService
		d.mu.Unlock()

		watch, _ := cacheService.EncodeWatch()
		d.watchCh <- watch
	}

	return watch
}

func (d *NacosDiscoverer) subscribe(service string, args map[string]string) error {
	serviceId := fmt.Sprintf("%s@%s@%s", args["namespace"], args["group"], service)
	param := &vo.SubscribeParam{
		ServiceName:       service,
		GroupName:         args["group"],
		SubscribeCallback: d.watch(serviceId),
	}

	client := d.namingClients[args["namespace"]]
	// TODO: retry if failed to Subscribe
	err := client.Subscribe(param)
	if err == nil {
		d.mu.Lock()
		d.params[serviceId] = param
		d.mu.Unlock()
	}
	return err
}

func (d *NacosDiscoverer) unsubscribe(service Service) {
	serviceId := fmt.Sprintf("%s@%s@%s", service.args["namespace"], service.args["group"], service.name)
	param := d.params[serviceId]

	client := d.namingClients[service.args["namespace"]]
	// the nacos unsubscribe function returns only nil
	// so ignore the error handling
	_ = client.Unsubscribe(param)
	delete(d.params, serviceId)
}

func (d *NacosDiscoverer) defaultNamespace(args *map[string]string) {
	// set default namespace
	if _, ok := (*args)["namespace"]; !ok {
		if (*args) == nil {
			*args = make(map[string]string, 1)
		}
		(*args)["namespace"] = d.ClientConfig.NamespaceId
	}
}
