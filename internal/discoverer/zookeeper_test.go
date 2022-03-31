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

func updateZkService(t *testing.T, conn *zk.Conn) {
	svcPath := zkPrefix + "/" + zkService
	_, stat, err := conn.Get(svcPath)
	assert.Nil(t, err)
	_, err = conn.Set(svcPath, []byte(zkNode2), stat.Version)
	assert.Nil(t, err)
}

func createZkService(t *testing.T, conn *zk.Conn) {
	_, err := conn.Create(zkPrefix+"/"+zkService, []byte(zkNode1), 0, zk.WorldACL(zk.PermAll))
	assert.Nil(t, err)
}

func removeZkService(t *testing.T, conn *zk.Conn) {
	svcPath := zkPrefix + "/" + zkService
	_, stat, err := conn.Get(svcPath)
	if err != nil {
		return
	}
	err = conn.Delete(zkPrefix+"/"+zkService, stat.Version)
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
	config := GetZkConfig(t)
	assert.NotNil(t, config)
	dis, err := Discoveries[zkName](config)
	conn := dis.(*ZookeeperDiscoverer).zkConn
	removeZkService(t, conn)
	assert.Nil(t, err)
	headers := []string{utils.EventAdd, "/apisix/routes/1", zkService}
	query, err := comm.NewQuery(headers, map[string]string{})
	assert.Nil(t, err)
	err = dis.Query(&query)
	time.Sleep(time.Second * time.Duration(2))
	assert.NotNil(t, err)
	msg := dis.Watch()
	createZkService(t, conn)
	zkDiscovererRegister(t, msg)
	time.Sleep(time.Second * time.Duration(2))
	updateZkService(t, conn)
	zkDiscovererUpdate(t, msg)
	time.Sleep(time.Second * time.Duration(2))
	removeZkService(t, conn)
	zkDiscovererRemove(t, msg)
}

func zkDiscovererRegister(t *testing.T, msg chan *comm.Message) {
	m := <-msg
	_, _, nodes, _ := m.Decode()
	for node := range nodes {
		assert.Equal(t, node, "127.0.0.1:1980")
		return
	}
}

func zkDiscovererUpdate(t *testing.T, msg chan *comm.Message) {
	m := <-msg
	_, _, nodes, _ := m.Decode()
	for node := range nodes {
		assert.Equal(t, node, "127.0.0.1:1981")
		return
	}
}

func zkDiscovererRemove(t *testing.T, msg chan *comm.Message) {
	m := <-msg
	_, _, nodes, _ := m.Decode()
	assert.Nil(t, nodes)
}
