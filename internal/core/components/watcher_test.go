package components

import (
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

func init() {
	// Init Mock Discoverer
	discoverer.Discoveries = map[string]discoverer.Discover{
		"mock": discoverer.NewDiscovererMock,
	}
}

func TestWatcherInit(t *testing.T) {
	giveOpt := storer.GenericStoreOption{
		BasePath: "test",
		ObjType:  reflect.TypeOf(entity.Route{}),
	}
	giveListRet := utils.Message{
		{
			Key:   "test/demo1",
			Value: `{"upstream":{"service_name": "test_service", "discovery_type": "mock"}}`,
		},
	}
	wantValues := []string{utils.EventAdd, "test/demo1", "test_service"}

	_ = discoverer.InitDiscoverer("mock", nil)
	discover := discoverer.GetDiscoverer("mock")
	mDiscover := discover.(interface{}).(*discoverer.MockInterface)
	mDiscover.On("Query", mock.Anything).Run(func(args mock.Arguments) {
		values, arg, err := args[0].(*comm.Query).Decode()
		assert.Nil(t, err)
		assert.Equal(t, wantValues, values)
		assert.Nil(t, arg)

	}).Return(nil)

	mStg := &storer.MockInterface{}
	mStg.On("List", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		assert.Equal(t, giveOpt.BasePath, args[0])
	}).Return(giveListRet, nil)

	err := storer.InitStore("mock", giveOpt, mStg)
	assert.Nil(t, err)

	watcher := Watcher{}
	watcher.Init()
}

func TestWatcherWatch(t *testing.T) {
	giveOpt := storer.GenericStoreOption{
		BasePath: "test",
		ObjType:  reflect.TypeOf(entity.Route{}),
	}

	watchCh := make(chan *storer.StoreEvent)
	mStg := &storer.MockInterface{}
	mStg.On("Watch", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {}).Return(watchCh)

	err := storer.InitStore("mock", giveOpt, mStg)
	assert.Nil(t, err)

	quries := make([]*comm.Query, 0)
	updates := make([]*comm.Update, 0)
	doneCh := make(chan struct{})

	_ = discoverer.InitDiscoverer("mock", nil)
	discover := discoverer.GetDiscoverer("mock")
	mDiscover := discover.(interface{}).(*discoverer.MockInterface)
	mDiscover.On("Query", mock.Anything).Run(func(args mock.Arguments) {
		quries = append(quries, args[0].(*comm.Query))
		doneCh <- struct{}{}
	}).Return(nil)
	mDiscover.On("Update", mock.Anything).Run(func(args mock.Arguments) {
		updates = append(updates, args[0].(*comm.Update))
		doneCh <- struct{}{}
	}).Return(nil)

	watcher := Watcher{}
	watcher.sem = make(chan struct{}, 10)
	watcher.Watch()

	caseDesc := "Test add new service information"
	giveEvent := utils.EventAdd
	giveKey := "test/demo1"
	giveValue := `{"upstream":{"service_name": "test_service", "discovery_type": "mock"}}`

	watchMsg := storer.NewStoreEvent(false)
	err = watchMsg.Add(giveEvent, giveKey, giveValue)
	assert.Nil(t, err, caseDesc)

	watchCh <- &watchMsg
	<-doneCh

	wantValues := []string{utils.EventAdd, giveKey, "test_service"}
	values, arg, err := quries[0].Decode()
	assert.Nil(t, err, caseDesc)
	assert.Equal(t, wantValues, values, caseDesc)
	assert.Nil(t, arg, caseDesc)

	caseDesc = "Test update service information"
	giveValue = `{"upstream": {
		"service_name": "test_service",
		"discovery_type": "mock",
		"discovery_args": {
			"group_name": "test_group"
		}
	}}`
	watchMsg = storer.NewStoreEvent(false)
	err = watchMsg.Add(giveEvent, giveKey, giveValue)
	assert.Nil(t, err, caseDesc)

	watchCh <- &watchMsg
	<-doneCh

	wantValues = []string{utils.EventUpdate, "test_service"}
	var wantOldArgs map[string]string = nil
	wantNewArgs := map[string]string{
		"group_name":   "test_group",
		"namespace_id": "",
	}
	values, oldArgs, newArgs, err := updates[0].Decode()
	assert.Nil(t, err, caseDesc)
	assert.Equal(t, wantValues, values, caseDesc)
	assert.Equal(t, wantOldArgs, oldArgs, caseDesc)
	assert.Equal(t, wantNewArgs, newArgs, caseDesc)

	caseDesc = "Test replace service information"
	giveValue = `{"upstream":{"service_name": "test_service2", "discovery_type": "mock"}}`

	watchMsg = storer.NewStoreEvent(false)
	err = watchMsg.Add(giveEvent, giveKey, giveValue)
	assert.Nil(t, err, caseDesc)

	watchCh <- &watchMsg
	<-doneCh
	<-doneCh

	wantValues = []string{utils.EventDelete, giveKey, "test_service"}
	values, arg, err = quries[1].Decode()
	assert.Nil(t, err, caseDesc)
	assert.Equal(t, wantValues, values, caseDesc)
	assert.Equal(t, wantNewArgs, arg, caseDesc)

	wantValues = []string{utils.EventAdd, giveKey, "test_service2"}
	values, arg, err = quries[2].Decode()
	assert.Nil(t, err, caseDesc)
	assert.Equal(t, wantValues, values, caseDesc)
	assert.Nil(t, arg, caseDesc)

	caseDesc = "Test delete service information"
	giveEvent = utils.EventDelete
	giveValue = `{"upstream":{"service_name": "test_service2", "discovery_type": "mock"}}`

	watchMsg = storer.NewStoreEvent(false)
	err = watchMsg.Add(giveEvent, giveKey, giveValue)
	assert.Nil(t, err, caseDesc)

	watchCh <- &watchMsg
	<-doneCh

	wantValues = []string{utils.EventDelete, giveKey, "test_service2"}
	values, arg, err = quries[3].Decode()
	assert.Nil(t, err, caseDesc)
	assert.Equal(t, wantValues, values, caseDesc)
	assert.Nil(t, arg, caseDesc)
}

func TestEncodeQuery(t *testing.T) {
	tests := []struct {
		caseDesc   string
		giveEvent  string
		giveTyp    string
		giveQueer  entity.Entity
		wantValues []string
		wantArgs   map[string]string
	}{
		{
			caseDesc:  "add event",
			giveEvent: utils.EventAdd,
			giveTyp:   "upstream",
			giveQueer: &entity.Upstream{
				BaseInfo: entity.BaseInfo{
					ID: "1",
				},
				UpstreamDef: entity.UpstreamDef{
					ServiceName: "test",
				},
			},
			wantValues: []string{utils.EventAdd, "upstream", "test"},
			wantArgs:   nil,
		},
		{
			caseDesc:  "delete event",
			giveEvent: utils.EventDelete,
			giveTyp:   "upstream",
			giveQueer: &entity.Upstream{
				BaseInfo: entity.BaseInfo{
					ID: "1",
				},
				UpstreamDef: entity.UpstreamDef{
					ServiceName: "test",
					DiscoveryArgs: &entity.UpstreamArg{
						GroupName: "test_group",
					},
				},
			},
			wantValues: []string{utils.EventDelete, "upstream", "test"},
			wantArgs: map[string]string{
				"group_name":   "test_group",
				"namespace_id": "",
			},
		},
	}

	for _, tc := range tests {
		query, err := encodeQuery(tc.giveEvent, tc.giveTyp, tc.giveQueer)
		assert.Nil(t, err, tc.caseDesc)

		values, args, err := query.Decode()
		assert.Nil(t, err, tc.caseDesc)
		assert.Equal(t, tc.wantValues, values, tc.caseDesc)
		assert.Equal(t, tc.wantArgs, args, tc.caseDesc)
	}
}

func TestEncodeUpdate(t *testing.T) {
	tests := []struct {
		caseDesc    string
		giveOld     entity.Entity
		giveNew     entity.Entity
		wantValues  []string
		wantOldArgs map[string]string
		wantNewArgs map[string]string
	}{
		{
			caseDesc: "sanity",
			giveOld: &entity.Upstream{
				BaseInfo: entity.BaseInfo{
					ID: "1",
				},
				UpstreamDef: entity.UpstreamDef{
					ServiceName: "test",
				},
			},
			giveNew: &entity.Upstream{
				BaseInfo: entity.BaseInfo{
					ID: "1",
				},
				UpstreamDef: entity.UpstreamDef{
					ServiceName: "test",
					DiscoveryArgs: &entity.UpstreamArg{
						GroupName: "test_group",
					},
				},
			},
			wantValues:  []string{utils.EventUpdate, "test"},
			wantOldArgs: nil,
			wantNewArgs: map[string]string{
				"group_name":   "test_group",
				"namespace_id": "",
			},
		},
	}

	for _, tc := range tests {
		update, err := encodeUpdate(tc.giveOld, tc.giveNew)
		assert.Nil(t, err, tc.caseDesc)

		values, oldArgs, newArgs, err := update.Decode()
		assert.Nil(t, err, tc.caseDesc)
		assert.Equal(t, tc.wantValues, values, tc.caseDesc)
		assert.Equal(t, tc.wantOldArgs, oldArgs, tc.caseDesc)
		assert.Equal(t, tc.wantNewArgs, newArgs, tc.caseDesc)
	}
}
