package discoverer

import (
	"fmt"
	"hash"
	"hash/crc32"
	"net/url"
	"reflect"
	"strconv"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/common/logger"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/message"
	"github.com/api7/gopkg/pkg/log"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func init() {
	Discoveries["nacos"] = NewNacosDiscoverer
}

func serviceID(service string, args map[string]interface{}) string {
	id, group := "", ""
	if args != nil {
		id, _ = args["namespace_id"].(string)
		group, _ = args["group_name"].(string)
	}
	serviceId := fmt.Sprintf("%s@%s@%s", id, group, service)
	return serviceId
}

type NacosService struct {
	id     string
	name   string
	args   map[string]interface{}
	nodes  []*message.Node             // nodes are the upstream machines of the service
	a6Conf map[string]*message.Message // entities are the upstreams/services/routes that use the service
}

type NacosDiscoverer struct {
	timeout uint64
	weight  int
	// nacos server configs, grouping by authentication information
	ServerConfigs map[string][]constant.ServerConfig
	// nacos naming clients, grouping by authentication information
	namingClients map[string][]naming_client.INamingClient
	// nacos username
	serverUser string
	// nacos password
	serverPassword string

	paramMutex sync.Mutex
	params     map[string]*vo.SubscribeParam
	cacheMutex sync.Mutex
	cache      map[string]*NacosService

	crc hash.Hash32

	msgCh chan *message.Message
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

		auth := config.User
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
		timeout:        uint64(timeout.Connect + timeout.Read + timeout.Send),
		weight:         config.Weight,
		ServerConfigs:  serverConfigs,
		serverUser:     config.User,
		serverPassword: config.Password,
		namingClients:  make(map[string][]naming_client.INamingClient),
		paramMutex:     sync.Mutex{},
		params:         make(map[string]*vo.SubscribeParam),
		cacheMutex:     sync.Mutex{},
		cache:          make(map[string]*NacosService),
		crc:            crc32.NewIEEE(),
		msgCh:          make(chan *message.Message, 10),
	}
	logger.SetLogger(log.DefaultLogger)
	return &discoverer, nil
}

func (d *NacosDiscoverer) Stop() {
	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()

	close(d.msgCh)

	// Unsubscribe all services
	for _, service := range d.cache {
		d.unsubscribe(service)
	}
}

func (d *NacosDiscoverer) Query(msg *message.Message) error {
	serviceId := serviceID(msg.ServiceName(), msg.DiscoveryArgs())
	println("Nacos: ", msg.ServiceName(), msg.DiscoveryArgs())
	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()

	if discover, ok := d.cache[serviceId]; ok {
		// cache information is already available
		msg.InjectNodes(discover.nodes)
		discover.a6Conf[msg.Key] = msg
	} else {
		// fetch new service information
		dis := &NacosService{
			id:   serviceId,
			name: msg.ServiceName(),
			args: msg.DiscoveryArgs(),
		}
		nodes, err := d.fetch(dis)
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			log.Warnf("No nodes found for service[%s]", serviceId)
			return fmt.Errorf("no nodes found for service[%s]", serviceId)
		}

		msg.InjectNodes(nodes)

		dis.nodes = nodes
		dis.a6Conf = map[string]*message.Message{
			msg.Key: msg,
		}

		d.cache[serviceId] = dis
	}
	d.msgCh <- msg

	return nil
}

func (d *NacosDiscoverer) Delete(msg *message.Message) error {
	serviceId := serviceID(msg.ServiceName(), msg.DiscoveryArgs())

	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()

	if discover, ok := d.cache[serviceId]; ok {
		delete(discover.a6Conf, msg.Key)

		// When a service is not used, it needs to be unsubscribed
		if len(discover.a6Conf) == 0 {
			d.unsubscribe(discover)
			delete(d.cache, serviceId)
		}
	}
	return nil
}

func (d *NacosDiscoverer) Update(oldMsg, msg *message.Message) error {
	msgArgs := oldMsg.DiscoveryArgs()
	newMsgArgs := msg.DiscoveryArgs()
	serviceId := serviceID(oldMsg.ServiceName(), msgArgs)
	newServiceId := serviceID(msg.ServiceName(), newMsgArgs)

	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()
	if discover, ok := d.cache[serviceId]; ok {
		if serviceId == newServiceId && reflect.DeepEqual(msgArgs["metadata"], newMsgArgs["metadata"]) {
			discover.a6Conf[msg.Key].Version = msg.Version
			return nil
		}

		d.unsubscribe(discover)

		newDiscover := &NacosService{
			args: msg.DiscoveryArgs(),
			id:   newServiceId,
			name: msg.ServiceName(),
		}
		nodes, err := d.fetch(newDiscover)
		if err != nil {
			return err
		}
		if len(nodes) == 0 {
			log.Warnf("No nodes found for service[%s]", serviceId)
			return fmt.Errorf("no nodes found for service[%s]", serviceId)
		}
		msg.InjectNodes(nodes)
		newDiscover.nodes = nodes
		newDiscover.a6Conf = map[string]*message.Message{
			msg.Key: msg,
		}

		delete(d.cache, serviceId)
		d.cache[newServiceId] = newDiscover

		d.msgCh <- msg
	}

	return nil
}

func (d *NacosDiscoverer) Watch() chan *message.Message {
	return d.msgCh
}

func (d *NacosDiscoverer) fetch(service *NacosService) ([]*message.Node, error) {
	// if the namespace client has not yet been created
	namespace, _ := service.args["namespace_id"].(string)
	if _, ok := d.namingClients[namespace]; !ok {
		err := d.newClient(namespace)
		if err != nil {
			return nil, err
		}
	}

	client := d.namingClients[namespace][d.hash(service.id, namespace)]

	groupName, _ := service.args["group_name"].(string)
	serviceInfo, err := client.GetService(vo.GetServiceParam{
		ServiceName: service.name,
		GroupName:   groupName,
	})
	if err != nil {
		log.Errorf("Nacos get service[%s] error: %s", service.name, err)
		return nil, err
	}

	// watch the new service
	if err = d.subscribe(service, client); err != nil {
		log.Errorf("Nacos subscribe service[%s] error: %s", service.name, err)
		return nil, err
	}

	// metadata
	metadata := service.args["metadata"]
	nodes := make([]*message.Node, 0)
	for _, host := range serviceInfo.Hosts {
		if metadata != nil {
			discard := 0
			for k, v := range metadata.(map[string]interface{}) {
				if host.Metadata[k] != v {
					discard = 1
				}
			}
			if discard == 1 {
				continue
			}
		}

		weight := int(host.Weight)
		if weight == 0 {
			weight = d.weight
		}

		nodes = append(nodes, &message.Node{
			Host:   host.Ip,
			Port:   int(host.Port),
			Weight: weight,
		})
	}

	return nodes, nil
}

func (d *NacosDiscoverer) newSubscribeCallback(serviceId string, metadata interface{}) func([]model.SubscribeService, error) {
	return func(services []model.SubscribeService, err error) {
		nodes := make([]*message.Node, 0)
		meta, ok := metadata.(map[string]interface{})

		for _, inst := range services {
			if ok {
				discard := 0
				for k, v := range meta {
					if inst.Metadata[k] != v {
						discard = 1
					}
				}
				if discard == 1 {
					continue
				}
			}

			weight := int(inst.Weight)
			if weight == 0 {
				weight = d.weight
			}

			nodes = append(nodes, &message.Node{
				Host:   inst.Ip,
				Port:   int(inst.Port),
				Weight: weight,
			})
		}

		if len(nodes) == 0 {
			log.Warnf("No valid nodes found for service[%s]", serviceId)
			return
		}

		d.cacheMutex.Lock()
		defer d.cacheMutex.Unlock()

		discover := d.cache[serviceId]
		discover.nodes = nodes

		for _, msg := range discover.a6Conf {
			msg.InjectNodes(nodes)
			d.msgCh <- msg
		}
	}
}

func (d *NacosDiscoverer) subscribe(service *NacosService, client naming_client.INamingClient) error {
	groupName, _ := service.args["group_name"].(string)
	log.Infof("Nacos subscribe service: %s, groupName: %s", service.name, groupName)

	param := &vo.SubscribeParam{
		ServiceName:       service.name,
		GroupName:         groupName,
		SubscribeCallback: d.newSubscribeCallback(service.id, service.args["metadata"]),
	}

	// TODO: retry if failed to Subscribe
	err := client.Subscribe(param)
	if err == nil {
		d.paramMutex.Lock()
		d.params[service.id] = param
		d.paramMutex.Unlock()
	}
	return err
}

func (d *NacosDiscoverer) unsubscribe(service *NacosService) {
	log.Infof("Nacos unsubscribe service %s", service.name)
	param := d.params[service.id]

	namespace, _ := service.args["namespace_id"].(string)
	client := d.namingClients[namespace][d.hash(service.id, namespace)]

	// the nacos unsubscribe function returns only nil
	// so ignore the error handling
	_ = client.Unsubscribe(param)
	delete(d.params, service.id)
}

func (d *NacosDiscoverer) newClient(namespace string) error {
	newClients := make([]naming_client.INamingClient, 0, len(d.ServerConfigs))
	username := d.serverUser
	password := d.serverPassword
	for _, serverConfigs := range d.ServerConfigs {

		clientConfig := constant.ClientConfig{
			TimeoutMs:            d.timeout,
			NamespaceId:          namespace,
			Username:             username,
			Password:             password,
			NotLoadCacheAtStart:  true,
			UpdateCacheWhenEmpty: true,
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
	// logger.InitLogger will be called, we should SetLogger again
	logger.SetLogger(log.DefaultLogger)
	log.Info("Successfully create a new Nacos client")
	return nil
}

// hash distributes the serviceId to different clients using CRC32
func (d *NacosDiscoverer) hash(serviceId, namespace string) int {
	d.crc.Reset()
	_, _ = d.crc.Write([]byte(serviceId))
	return int(d.crc.Sum32()) % len(d.namingClients[namespace])
}
