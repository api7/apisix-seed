package discoverer

import (
	"testing"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/api7/apisix-seed/internal/utils"
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

func getNode(msg chan *comm.Message) string {
	m := <-msg
	_, _, nodes, _ := m.Decode()
	for node := range nodes {
		return node
	}
	return ""
}

func TestNewZkDiscoverer(t *testing.T) {
	config, err := getZkConfig()
	assert.Nil(t, err)
	assert.NotNil(t, config)
	dis, err := NewZookeeperDiscoverer(config)
	assert.Nil(t, err)
	assert.NotNil(t, dis)
}

func TestZkDiscoverer(t *testing.T) {
	config, err := getZkConfig()
	assert.Nil(t, err)
	assert.NotNil(t, config)

	var dis Discoverer
	dis, err = Discoveries["zookeeper"](config)
	assert.Nil(t, err)

	conn := dis.(*ZookeeperDiscoverer).zkConn
	svcName := "svc"
	svcPath := "/zookeeper/" + svcName
	// clear zookeeper service
	err = removeZkService(conn, svcPath)
	assert.Nil(t, err)

	headers := []string{utils.EventAdd, "/apisix/routes/1", svcName}
	query, err := comm.NewQuery(headers, map[string]string{})
	assert.Nil(t, err)

	err = dis.Query(&query)
	assert.NotNil(t, err)
	msg := dis.Watch()

	// create service
	err = createZkService(conn, svcPath, "{\"host\":\"127.0.0.1:1980\",\"weight\":100}")
	assert.Nil(t, err)
	assert.Equal(t, getNode(msg), "127.0.0.1:1980")

	// update service
	err = updateZkService(conn, svcPath, "{\"host\":\"127.0.0.1:1981\",\"weight\":100}")
	assert.Nil(t, err)
	assert.Equal(t, getNode(msg), "127.0.0.1:1981")

	// remove service
	err = removeZkService(conn, svcPath)
	assert.Nil(t, err)
	assert.Equal(t, getNode(msg), "")
}
