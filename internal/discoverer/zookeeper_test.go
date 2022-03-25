package discoverer

import (
	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/api7/apisix-seed/internal/utils"
	"github.com/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
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

func InitZkConn(t *testing.T) {
	if zkConn == nil {
		config := GetZkConfig(t)
		conn, _, err := zk.Connect(config.Hosts, time.Second*time.Duration(config.Timeout))
		assert.Nil(t, err)
		assert.NotNil(t, conn)
		zkConn = conn
	}
}

func UpdateZkService(t *testing.T) {
	InitZkConn(t)
	_, err := zkConn.Set(zkPrefix+"/"+zkService, []byte(zkNode2), 0)
	assert.Nil(t, err)
}

func CreateZkService(t *testing.T) {
	InitZkConn(t)
	_, err := zkConn.Create(zkPrefix+"/"+zkService, []byte(zkNode1), 0, zk.WorldACL(zk.PermAll))
	assert.Nil(t, err)
}

func RemoveZkService(t *testing.T) {
	InitZkConn(t)
	_ = zkConn.Delete(zkPrefix+"/"+zkService, 0)
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
	ZkDiscovererRegister(t, dis)
	ZkDiscovererUpdate(t, dis)
}

func ZkDiscovererRegister(t *testing.T, dis interface{}) {
	msg := <-dis.(*ZookeeperDiscoverer).Watch()
	_, _, nodes, _ := msg.Decode()
	for node := range nodes {
		assert.Equal(t, node, "127.0.0.1:1980")
		break
	}
}

func ZkDiscovererUpdate(t *testing.T, dis interface{}) {
	UpdateZkService(t)
	msg := <-dis.(*ZookeeperDiscoverer).Watch()
	_, _, nodes, _ := msg.Decode()
	for node := range nodes {
		assert.Equal(t, node, "127.0.0.1:1981")
		break
	}
}
