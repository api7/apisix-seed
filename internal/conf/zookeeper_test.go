package conf

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func ZkReadConfig(t *testing.T) *Config {
	wd, _ := os.Getwd()
	dir := wd[:strings.Index(wd, "internal")]
	content, err := ioutil.ReadFile(dir + "conf/conf.yaml")
	assert.Nil(t, err)
	config := &Config{}
	err = yaml.Unmarshal(content, config)
	assert.Nil(t, err)
	return config
}

func TestZkBuilderConfig(t *testing.T) {
	config := ZkReadConfig(t)
	assert.NotNil(t, config)
	zkName := "zookeeper"

	for name, raw := range config.Discovery {
		if name != zkName {
			continue
		}
		zkConfBuilder := DisBuilders[name]
		zkConfBlock, err := yaml.Marshal(raw)
		assert.Nil(t, err)
		zkConf, err := zkConfBuilder(zkConfBlock)
		assert.Nil(t, err)
		assert.Equal(t, zkConf.(*Zookeeper).Hosts, []string{"127.0.0.1:2181"})
		assert.Equal(t, zkConf.(*Zookeeper).Prefix, "/zookeeper")
		assert.Equal(t, zkConf.(*Zookeeper).Weight, 100)
		assert.Equal(t, zkConf.(*Zookeeper).Timeout, 60)
	}
}
