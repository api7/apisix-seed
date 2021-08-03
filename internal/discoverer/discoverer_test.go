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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceEncodeWatch(t *testing.T) {
	tests := []struct {
		caseDesc    string
		giveService Service
		wantMsg     string
	}{
		{
			caseDesc: "Test Watch Decode",
			giveService: Service{
				name: "test",
				nodes: []Node{
					{host: "127.0.0.1:80", weight: 10},
					{host: "127.0.0.1:8080", weight: 20},
				},
				entities: []string{
					"upstream;1",
					"upstream;2",
				},
				args: nil,
			},
			wantMsg: `key: event, value: update
key: service, value: test
key: entity, value: upstream;1
key: entity, value: upstream;2
key: node, value: 127.0.0.1:80
key: weight, value: 10
key: node, value: 127.0.0.1:8080
key: weight, value: 20`,
		},
	}

	for _, tc := range tests {
		watch, err := tc.giveService.EncodeWatch()
		assert.Nil(t, err)
		assert.True(t, watch.String() == tc.wantMsg, tc.caseDesc)
	}
}
