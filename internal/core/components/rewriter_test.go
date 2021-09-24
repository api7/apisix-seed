package components

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/api7/apisix-seed/internal/core/entity"
	"github.com/api7/apisix-seed/internal/core/storer"
	"github.com/api7/apisix-seed/internal/discoverer"
	"github.com/api7/apisix-seed/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type TestNodes struct {
	entity.BaseInfo
	Nodes []*entity.Node
}

func (t *TestNodes) SetNodes(nodes []*entity.Node) {
	t.Nodes = nodes
}

func init() {
	// Init Mock Discoverer
	discoverer.Discoveries = map[string]discoverer.Discover{
		"mock": discoverer.NewDiscovererMock,
	}
	_ = discoverer.InitDiscoverer("mock", nil)
}

func TestRewriter(t *testing.T) {
	caseDesc := "sanity"
	giveCache := map[string]interface{}{
		"test/test1": &TestNodes{},
	}
	giveKey := "test1"
	giveEntities := make(utils.Message, 0, 1)
	giveEntities.Add("entity", "mock;test1")
	giveNodes := make(utils.Message, 0, 2)
	giveNodes.Add("node", "test.com:80")
	giveNodes.Add("weight", "10")

	wantNodes := []*entity.Node{
		{
			Host:   "test.com",
			Port:   80,
			Weight: 10,
		},
	}

	headerVals := []string{utils.EventUpdate, giveKey}
	watch, err := comm.NewWatch(headerVals, giveEntities, giveNodes)
	assert.Nil(t, err, caseDesc)

	watchCh := make(chan *comm.Watch)
	discover := discoverer.GetDiscoverer("mock")
	mDiscover := discover.(interface{}).(*discoverer.MockInterface)
	mDiscover.On("Watch").Run(func(args mock.Arguments) {}).Return(watchCh)

	rewriter := Rewriter{}
	rewriter.Init()

	doneCh := make(chan struct{})

	mStg := &storer.MockInterface{}
	mStg.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		assert.Equal(t, giveKey, args[0], caseDesc)
		input := TestNodes{}
		err := json.Unmarshal([]byte(args[1].(string)), &input)
		assert.Nil(t, err)
		assert.Equal(t, len(wantNodes), len(input.Nodes), caseDesc)
		for i := range wantNodes {
			assert.Equal(t, *wantNodes[i], *input.Nodes[i], caseDesc)
		}
		assert.NotEqual(t, 0, input.UpdateTime, caseDesc)
		doneCh <- struct{}{}
	}).Return(nil)

	err = storer.InitStore("mock", storer.GenericStoreOption{
		BasePath: "test",
		ObjType:  reflect.TypeOf(TestNodes{}),
	}, mStg)
	assert.Nil(t, err, caseDesc)

	store := storer.GetStore("mock")
	for k, v := range giveCache {
		store.Store(k, v)
	}

	watchCh <- &watch
	<-doneCh

	obj, ok := store.Store("test/"+giveKey, nil)
	assert.True(t, ok, caseDesc)
	objTn, ok := obj.(*TestNodes)
	assert.True(t, ok, caseDesc)

	assert.Equal(t, len(wantNodes), len(objTn.Nodes), caseDesc)
	for i := range wantNodes {
		assert.Equal(t, *wantNodes[i], *objTn.Nodes[i], caseDesc)
	}
}

func TestDivideEntities(t *testing.T) {
	tests := []struct {
		caseDesc     string
		giveEntities []string
		wantDivide   map[string][]string
	}{
		{
			caseDesc:     "empty entities",
			giveEntities: nil,
			wantDivide:   map[string][]string{},
		},
		{
			caseDesc:     "normal case",
			giveEntities: []string{"upstream;1", "route;1"},
			wantDivide: map[string][]string{
				"upstream": {"1"},
				"route":    {"1"},
			},
		},
	}

	for _, tc := range tests {
		divide := divideEntities(tc.giveEntities)
		flag := reflect.DeepEqual(divide, tc.wantDivide)
		assert.True(t, flag, tc.caseDesc)
	}
}
