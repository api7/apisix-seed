package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/components"
	"github.com/api7/apisix-seed/internal/core/storer"
	"github.com/api7/apisix-seed/internal/discoverer"
	"github.com/api7/apisix-seed/internal/log"
)

func main() {
	conf.InitConf()
	log.InitLogger()

	etcdClient, err := storer.NewEtcd(conf.ETCDConfig)
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := storer.InitStores(etcdClient)
		if err != nil {
			panic(err)
		}
	}()
	go func() {
		defer wg.Done()
		err := discoverer.InitDiscoverers()
		if err != nil {
			panic(err)
		}
	}()
	wg.Wait()

	rewriter := components.Rewriter{}
	rewriter.Init()
	// TODO: Init Watcher

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Infof("APISIX-Seed receive %s and start shutting down\n", sig.String())
}
