package comm

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/api7/apisix-seed/internal/utils"
)

func TestWatch(t *testing.T) {
	tests := []struct {
		caseDesc     string
		giveValues   []string
		giveEntities []string
		giveNodes    map[string]float64
	}{
		{
			caseDesc:     "Test Watch Encode without nodes",
			giveValues:   []string{"update", "test"},
			giveEntities: []string{"upstream;1"},
			giveNodes:    nil,
		},
		{
			caseDesc:     "Test Watch Encode with nodes",
			giveValues:   []string{"update", "test"},
			giveEntities: []string{"service;1"},
			giveNodes: map[string]float64{
				"test.com:80": 10,
			},
		},
	}

	for _, tc := range tests {
		entityMsg := make(utils.Message, 0, len(tc.giveEntities))
		for _, entity := range tc.giveEntities {
			entityMsg.Add("entity", entity)
		}
		nodeMsg := make(utils.Message, 0, 2*len(tc.giveNodes))
		for host, weight := range tc.giveNodes {
			nodeMsg.Add("node", host)
			nodeMsg.Add("weight", strconv.Itoa(int(weight)))
		}

		watch, err := NewWatch(tc.giveValues, entityMsg, nodeMsg)
		assert.Nil(t, err, tc.caseDesc)

		values, entities, nodes, err := watch.Decode()
		assert.Nil(t, err, tc.caseDesc)
		assert.Equal(t, tc.giveValues, values, tc.caseDesc)
		assert.Equal(t, tc.giveEntities, entities, tc.caseDesc)
		assert.Equal(t, tc.giveNodes, nodes, tc.caseDesc)
	}
}
