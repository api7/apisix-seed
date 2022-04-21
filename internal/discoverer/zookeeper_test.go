package discoverer

import (
	"testing"

	"github.com/api7/apisix-seed/internal/core/message"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

var zkYamlConfig = `
hosts:
  - "127.0.0.1:2181"
prefix: /zookeeper
weight: 100
timeout: 10
`

func getZkConfig() (*conf.Zookeeper, error) {
	zkConf := &conf.Zookeeper{}
	err := yaml.Unmarshal([]byte(zkYamlConfig), zkConf)
	if err != nil {
		return nil, err
	}
	return zkConf, nil
}

func updateZkService(conn *zk.Conn, svcPath string, svcNode string) error {
	_, stat, err := conn.Get(svcPath)
	if err != nil {
		return err
	}
	_, err = conn.Set(svcPath, []byte(svcNode), stat.Version)
	return err
}

func createZkService(conn *zk.Conn, svcPath string, svcNode string) error {
	_, err := conn.Create(svcPath, []byte(svcNode), 0, zk.WorldACL(zk.PermAll))
	return err
}

func removeZkService(conn *zk.Conn, svcPath string) error {
	_, stat, err := conn.Get(svcPath)
	if err != nil {
		// Does not exist and returns nil
		return nil
	}
	err = conn.Delete(svcPath, stat.Version)
	return err
}

func zkMsg2Value(msg *message.Message) string {
	str, _ := msg.Marshal()
	return string(str)
}

func newZkDiscoverer() (*ZookeeperDiscoverer, error) {
	config, err := getZkConfig()
	if err != nil {
		return nil, err
	}

	dis, err := NewZookeeperDiscoverer(config)

	return dis.(*ZookeeperDiscoverer), err
}

func TestNewZookeeperDiscoverer(t *testing.T) {
	dis, err := newZkDiscoverer()
	assert.Nil(t, err)
	assert.NotNil(t, dis)
}

func TestZookeeperDiscoverer(t *testing.T) {
	dis, err := newZkDiscoverer()
	assert.Nil(t, err)
	assert.NotNil(t, dis)

	conn := dis.zkConn
	svcName := "svc"
	svcPath := "/zookeeper/" + svcName
	// clear zookeeper service
	err = removeZkService(conn, svcPath)
	assert.Nil(t, err)

	key := "/apisix/routes/1"
	value := `{"uri":"/hh","upstream":{"discovery_type":"zookeeper","service_name":"svc"}}`
	msg, err := message.NewMessage(key, []byte(value), 1, message.EventAdd)
	assert.Nil(t, err)

	err = dis.Query(msg)
	assert.NotNil(t, err)
	msgChan := dis.Watch()

	// create service
	err = createZkService(conn, svcPath, `{"host":"127.0.0.1","port":1980,"weight":100}`)
	assert.Nil(t, err)
	newMsg := <-msgChan
	expectValue := `{"uri":"/hh","upstream":{"_discovery_type":"zookeeper","_service_name":"svc","nodes":[{"host":"127.0.0.1","port":1980,"weight":100}]}}`
	assert.JSONEq(t, expectValue, zkMsg2Value(newMsg))

	// update service
	err = updateZkService(conn, svcPath, `{"host":"127.0.0.1","port":1981,"weight":100}`)
	assert.Nil(t, err)
	newMsg = <-msgChan
	expectValue = `{"uri":"/hh","upstream":{"_discovery_type":"zookeeper","_service_name":"svc","nodes":[{"host":"127.0.0.1","port":1981,"weight":100}]}}`
	assert.JSONEq(t, expectValue, zkMsg2Value(newMsg))

	// remove service
	err = removeZkService(conn, svcPath)
	assert.Nil(t, err)
	newMsg = <-msgChan
	expectValue = `{"uri":"/hh","upstream":{"_discovery_type":"zookeeper","_service_name":"svc","nodes":[]}}`
	assert.JSONEq(t, expectValue, zkMsg2Value(newMsg))
}
