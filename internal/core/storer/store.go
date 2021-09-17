package storer

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/api7/apisix-seed/internal/core/entity"
	"github.com/api7/apisix-seed/internal/utils"
)

type Interface interface {
	List(context.Context, string) (utils.Message, error)
	Update(context.Context, string, string) error
	Watch(context.Context, string) <-chan *Watch
}

type GenericStoreOption struct {
	BasePath string
	ObjType  reflect.Type
}

type GenericStore struct {
	Typ string
	Stg Interface

	cache sync.Map
	opt   GenericStoreOption

	cancel context.CancelFunc
}

func NewGenericStore(typ string, opt GenericStoreOption, stg Interface) (*GenericStore, error) {
	if opt.BasePath == "" {
		return nil, fmt.Errorf("base path can not be empty")
	}
	if opt.ObjType == nil {
		return nil, fmt.Errorf("object type can not be nil")
	}

	if opt.ObjType.Kind() == reflect.Ptr {
		opt.ObjType = opt.ObjType.Elem()
	}
	if opt.ObjType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("object type is invalid")
	}
	s := &GenericStore{
		Typ: typ,
		Stg: stg,
		opt: opt,
	}

	return s, nil
}

func (s *GenericStore) List() ([]interface{}, error) {
	lc, lcancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer lcancel()
	ret, err := s.Stg.List(lc, s.opt.BasePath)
	if err != nil {
		return nil, err
	}

	objPtrs := make([]interface{}, len(ret))
	for i := range ret {
		objPtr, err := s.StringToObjPtr(ret[i].Value, ret[i].Key)
		if err != nil {
			return nil, err
		}

		objPtrs[i] = objPtr
	}

	return objPtrs, nil
}

func (s *GenericStore) Watch() <-chan *Watch {
	c, cancel := context.WithCancel(context.TODO())
	s.cancel = cancel

	ch := s.Stg.Watch(c, s.opt.BasePath)

	return ch
}

func (s *GenericStore) Unwatch() {
	s.cancel()
}

func (s *GenericStore) UpdateNodes(ctx context.Context, key string, nodes []*entity.Node) (err error) {
	if key == "" {
		return fmt.Errorf("key is required")
	}

	storedObj, ok := s.cache.Load(key)
	if !ok {
		return fmt.Errorf("key: %s is not found", key)
	}

	// Update Nodes Information
	if setter, ok := storedObj.(entity.NodesSetter); ok {
		setter.SetNodes(nodes)
	} else {
		return fmt.Errorf("obj can't set nodes")
	}

	if setter, ok := storedObj.(entity.BaseInfoSetter); ok {
		info := setter.GetBaseInfo()
		info.Updating(info)
	}

	var bs []byte
	if aller, ok := storedObj.(entity.Aller); ok {
		bs, err = entity.Marshal(aller)
	} else {
		bs, err = json.Marshal(storedObj)
	}

	if err != nil {
		return fmt.Errorf("marshal failed: %s", err)
	}
	if err = s.Stg.Update(ctx, key, string(bs)); err != nil {
		return err
	}

	return nil
}

func (s *GenericStore) Store(key string, objPtr interface{}) (interface{}, bool) {
	return s.cache.LoadOrStore(s.key(key), objPtr)
}

func (s *GenericStore) Delete(key string) (interface{}, bool) {
	return s.cache.LoadAndDelete(key)
}

func (s *GenericStore) StringToObjPtr(str, key string) (interface{}, error) {
	key = s.key(key)

	objPtr := reflect.New(s.opt.ObjType)
	ret := objPtr.Interface()

	var err error
	if aller, ok := ret.(entity.Aller); ok {
		err = entity.Unmarshal([]byte(str), aller)
	} else {
		err = json.Unmarshal([]byte(str), ret)
	}
	if err != nil {
		return nil, fmt.Errorf("unmarshal failed\n\tRelated Key:\t\t%s\n\tError Description:\t%s", key, err)
	}

	if setter, ok := ret.(entity.BaseInfoSetter); ok {
		info := setter.GetBaseInfo()
		info.KeyCompat(key)
	}

	return ret, nil
}

func (s *GenericStore) key(key string) string {
	return key[len(s.opt.BasePath)+1:]
}
