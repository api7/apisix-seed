package conf

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type DisBuilder func([]byte) (interface{}, error)

var (
	WorkDir     = "."
	ETCDConfig  *Etcd
	LogConfig   *Log
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

type Log struct {
	Level        string
	Path         string
	MaxAge       time.Duration
	MaxSize      int64
	RotationTime time.Duration `yaml:"roation_time"`
}

type Config struct {
	Etcd      Etcd
	Log       Log
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

		initLogConfig(config.Log)

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

func initLogConfig(conf Log) {
	if conf.Path == "" {
		LogConfig = &Log{
			Level: conf.Level,
		}
		return
	}
	maxAge := conf.MaxAge
	if maxAge == 0 {
		maxAge = 7 * 24 * time.Hour
	}
	maxSize := conf.MaxSize
	if maxSize == 0 {
		maxSize = 100 * 1024 * 1024
	}
	roationTime := conf.RotationTime
	if roationTime == 0 {
		roationTime = time.Hour
	}
	LogConfig = &Log{
		Level:        conf.Level,
		Path:         conf.Path,
		MaxAge:       maxAge,
		MaxSize:      maxSize,
		RotationTime: roationTime,
	}
}
