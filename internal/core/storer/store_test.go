package storer

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/api7/apisix-seed/internal/core/message"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFromatKey(t *testing.T) {
	tests := []struct {
		caseDesc   string
		giveKey    string
		givePrefix string
		wantPrefix string
		wantEnity  string
		wantID     string
	}{
		{
			caseDesc:   "Normal case 1",
			giveKey:    "/prefix/entity/1",
			givePrefix: "/prefix",
			wantPrefix: "/prefix",
			wantEnity:  "entity",
			wantID:     "1",
		},
		{
			caseDesc:   "Normal case 2",
			giveKey:    "/prefix/entity/1/22",
			givePrefix: "/prefix",
			wantPrefix: "/prefix",
			wantEnity:  "entity",
			wantID:     "1/22",
		},
		{
			caseDesc:   "prefix not match",
			giveKey:    "/prefix/entity/1/22",
			givePrefix: "/aaaa",
			wantPrefix: "",
			wantEnity:  "",
			wantID:     "",
		},
		{
			caseDesc:   "prefix equal key",
			giveKey:    "/prefix/entity/1/22",
			givePrefix: "/prefix/entity/1/22",
			wantPrefix: "",
			wantEnity:  "",
			wantID:     "",
		},
		{
			caseDesc:   "prefix length is small than key",
			giveKey:    "/prefix/entity/1/22",
			givePrefix: "/prefix/entity/1/22/dsadas",
			wantPrefix: "",
			wantEnity:  "",
			wantID:     "",
		},
		{
			caseDesc:   "key is invalid",
			giveKey:    "/prefix//",
			givePrefix: "/prefix",
			wantPrefix: "/prefix",
			wantEnity:  "",
			wantID:     "",
		},
	}

	for _, tc := range tests {
		prefix, entity, id := FromatKey(tc.giveKey, tc.givePrefix)
		assert.Equal(t, tc.wantPrefix, prefix, tc.caseDesc)
		assert.Equal(t, tc.wantEnity, entity)
		assert.Equal(t, tc.wantID, id)
	}
}

func TestNewGenericStore(t *testing.T) {
	tests := []struct {
		caseDesc  string
		giveOpt   GenericStoreOption
		wantStore *GenericStore
		wantErr   error
	}{
		{
			caseDesc: "Normal Case",
			giveOpt: GenericStoreOption{
				BasePath: "test",
			},
			wantStore: &GenericStore{
				Stg: nil,
				opt: GenericStoreOption{
					BasePath: "test",
				},
			},
		},
		{
			caseDesc: "No BasePath",
			giveOpt: GenericStoreOption{
				BasePath: "",
			},
			wantErr: fmt.Errorf("base path can not be empty"),
		},
	}

	for _, tc := range tests {
		s, err := NewGenericStore("test", tc.giveOpt, nil)
		assert.Equal(t, tc.wantErr, err, tc.caseDesc)
		if err != nil {
			continue
		}
		assert.Equal(t, tc.wantStore.Stg, s.Stg, tc.caseDesc)
		flag := reflect.DeepEqual(&tc.wantStore.cache, &s.cache)
		assert.True(t, flag, tc.caseDesc)
		assert.Equal(t, tc.wantStore.opt.BasePath, s.opt.BasePath, tc.caseDesc)
	}
}

func TestList(t *testing.T) {
	testMsgs := []struct {
		desc  string
		key   string
		a6Str string
	}{
		{
			desc:  "normal",
			key:   "/apisxi/routes/1",
			a6Str: `{"uri":"/test","upstream":{"service_name":"APISIX-ZK","type":"roundrobin","discovery_type":"mock_zk"}}`,
		},
		{
			desc:  "routes with nodes",
			key:   "/apisxi/routes/nodes",
			a6Str: `{"uri":"/test","upstream":{"nodes":{}}}`,
		},
	}
	msgs := make([]*message.Message, 0, len(testMsgs))
	for _, v := range testMsgs {
		msg, err := message.NewMessage(v.key, []byte(v.a6Str), 1, message.EventAdd, message.A6RoutesConf)
		assert.Nil(t, err, v.desc)
		msgs = append(msgs, msg)
	}
	mStg := &MockInterface{}
	mStg.On("List", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
	}).Return(msgs, nil)

	caseDesc := "list without filter"
	store, err := NewGenericStore("test", GenericStoreOption{
		BasePath: "/apisix/routes",
		Prefix:   "/apisix",
	}, mStg)
	assert.Nil(t, err, caseDesc)
	_, err = store.List(nil)
	assert.Nil(t, err, caseDesc)
	keys := make([]string, 0, len(testMsgs))
	store.cache.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	assert.ElementsMatch(t, []string{"/apisxi/routes/1", "/apisxi/routes/nodes"}, keys)

	caseDesc = "list with filter"
	store, err = NewGenericStore("test", GenericStoreOption{
		BasePath: "/apisix/routes",
		Prefix:   "/apisix",
	}, mStg)
	assert.Nil(t, err, caseDesc)
	_, err = store.List(message.ServiceFilter)
	assert.Nil(t, err, caseDesc)
	store.cache.Range(func(key, value interface{}) bool {
		assert.Equal(t, "/apisxi/routes/1", key.(string), caseDesc)
		return true
	})
}

func TestWatch(t *testing.T) {
	caseDesc := "sanity"
	ch := make(chan []*message.Message, 1)
	mStg := &MockInterface{}
	mStg.On("Watch", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
	}).Return(ch)
	store, err := NewGenericStore("test", GenericStoreOption{
		BasePath: "/apisix/routes",
		Prefix:   "/apisix",
	}, mStg)
	assert.Nil(t, err, caseDesc)

	a6Str := `{"uri":"/test","upstream":{"service_name":"APISIX-ZK","type":"roundrobin","discovery_type":"mock_zk"}}`
	givenMsg, err := message.NewMessage("/apisxi/routes/a", []byte(a6Str), 1, message.EventAdd, message.A6RoutesConf)
	assert.Nil(t, err, caseDesc)
	ch <- []*message.Message{givenMsg}

	msgs := <-store.Watch()
	assert.Equal(t, "/apisxi/routes/a", msgs[0].Key)
}

func TestUpdateNodes(t *testing.T) {
	caseDesc := "sanity"
	mStg := &MockInterface{}
	mStg.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
	}).Return(nil)
	store, err := NewGenericStore("test", GenericStoreOption{
		BasePath: "/apisix/routes",
		Prefix:   "/apisix",
	}, mStg)
	assert.Nil(t, err, caseDesc)

	a6Str := `{"uri":"/test","upstream":{"service_name":"APISIX-ZK","type":"roundrobin","discovery_type":"mock_zk"}}`
	givenMsg, err := message.NewMessage("/apisxi/routes/a", []byte(a6Str), 1, message.EventAdd, message.A6RoutesConf)
	assert.Nil(t, err, caseDesc)

	err = store.UpdateNodes(context.Background(), givenMsg)
	assert.Nil(t, err, caseDesc)
}
