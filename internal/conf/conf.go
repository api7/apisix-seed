package conf

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
)

type DisBuilder func([]byte) (interface{}, error)

var (
	WorkDir     = "."
	ETCDConfig  *Etcd
	DisConfigs  = make(map[string]interface{})
	DisBuilders = make(map[string]DisBuilder)
)

type TLS struct {
	CertFile string `yaml:"cert"`
	KeyFile  string `yaml:"key"`
	Verify   bool
}

type Etcd struct {
	Host     []string
	Prefix   string
	Timeout  int
	User     string
	Password string
	TLS      *TLS
}

type Config struct {
	Etcd      Etcd
	Discovery map[string]interface{}
}

func InitConf() {
	if workDir := os.Getenv("APISIX_SEED_WORKDIR"); workDir != "" {
		WorkDir = workDir
	}

	filePath := WorkDir + "/conf/conf.yaml"
	if configurationContent, err := ioutil.ReadFile(filePath); err != nil {
		panic(fmt.Sprintf("fail to read configuration: %s", filePath))
	} else {
		config := Config{}
		err := yaml.Unmarshal(configurationContent, &config)
		if err != nil {
			log.Printf("conf: %s, error: %v", configurationContent, err)
		}

		for name, rawConfig := range config.Discovery {
			builder, ok := DisBuilders[name]
			if !ok {
				panic(fmt.Sprintf("unkown discovery configuration: %s", name))
			}

			rawStr, _ := yaml.Marshal(rawConfig)
			disConfig, err := builder(rawStr)
			if err != nil {
				panic(fmt.Sprintf("fail to load discovery configuration: %s", err))
			}

			DisConfigs[name] = disConfig
		}

		if len(config.Etcd.Host) > 0 {
			initEtcdConfig(config.Etcd)
		}
	}
}

// initialize etcd config
func initEtcdConfig(conf Etcd) {
	var host = []string{"127.0.0.1:2379"}
	if len(conf.Host) > 0 {
		host = conf.Host
	}

	prefix := "/apisix"
	if len(conf.Prefix) > 0 {
		prefix = conf.Prefix
	}

	ETCDConfig = &Etcd{
		Host:     host,
		User:     conf.User,
		Password: conf.Password,
		TLS:      conf.TLS,
		Prefix:   prefix,
	}
}
