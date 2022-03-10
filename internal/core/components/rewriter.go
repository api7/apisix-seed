package components

import (
	"context"
	"runtime"
	"strings"
	"sync"

	"github.com/api7/apisix-seed/internal/core/comm"
	"github.com/api7/apisix-seed/internal/core/entity"
	"github.com/api7/apisix-seed/internal/core/storer"
	"github.com/api7/apisix-seed/internal/discoverer"
	"github.com/api7/apisix-seed/internal/log"
	"github.com/api7/apisix-seed/internal/utils"
)

type Rewriter struct {
	ctx    context.Context
	cancel context.CancelFunc

	// Limit the number of simultaneously update
	sem chan struct{}
}

func (r *Rewriter) Init() {
	r.ctx, r.cancel = context.WithCancel(context.TODO())
	// the number of semaphore is referenced to https://github.com/golang/go/blob/go1.17.1/src/cmd/compile/internal/noder/noder.go#L38
	r.sem = make(chan struct{}, runtime.GOMAXPROCS(0)+10)

	// Watch for service updates from Discoverer
	for _, dis := range discoverer.GetDiscoverers() {
		msgCh := dis.Watch()
		go r.watch(msgCh)
	}
}

func (r *Rewriter) Close() {
	log.Info("Rewriter close")
	r.cancel()

	for _, dis := range discoverer.GetDiscoverers() {
		dis.Stop()
	}
}

func (r *Rewriter) watch(ch chan *comm.Message) {
	for {
		select {
		case <-r.ctx.Done():
			return
		case msg := <-ch:
			// hand watcher notify message
			values, entities, nodes, err := msg.Decode()
			if err != nil {
				log.Warnf("Rewriter decode watch message error: %s", err)
				continue
			}

			if len(nodes) == 0 {
				log.Errorf("Rewriter found empty nodes")
			}
			if values[0] == utils.EventUpdate {
				log.Infof("Rewriter update the service information of entities: %s", msg.String())
				r.update(entities, entity.NodesFormat(nodes).([]*entity.Node))
			}
		}
	}
}

func (r *Rewriter) update(entities []string, nodes []*entity.Node) {
	wg := sync.WaitGroup{}
	wg.Add(len(entities))
	for _, entityID := range entities {
		hubKey := getHubKey(entityID)
		select {
		case <-r.ctx.Done():
			return
		case r.sem <- struct{}{}:
			go r.write(entityID, hubKey, nodes, &wg)
		}
	}
	wg.Wait()
}

func (r *Rewriter) write(key string, hubKey storer.HubKey, nodes []*entity.Node, wg *sync.WaitGroup) {
	defer func() {
		<-r.sem
		wg.Done()
	}()

	_ = storer.GetStore(hubKey).UpdateNodes(r.ctx, key, nodes)
}

func getHubKey(entityID string) storer.HubKey {
	s := strings.Split(entityID, "/")[2]
	return storer.HubKey(s[:len(s)-1])
}
