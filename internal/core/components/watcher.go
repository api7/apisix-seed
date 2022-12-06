package components

import (
	"context"
	"errors"
	"runtime"
	"sync"

	"github.com/api7/apisix-seed/internal/core/message"

	"github.com/api7/apisix-seed/internal/core/storer"
	"github.com/api7/apisix-seed/internal/discoverer"
	"github.com/api7/apisix-seed/internal/log"
)

type Watcher struct {
	ctx    context.Context
	cancel context.CancelFunc

	// Limit the number of simultaneously query
	sem chan struct{}
}

// Init: load apisix config from etcd, query service from discovery
func (w *Watcher) Init() error {
	// the number of semaphore is referenced to https://github.com/golang/go/blob/go1.17.1/src/cmd/compile/internal/noder/noder.go#L38
	w.sem = make(chan struct{}, runtime.GOMAXPROCS(0)+10)

	loadSuccess := true
	// List the initial information
	for _, s := range storer.GetStores() {
		//eg: query from etcd by prefix /apisix/routes/
		msgs, err := s.List(message.ServiceFilter)
		if err != nil {
			log.Errorf("storer list error: %v", err)
			loadSuccess = false
			break
		}

		if len(msgs) == 0 {
			continue
		}
		wg := sync.WaitGroup{}
		wg.Add(len(msgs))
		for _, msg := range msgs {
			w.sem <- struct{}{}
			go w.handleQuery(msg, &wg)
		}
		wg.Wait()
	}

	if !loadSuccess {
		return errors.New("failed to load all etcd resources")
	}
	return nil
}

// Watch: when updating route、service、upstream, query service from discovery
func (w *Watcher) Watch() {
	w.ctx, w.cancel = context.WithCancel(context.TODO())

	// Watch for entity updates from Storer
	for _, s := range storer.GetStores() {
		go w.handleWatch(s)
	}
}

func (w *Watcher) Close() {
	w.cancel()

	for _, s := range storer.GetStores() {
		s.Unwatch()
	}
}

// handleQuery: init and query the service from discovery by apisix's conf
func (w *Watcher) handleQuery(msg *message.Message, wg *sync.WaitGroup) {
	defer func() {
		<-w.sem
		wg.Done()
	}()

	_ = discoverer.GetDiscoverer(msg.DiscoveryType()).Query(msg)
}

func (w *Watcher) handleWatch(s *storer.GenericStore) {
	ch := s.Watch()

	for {
		select {
		case <-w.ctx.Done():
			return
		case msgs := <-ch:
			wg := sync.WaitGroup{}
			wg.Add(len(msgs))
			for _, msg := range msgs {
				w.sem <- struct{}{}
				go w.handleValue(msg, &wg, s)
			}
			wg.Wait()
		}
	}
}

func (w *Watcher) handleValue(msg *message.Message, wg *sync.WaitGroup, s *storer.GenericStore) {
	defer func() {
		<-w.sem
		wg.Done()
	}()

	log.Infof("Watcher handle %d event: key=%s", msg.Action, msg.Key)
	switch msg.Action {
	case message.EventAdd:
		w.update(msg, s)
	case message.EventDelete:
		w.delete(msg, s)
	}
}

func (w *Watcher) update(msg *message.Message, s *storer.GenericStore) {
	if !message.ServiceFilter(msg) {
		if !msg.HasNodesAttr() {
			return
		}
		w.delete(msg, s)
		return
	}

	obj, ok := s.Store(msg.Key, msg)
	if !ok {
		// Obtains a new entity with service information
		log.Infof("Watcher obtains a new entity %s with service information", msg.Key)
		_ = discoverer.GetDiscoverer(msg.DiscoveryType()).Query(msg)
		return
	}

	oldMsg := obj.(*message.Message)
	if message.ServiceUpdate(oldMsg, msg) {
		// Updates the service information of existing entity
		log.Infof("Watcher updates the service information of existing entity %s", msg.Key)
		_ = discoverer.GetDiscoverer(msg.DiscoveryType()).Update(oldMsg, msg)
		return
	}

	if message.ServiceReplace(oldMsg, msg) {
		// Replaces the service information of existing entity
		log.Infof("Watcher replaces the service information of existing entity %s", msg.Key)

		_ = discoverer.GetDiscoverer(oldMsg.DiscoveryType()).Delete(oldMsg)
		_ = discoverer.GetDiscoverer(msg.DiscoveryType()).Query(msg)

		return
	}

	log.Infof("Watcher update version only, key: %s, version: %d", msg.Key, msg.Version)
	_ = discoverer.GetDiscoverer(msg.DiscoveryType()).Update(oldMsg, msg)
}

func (w *Watcher) delete(msg *message.Message, s *storer.GenericStore) {
	obj, ok := s.Delete(msg.Key)
	if !ok {
		return
	}
	// Deletes an existing entity
	delMsg := obj.(*message.Message)
	log.Infof("Watcher deletes an existing entity %s", delMsg.Key)
	_ = discoverer.GetDiscoverer(delMsg.DiscoveryType()).Delete(delMsg)
}
