package components

import (
	"testing"
	"time"

	"github.com/api7/apisix-seed/internal/core/message"

	"github.com/api7/apisix-seed/internal/core/storer"
	"github.com/api7/apisix-seed/internal/discoverer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRewriter(t *testing.T) {
	caseDesc := "sanity"
	// init discover
	// Init Mock Discoverer
	discoverer.Discoveries = map[string]discoverer.NewDiscoverFunc{
		"mocks_dis": discoverer.NewDiscovererMock,
	}
	_ = discoverer.InitDiscoverer("mocks_dis", nil)

	watchCh := make(chan *message.Message, 1)
	discover := discoverer.GetDiscoverer("mocks_dis")
	mDiscover := discover.(*discoverer.MockInterface)
	mDiscover.On("Watch").Run(func(args mock.Arguments) {}).Return(watchCh)

	givenA6Str := `{
    "uri": "/nacosWithNamespaceId/*",
    "upstream": {
        "service_name": "APISIX-NACOS",
        "type": "roundrobin",
        "discovery_type": "nacos",
        "discovery_args": {
          "group_name": "DEFAULT_GROUP"
        }
    }
}`
	givenNodes := &message.Node{
		Host:   "1.1.1.1",
		Port:   8080,
		Weight: 1,
	}

	expectKey := "/prefix/mocks/1"
	expectA6Str := `{
    "uri": "/nacosWithNamespaceId/*",
    "upstream": {
        "nodes": {
            "host":"1.1.1.1",
            "port": 8080,
            "weight": 1
        },
        "_service_name": "APISIX-NACOS",
        "type": "roundrobin",
        "_discovery_type": "nacos",
        "discovery_args": {
          "group_name": "DEFAULT_GROUP"
        }
    }
}`
	// mock new upstream nodes was found
	msg, err := message.NewMessage("/prefix/mocks/1", []byte(givenA6Str), 1, message.EventAdd)
	assert.Nil(t, err, caseDesc)

	msg.InjectNodes(givenNodes)
	watchCh <- msg

	// mock rewrite
	mStg := &storer.MockInterface{}
	mStg.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		assert.Equal(t, expectKey, args[0], caseDesc)
		assert.JSONEq(t, expectA6Str, args[1].(string), caseDesc)
	}).Return(nil)
	storer.ClrearStores()
	err = storer.InitStore("mocks", storer.GenericStoreOption{
		BasePath: "/prefix/mocks",
		Prefix:   "/prefix",
	}, mStg)
	assert.Nil(t, err, caseDesc)

	rewriter := Rewriter{
		Prefix: "/prefix",
	}
	rewriter.Init()

	time.Sleep(time.Second)
}
