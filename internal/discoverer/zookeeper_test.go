package discoverer

import (
	"testing"
	"time"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/api7/apisix-seed/internal/utils"
	"github.com/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var zkName = "zookeeper"
var zkPrefix = "/zookeeper"
var zkService = "svc"
var zkNode1 = "{\"host\":\"127.0.0.1:1980\",\"weight\":100}"
var zkNode2 = "{\"host\":\"127.0.0.1:1981\",\"weight\":100}"
var zkConn *zk.Conn
var zkYamlConfig = `
hosts:
  - "127.0.0.1:2181"
prefix: /zookeeper
weight: 100
timeout: 10
`

func GetZkConfig(t *testing.T) *conf.Zookeeper {
	zkConf := &conf.Zookeeper{}
	err := yaml.Unmarshal([]byte(zkYamlConfig), zkConf)
	assert.Nil(t, err)

	return zkConf
}

func initZkConn(t *testing.T) *zk.Conn {
	if zkConn == nil {
		config := GetZkConfig(t)
		conn, _, err := zk.Connect(config.Hosts, time.Second*time.Duration(config.Timeout))
		assert.Nil(t, err)
		assert.NotNil(t, conn)
		zkConn = conn
	}
	return zkConn
}

func updateZkService(t *testing.T) {
	conn := initZkConn(t)
	_, err := conn.Set(zkPrefix+"/"+zkService, []byte(zkNode2), 0)
	assert.Nil(t, err)
}

func createZkService(t *testing.T) {
	conn := initZkConn(t)
	_, err := conn.Create(zkPrefix+"/"+zkService, []byte(zkNode1), 0, zk.WorldACL(zk.PermAll))
	assert.Nil(t, err)
}

func removeZkService(t *testing.T) {
	conn := initZkConn(t)
	svcPath := zkPrefix + "/" + zkService
	data, _, err := conn.Get(svcPath)
	if len(data) == 0 && err != nil {
		return
	}
	err = conn.Delete(zkPrefix+"/"+zkService, 1)
	assert.Nil(t, err)
}

func TestNewZkDiscoverer(t *testing.T) {
	config := GetZkConfig(t)
	assert.NotNil(t, config)
	dis, err := NewZookeeperDiscoverer(config)
	assert.Nil(t, err)
	assert.NotNil(t, dis)
}

func TestZkDiscoverer(t *testing.T) {
	removeZkService(t)
	createZkService(t)
	config := GetZkConfig(t)
	assert.NotNil(t, config)
	dis, err := Discoveries[zkName](config)
	assert.Nil(t, err)
	headers := []string{utils.EventAdd, "/apisix/routes/1", zkService}
	query, err := comm.NewQuery(headers, map[string]string{})
	assert.Nil(t, err)
	err = dis.Query(&query)
	assert.Nil(t, err)
	msg := dis.Watch()
	ZkDiscovererRegister(t, msg)
	ZkDiscovererUpdate(t, msg)
	ZkDiscovererRemove(t, msg)
}

func ZkDiscovererRegister(t *testing.T, msg chan *comm.Message) {
	m := <-msg
	_, _, nodes, _ := m.Decode()
	for node := range nodes {
		assert.Equal(t, node, "127.0.0.1:1980")
		return
	}
}

func ZkDiscovererUpdate(t *testing.T, msg chan *comm.Message) {
	updateZkService(t)
	m := <-msg
	_, _, nodes, _ := m.Decode()
	for node := range nodes {
		assert.Equal(t, node, "127.0.0.1:1981")
		return
	}
}

func ZkDiscovererRemove(t *testing.T, msg chan *comm.Message) {
	removeZkService(t)
	m := <-msg
	_, _, nodes, _ := m.Decode()
	assert.Nil(t, nodes)
}
