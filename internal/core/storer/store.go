package storer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/api7/apisix-seed/internal/core/message"
	"github.com/api7/apisix-seed/internal/log"
)

type Interface interface {
	List(context.Context, string) ([]*message.Message, error)
	Update(context.Context, string, string) error
	Watch(context.Context, string) <-chan []*message.Message
}

type GenericStoreOption struct {
	BasePath string
	Prefix   string
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
		log.Error("base path empty")
		return nil, fmt.Errorf("base path can not be empty")
	}

	s := &GenericStore{
		Typ: typ,
		Stg: stg,
		opt: opt,
	}

	return s, nil
}

func (s *GenericStore) List(filter func(*message.Message) bool) ([]*message.Message, error) {
	lc, lcancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer lcancel()
	ret, err := s.Stg.List(lc, s.opt.BasePath)
	if err != nil {
		return nil, err
	}

	objPtrs := make([]*message.Message, 0)
	for i := range ret {
		if filter == nil || filter(ret[i]) {
			s.Store(ret[i].Key, ret[i])
			objPtrs = append(objPtrs, ret[i])
		}
	}

	return objPtrs, nil
}

func (s *GenericStore) Watch() <-chan []*message.Message {
	c, cancel := context.WithCancel(context.TODO())
	s.cancel = cancel

	ch := s.Stg.Watch(c, s.opt.BasePath)

	return ch
}

func (s *GenericStore) Unwatch() {
	s.cancel()
}

func (s *GenericStore) UpdateNodes(ctx context.Context, msg *message.Message) (err error) {
	bs, err := msg.Marshal()
	if err != nil {
		log.Errorf("json marshal failed: %s", err)
		return fmt.Errorf("marshal failed: %s", err)
	}
	if err = s.Stg.Update(ctx, msg.Key, string(bs)); err != nil {
		return err
	}

	return nil
}

func (s *GenericStore) Store(key string, objPtr interface{}) (interface{}, bool) {
	oldObj, ok := s.cache.LoadOrStore(key, objPtr)
	if ok {
		s.cache.Store(key, objPtr)
	}
	return oldObj, ok
}

func (s *GenericStore) Delete(key string) (interface{}, bool) {
	return s.cache.LoadAndDelete(key)
}

func (s *GenericStore) BasePath() string {
	return s.opt.BasePath
}
