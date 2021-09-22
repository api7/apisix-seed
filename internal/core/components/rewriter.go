package components

import (
	"context"
	"runtime"
	"sync"

	"github.com/api7/apisix-seed/internal/core/entity"
	"github.com/api7/apisix-seed/internal/core/storer"
	"github.com/api7/apisix-seed/internal/discoverer"
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
	r.sem = make(chan struct{}, runtime.GOMAXPROCS(0)+10)

	// Watch for service updates from Discoverer
	for _, dis := range discoverer.GetDiscoverers() {
		ch := dis.Watch()
		go func() {
			for {
				select {
				case <-r.ctx.Done():
					return
				case watch := <-ch:
					values, entities, nodes, err := watch.Decode()
					if err != nil {
						continue
					}

					if values[0] == utils.EventUpdate {
						r.update(entities, entity.NodesFormat(nodes).([]*entity.Node))
					}
				}
			}
		}()
	}
}

func (r *Rewriter) Close() {
	r.cancel()

	for _, dis := range discoverer.GetDiscoverers() {
		dis.Stop()
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
				go func(key string) {
					defer func() {
						<-r.sem
						wg.Done()
					}()

					_ = storer.GetStore(hubKey).UpdateNodes(r.ctx, key, nodes)
				}(key)
			}
		}
	}
	wg.Wait()
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
