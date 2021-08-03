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

package discoverer

import (
	"strconv"

	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/api7/apisix-seed/internal/utils"
)

var (
	Discoveries = make(map[string]Discover)
)

type Discover func(disConfig interface{}) Discoverer

// Discoverer defines the component that interact nacos, consul and so on
type Discoverer interface {
	Stop()
	Query(*comm.Query) error
	Watch() chan *comm.Watch
}

// Node defines the upstream machine information
type Node struct {
	host   string
	weight int
}

// Service defines the service information for discoverer
type Service struct {
	name     string
	nodes    []Node   // nodes are the upstream machines of the service
	entities []string // entities are the upstreams/services/routes that use the service
	args     map[string]string
}

func (s *Service) EncodeNodes() utils.Message {
	msg := make(utils.Message, 0, 2*len(s.nodes))
	for _, node := range s.nodes {
		msg.Add("node", node.host)
		msg.Add("weight", strconv.Itoa(node.weight))
	}
	return msg
}

func (s *Service) EncodeEntities() utils.Message {
	msg := make(utils.Message, 0, len(s.entities))
	for _, entity := range s.entities {
		msg.Add("entity", entity)
	}
	return msg
}

// EncodeWatch encodes a service to the watch message
func (s *Service) EncodeWatch() (*comm.Watch, error) {
	header, err := comm.NewWatchHeader([]string{utils.EventUpdate, s.name})
	if err != nil {
		return nil, err
	}
	entities := s.EncodeEntities()
	nodes := s.EncodeNodes()

	watch := comm.NewWatch(header, entities, nodes)
	return &watch, nil
}
