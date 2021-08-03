/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package conf

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v3"
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
