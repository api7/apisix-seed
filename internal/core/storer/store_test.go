package storer

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/api7/apisix-seed/internal/core/entity"
	"github.com/api7/apisix-seed/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
				ObjType:  reflect.TypeOf(GenericStoreOption{}),
			},
			wantStore: &GenericStore{
				Stg: nil,
				opt: GenericStoreOption{
					BasePath: "test",
					ObjType:  reflect.TypeOf(GenericStoreOption{}),
				},
			},
		},
		{
			caseDesc: "No BasePath",
			giveOpt: GenericStoreOption{
				BasePath: "",
				ObjType:  reflect.TypeOf(GenericStoreOption{}),
			},
			wantErr: fmt.Errorf("base path can not be empty"),
		},
		{
			caseDesc: "No object type",
			giveOpt: GenericStoreOption{
				BasePath: "test",
				ObjType:  nil,
			},
			wantErr: fmt.Errorf("object type can not be nil"),
		},
		{
			caseDesc: "Invalid object type",
			giveOpt: GenericStoreOption{
				BasePath: "test",
				ObjType:  reflect.TypeOf(""),
			},
			wantErr: fmt.Errorf("object type is invalid"),
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
		assert.Equal(t, tc.wantStore.opt.ObjType, s.opt.ObjType, tc.caseDesc)
	}
}

type TestStruct struct {
	entity.BaseInfo
	Field1 string
	Field2 string
}

func TestList(t *testing.T) {
	tests := []struct {
		caseDesc    string
		giveOpt     GenericStoreOption
		giveListErr error
		giveListRet utils.Message
		wantErr     error
		wantCache   map[string]interface{}
	}{
		{
			caseDesc: "sanity",
			giveOpt: GenericStoreOption{
				BasePath: "test",
				ObjType:  reflect.TypeOf(TestStruct{}),
			},
			giveListRet: utils.Message{
				{
					Key:   "test/demo1-f1",
					Value: `{"Field1":"demo1-f1", "Field2":"demo1-f2"}`,
				},
				{
					Key:   "test/demo2-f1",
					Value: `{"Field1":"demo2-f1", "Field2":"demo2-f2"}`,
				},
			},
			wantCache: map[string]interface{}{
				"demo1-f1": &TestStruct{
					BaseInfo: entity.BaseInfo{ID: "demo1-f1"},
					Field1:   "demo1-f1",
					Field2:   "demo1-f2",
				},
				"demo2-f1": &TestStruct{
					BaseInfo: entity.BaseInfo{ID: "demo2-f1"},
					Field1:   "demo2-f1",
					Field2:   "demo2-f2",
				},
			},
		},
		{
			caseDesc: "list error",
			giveOpt: GenericStoreOption{
				BasePath: "test",
				ObjType:  reflect.TypeOf(TestStruct{}),
			},
			giveListErr: fmt.Errorf("list error"),
			wantErr:     fmt.Errorf("list error"),
		},
		{
			caseDesc: "json error",
			giveOpt: GenericStoreOption{
				BasePath: "test",
				ObjType:  reflect.TypeOf(TestStruct{}),
			},
			giveListRet: utils.Message{
				{
					Key:   "test/demo1-f1",
					Value: `{"Field1","demo1-f1", "Field2":"demo1-f2"}`,
				},
			},
			wantErr: fmt.Errorf("unmarshal failed\n\tRelated Key:\t\tdemo1-f1\n\tError Description:\t" +
				"invalid character ',' after object key"),
		},
	}

	for _, tc := range tests {
		mStg := &MockInterface{}
		mStg.On("List", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			assert.Equal(t, tc.giveOpt.BasePath, args[0], tc.caseDesc)
		}).Return(tc.giveListRet, tc.giveListErr)

		store, err := NewGenericStore("test", tc.giveOpt, mStg)
		assert.Nil(t, err, tc.caseDesc)

		_, err = store.List(nil)
		if err != nil {
			assert.NotNil(t, tc.wantErr, tc.caseDesc)
			assert.Equal(t, tc.wantErr.Error(), err.Error(), tc.caseDesc)
			continue
		}

		store.cache.Range(func(key, value interface{}) bool {
			assert.Equal(t, tc.wantCache[key.(string)], value, tc.caseDesc)
			return true
		})
	}
}

func TestWatch(t *testing.T) {
	tests := []struct {
		caseDesc    string
		giveOpt     GenericStoreOption
		giveWatchCh chan *Watch
		giveWatch   *Watch
	}{
		{
			caseDesc: "sanity",
			giveOpt: GenericStoreOption{
				BasePath: "test",
				ObjType:  reflect.TypeOf(TestStruct{}),
			},
			giveWatchCh: make(chan *Watch, 1),
			giveWatch: &Watch{
				Events: []Event{
					{
						utils.Message{
							{Key: "event", Value: utils.EventDelete},
							{Key: "key", Value: "test/demo1-f1"},
							{Key: "value", Value: ""},
						},
					},
					{
						utils.Message{
							{Key: "event", Value: utils.EventAdd},
							{Key: "key", Value: "test/demo3-f1"},
							{Key: "value", Value: `{"Field1":"demo3-f1", "Field2":"demo3-f2"}`},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		mStg := &MockInterface{}
		mStg.On("Watch", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			assert.Equal(t, tc.giveOpt.BasePath, args[0], tc.caseDesc)
		}).Return(tc.giveWatchCh)

		store, err := NewGenericStore("test", tc.giveOpt, mStg)
		assert.Nil(t, err, tc.caseDesc)
		tc.giveWatchCh <- tc.giveWatch
		watch := <-store.Watch()
		assert.Equal(t, tc.giveWatch, watch, tc.caseDesc)
	}
}

type TestNodes struct {
	entity.BaseInfo
	Nodes []*entity.Node
}

func (t *TestNodes) SetNodes(nodes []*entity.Node) {
	t.Nodes = nodes
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		caseDesc  string
		giveOpt   GenericStoreOption
		giveCache map[string]interface{}
		giveErr   error
		giveKey   string
		giveNodes []*entity.Node
		wantErr   error
	}{
		{
			caseDesc: "sanity",
			giveOpt: GenericStoreOption{
				BasePath: "test",
				ObjType:  reflect.TypeOf(TestNodes{}),
			},
			giveCache: map[string]interface{}{
				"test/test1": &TestNodes{},
			},
			giveKey: "test1",
			giveNodes: []*entity.Node{
				{Host: "test.com", Weight: 10},
			},
		},
		{
			caseDesc: "not found",
			giveOpt: GenericStoreOption{
				BasePath: "test",
				ObjType:  reflect.TypeOf(TestNodes{}),
			},
			giveCache: map[string]interface{}{
				"test/test1": &TestNodes{},
			},
			giveKey: "test2",
			wantErr: fmt.Errorf("key: test2 is not found"),
		},
		{
			caseDesc: "not NodesSetter",
			giveOpt: GenericStoreOption{
				BasePath: "test",
				ObjType:  reflect.TypeOf(TestStruct{}),
			},
			giveCache: map[string]interface{}{
				"test/test1": &TestStruct{},
			},
			giveKey: "test1",
			wantErr: fmt.Errorf("obj can't set nodes"),
		},
	}

	for _, tc := range tests {
		mStg := &MockInterface{}
		mStg.On("Update", mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			assert.Equal(t, tc.giveKey, args[0], tc.caseDesc)
			input := TestNodes{}
			err := json.Unmarshal([]byte(args[1].(string)), &input)
			assert.Nil(t, err)
			assert.Equal(t, len(tc.giveNodes), len(input.Nodes), tc.caseDesc)
			for i := range tc.giveNodes {
				assert.Equal(t, *tc.giveNodes[i], *input.Nodes[i], tc.caseDesc)
			}
			assert.NotEqual(t, 0, input.UpdateTime, tc.caseDesc)
		}).Return(tc.giveErr)

		store, err := NewGenericStore("test", tc.giveOpt, mStg)
		assert.Nil(t, err, tc.caseDesc)

		for k, v := range tc.giveCache {
			store.Store(k, v)
		}

		err = store.UpdateNodes(context.TODO(), tc.giveKey, tc.giveNodes)
		if err != nil {
			assert.Equal(t, tc.wantErr, err, tc.caseDesc)
			continue
		}
		ret, ok := store.cache.Load(tc.giveKey)
		assert.True(t, ok, tc.caseDesc)
		retTn, ok := ret.(*TestNodes)
		assert.True(t, ok, tc.caseDesc)

		assert.Equal(t, len(tc.giveNodes), len(retTn.Nodes), tc.caseDesc)
		for i := range tc.giveNodes {
			assert.Equal(t, *tc.giveNodes[i], *retTn.Nodes[i], tc.caseDesc)
		}
	}
}

func TestStringToObjPtr(t *testing.T) {
	s, err := NewGenericStore("upstream", GenericStoreOption{
		BasePath: "test",
		ObjType:  reflect.TypeOf(entity.Upstream{}),
	}, nil)
	assert.Nil(t, err)
	rawID, id := "test/1", "1"
	argStr := `{"discovery_args":{"namespace_id":"dev", "group_name":"test"}}`
	argInterface, err := s.StringToObjPtr(argStr, rawID)
	assert.Nil(t, err)
	arg := argInterface.(*entity.Upstream)
	assert.Equal(t, id, arg.ID)
}
