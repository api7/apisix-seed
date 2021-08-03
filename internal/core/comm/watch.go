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

package comm

import (
	"errors"

	"github.com/api7/apisix-seed/internal/utils"
	"go.uber.org/zap/buffer"
)

var watchHeader = [2]string{"event", "service"}

type Watch struct {
	header   utils.Message
	entities utils.Message
	nodes    utils.Message
}

func NewWatchHeader(values []string) (utils.Message, error) {
	if len(values) != len(watchHeader) {
		return nil, errors.New("incorrect watch header values")
	}

	msg := make(utils.Message, 0, len(watchHeader))
	for idx, key := range watchHeader {
		msg.Add(key, values[idx])
	}
	return msg, nil
}

func NewWatch(header, entities, nodes utils.Message) Watch {
	return Watch{
		header:   header,
		entities: entities,
		nodes:    nodes,
	}
}

func (msg *Watch) String() string {
	msgString := buffer.Buffer{}

	msgs := []utils.Message{msg.header, msg.entities, msg.nodes}
	for i, msg := range msgs {
		str := msg.String()
		if i != 0 {
			msgString.AppendString("\n")
		}
		msgString.AppendString(str)
	}
	return msgString.String()
}
