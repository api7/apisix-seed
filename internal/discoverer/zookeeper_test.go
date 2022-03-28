package discoverer

import (
	"io/ioutil"
	"os"
	"strings"
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

func GetZkConfig(t *testing.T) *conf.Zookeeper {
	wd, _ := os.Getwd()
	dir := wd[:strings.Index(wd, "internal")]
	content, err := ioutil.ReadFile(dir + "conf/conf.yaml")
	assert.Nil(t, err)
	config := &conf.Config{}
	err = yaml.Unmarshal(content, config)
	assert.Nil(t, err)
	zkConf := &conf.Zookeeper{}
	for n, c := range config.Discovery {
		if n == zkName {
			raw, err := yaml.Marshal(c)
			assert.Nil(t, err)
			err = yaml.Unmarshal(raw, &zkConf)
			assert.Nil(t, err)
			break
		}
	}
	return zkConf
}

func InitZkConn(t *testing.T) *zk.Conn {
	if zkConn == nil {
		config := GetZkConfig(t)
		conn, _, err := zk.Connect(config.Hosts, time.Second*time.Duration(config.Timeout))
		assert.Nil(t, err)
		assert.NotNil(t, conn)
		zkConn = conn
	}
	return zkConn
}

func UpdateZkService(t *testing.T) {
	conn := InitZkConn(t)
	_, err := conn.Set(zkPrefix+"/"+zkService, []byte(zkNode2), 0)
	assert.Nil(t, err)
}

func CreateZkService(t *testing.T) {
	conn := InitZkConn(t)
	_, err := conn.Create(zkPrefix+"/"+zkService, []byte(zkNode1), 0, zk.WorldACL(zk.PermAll))
	assert.Nil(t, err)
}

func RemoveZkService(t *testing.T) {
	conn := InitZkConn(t)
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
	RemoveZkService(t)
	CreateZkService(t)
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
	UpdateZkService(t)
	m := <-msg
	_, _, nodes, _ := m.Decode()
	for node := range nodes {
		assert.Equal(t, node, "127.0.0.1:1981")
		return
	}
}

func ZkDiscovererRemove(t *testing.T, msg chan *comm.Message) {
	RemoveZkService(t)
	m := <-msg
	_, _, nodes, _ := m.Decode()
	assert.Nil(t, nodes)
}
