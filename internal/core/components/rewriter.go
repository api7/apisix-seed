package components

import (
	"context"
	"runtime"
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
		ch := dis.Watch()
		go r.watch(ch)
	}
}

func (r *Rewriter) Close() {
	r.cancel()

	for _, dis := range discoverer.GetDiscoverers() {
		dis.Stop()
	}
}

func (r *Rewriter) watch(ch chan *comm.Watch) {
	for {
		select {
		case <-r.ctx.Done():
			return
		case watch := <-ch:
			values, entities, nodes, err := watch.Decode()
			if err != nil {
				log.Warnf("Rewriter decode watch message error: %s", err)
				continue
			}

			if values[0] == utils.EventUpdate {
				log.Info("Rewriter update the service information of entities")
				r.update(entities, entity.NodesFormat(nodes).([]*entity.Node))
			}
		}
	}
}

func (r *Rewriter) update(entities []string, nodes []*entity.Node) {
	divides := divideEntities(entities)

	wg := sync.WaitGroup{}
	wg.Add(len(entities))
	for typ, keys := range divides {
		hubKey := storer.HubKey(typ)
		for _, key := range keys {
			select {
			case <-r.ctx.Done():
				return
			case r.sem <- struct{}{}:
				go r.write(key, hubKey, nodes, &wg)
			}
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

// Divide the different types of entities
func divideEntities(entities []string) map[string][]string {
	divides := make(map[string][]string)
	for _, entityID := range entities {
		entityTyp, id := discoverer.DecodeEntityID(entityID)
		divides[entityTyp] = append(divides[entityTyp], id)
	}
	return divides
}
