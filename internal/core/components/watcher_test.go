package components

import (
	"testing"

	"github.com/api7/apisix-seed/internal/core/message"

	"github.com/api7/apisix-seed/internal/core/storer"
	"github.com/api7/apisix-seed/internal/discoverer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWatcherInit(t *testing.T) {
	caseDesc := "test WatcherInit"

	givenOpt := storer.GenericStoreOption{
		BasePath: "/prefix/mocks",
		Prefix:   "/prefix",
	}

	givenKey := "/prefix/mocks/1"
	givenA6Str := `{
    "uri": "/nacosWithNamespaceId/*",
    "upstream": {
        "service_name": "APISIX-NACOS",
        "type": "roundrobin",
        "discovery_type": "mock_nacos",
        "discovery_args": {
          "group_name": "DEFAULT_GROUP"
        }
    }
}`

	// inject mock function
	mStg := &storer.MockInterface{}
	mStg.On("List", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		assert.Equal(t, givenOpt.BasePath, args[0])
	}).Return(func() []*message.Message {
		msg, err := message.NewMessage(givenKey, []byte(givenA6Str), 1, message.EventAdd)
		assert.Nil(t, err, caseDesc)
		return []*message.Message{msg}
	}(), nil)

	storer.ClrearStores()
	// init store
	err := storer.InitStore("mocks", givenOpt, mStg)
	assert.Nil(t, err, caseDesc)

	discoverer.Discoveries = map[string]discoverer.NewDiscoverFunc{
		"mock_nacos": discoverer.NewDiscovererMock,
	}
	_ = discoverer.InitDiscoverer("mock_nacos", nil)

	discover := discoverer.GetDiscoverer("mock_nacos")
	mDiscover := discover.(interface{}).(*discoverer.MockInterface)
	mDiscover.On("Query", mock.Anything).Run(func(args mock.Arguments) {
		msg := args[0].(*message.Message)
		assert.Equal(t, givenKey, msg.Key)
		assert.Equal(t, "APISIX-NACOS", msg.ServiceName())
		assert.Equal(t, "mock_nacos", msg.DiscoveryType())
		assert.Equal(t, "DEFAULT_GROUP", msg.DiscoveryArgs()["group_name"])

	}).Return(nil)

	watcher := Watcher{}
	watcher.Init()
}

func TestWatcherWatch(t *testing.T) {
	watchCh := make(chan []*message.Message)
	mStg := &storer.MockInterface{}
	mStg.On("Watch", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {}).Return(watchCh)

	storer.ClrearStores()

	caseDesc := "Test watcher watch"
	givenOpt := storer.GenericStoreOption{
		BasePath: "/prefix/mocks",
		Prefix:   "/prefix",
	}
	err := storer.InitStore("mocks", givenOpt, mStg)
	assert.Nil(t, err, caseDesc)

	discoverer.Discoveries = map[string]discoverer.NewDiscoverFunc{
		"mock_nacos": discoverer.NewDiscovererMock,
		"mock_zk":    discoverer.NewDiscovererMock,
	}
	_ = discoverer.InitDiscoverer("mock_nacos", nil)
	nDiscover := discoverer.GetDiscoverer("mock_nacos").(*discoverer.MockInterface)
	_ = discoverer.InitDiscoverer("mock_zk", nil)
	zDiscover := discoverer.GetDiscoverer("mock_zk").(*discoverer.MockInterface)

	givenKey := "/prefix/mocks/1"

	nDiscover.On("Query", mock.Anything).Run(func(args mock.Arguments) {
		msg := args[0].(*message.Message)
		assert.Equal(t, "APISIX-NACOS", msg.ServiceName(), caseDesc)
		assert.Equal(t, "DEFAULT_GROUP", msg.DiscoveryArgs()["group_name"], caseDesc)
	}).Return(nil)

	nDiscover.On("Update", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		oldMsg := args[0].(*message.Message)
		newMsg := args[1].(*message.Message)
		assert.Equal(t, "APISIX-NACOS", oldMsg.ServiceName(), caseDesc)
		assert.Equal(t, "DEFAULT_GROUP", oldMsg.DiscoveryArgs()["group_name"], caseDesc)
		assert.Equal(t, "APISIX-NACOS", newMsg.ServiceName(), caseDesc)
		assert.Equal(t, "NEWDEFAULT_GROUP", newMsg.DiscoveryArgs()["group_name"], caseDesc)
	}).Return(nil)
	nDiscover.On("Delete", mock.Anything).Run(func(args mock.Arguments) {
		msg := args[0].(*message.Message)
		assert.Equal(t, "APISIX-NACOS", msg.ServiceName(), caseDesc)
		assert.Equal(t, "NEWDEFAULT_GROUP", msg.DiscoveryArgs()["group_name"], caseDesc)
	}).Return(nil)

	zDiscover.On("Query", mock.Anything).Run(func(args mock.Arguments) {
		msg := args[0].(*message.Message)
		assert.Equal(t, "APISIX-ZK", msg.ServiceName(), caseDesc)
	}).Return(nil)
	zDiscover.On("Delete", mock.Anything).Run(func(args mock.Arguments) {
		msg := args[0].(*message.Message)
		assert.Equal(t, "APISIX-ZK", msg.ServiceName(), caseDesc)
	}).Return(nil)

	watcher := Watcher{}
	watcher.sem = make(chan struct{}, 10)
	watcher.Watch()

	givenA6Str := `{
    "uri": "/hh/*",
    "upstream": {
        "service_name": "APISIX-NACOS",
        "type": "roundrobin",
        "discovery_type": "mock_nacos",
        "discovery_args": {
            "group_name": "DEFAULT_GROUP"
        }
    }
}`
	queryMsg, err := message.NewMessage(givenKey, []byte(givenA6Str), 1, message.EventAdd)
	assert.Nil(t, err, caseDesc)
	watchCh <- []*message.Message{queryMsg}

	givenUpdatedA6Str := `{
    "uri": "/hh/*",
    "upstream": {
        "service_name": "APISIX-NACOS",
        "type": "roundrobin",
        "discovery_type": "mock_nacos",
        "discovery_args": {
            "group_name": "NEWDEFAULT_GROUP"
        }
    }
}`
	updateMsg, err := message.NewMessage(givenKey, []byte(givenUpdatedA6Str), 1, message.EventAdd)
	assert.Nil(t, err, caseDesc)
	watchCh <- []*message.Message{updateMsg}

	givenReplacedA6Str := `{
    "uri": "/hh/*",
    "upstream": {
        "service_name": "APISIX-ZK",
        "type": "roundrobin",
        "discovery_type": "mock_zk"
    }
}`
	replaceMsg, err := message.NewMessage(givenKey, []byte(givenReplacedA6Str), 1, message.EventAdd)
	assert.Nil(t, err, caseDesc)
	watchCh <- []*message.Message{replaceMsg}

	deleteMsg, err := message.NewMessage(givenKey, nil, 1, message.EventDelete)
	assert.Nil(t, err, caseDesc)
	watchCh <- []*message.Message{deleteMsg}
}
