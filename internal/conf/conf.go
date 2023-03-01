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
	ETCDConfigs []*Etcd = make([]*Etcd, 0)
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
	RotationTime time.Duration `yaml:"rotation_time"`
}

type Config struct {
	Etcd      []Etcd
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
		for _, singleEtcd := range config.Etcd {
			if len(singleEtcd.Host) > 0 {
				initEtcdConfig(singleEtcd)
			}
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

	ETCDConfigs = append(ETCDConfigs, &Etcd{
		Host:     host,
		User:     conf.User,
		Password: conf.Password,
		TLS:      conf.TLS,
		Prefix:   prefix,
	})
}

func initLogConfig(conf Log) {
	level := conf.Level
	if level == "" {
		level = "warn"
	}
	if conf.Path == "" {
		LogConfig = &Log{
			Level: level,
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
	rotationTime := conf.RotationTime
	if rotationTime == 0 {
		rotationTime = time.Hour
	}
	LogConfig = &Log{
		Level:        level,
		Path:         conf.Path,
		MaxAge:       maxAge,
		MaxSize:      maxSize,
		RotationTime: rotationTime,
	}
}
