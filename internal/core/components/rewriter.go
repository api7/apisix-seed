package components

import (
	"context"
	"runtime"

	"github.com/api7/gopkg/pkg/log"

	"github.com/api7/apisix-seed/internal/core/message"

	"github.com/api7/apisix-seed/internal/core/storer"
	"github.com/api7/apisix-seed/internal/discoverer"
)

type Rewriter struct {
	ctx    context.Context
	cancel context.CancelFunc

	// Limit the number of simultaneously update
	sem chan struct{}

	Prefix string
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

func (r *Rewriter) watch(ch chan *message.Message) {
	for {
		select {
		case <-r.ctx.Done():
			return
		case msg := <-ch:
			// hand watcher notify message
			_, entity, _ := storer.FromatKey(msg.Key, r.Prefix)
			if entity == "" {
				log.Errorf("key format Invaild: %s", msg.Key)
				return
			}
			if err := storer.GetStore(entity).UpdateNodes(r.ctx, msg); err != nil {
				log.Errorf("update nodes failed: %s", err)
			}
		}
	}
}
