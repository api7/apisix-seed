package conf

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var zkYamlConfig = `
hosts:
  - "127.0.0.1:2181"
prefix: /zookeeper
weight: 100
timeout: 10
`

func TestZkBuilderConfig(t *testing.T) {
	zkConfBuilder, ok := DisBuilders["zookeeper"]
	assert.Equal(t, true, ok)
	zkConf, err := zkConfBuilder([]byte(zkYamlConfig))
	assert.Nil(t, err)
	assert.Equal(t, zkConf.(*Zookeeper).Hosts, []string{"127.0.0.1:2181"})
	assert.Equal(t, zkConf.(*Zookeeper).Prefix, "/zookeeper")
	assert.Equal(t, zkConf.(*Zookeeper).Weight, 100)
	assert.Equal(t, zkConf.(*Zookeeper).Timeout, 10)
}
