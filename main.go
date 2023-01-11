package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/api7/gopkg/pkg/log"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap/zapcore"

	"github.com/api7/apisix-seed/internal/conf"
	"github.com/api7/apisix-seed/internal/core/components"
	"github.com/api7/apisix-seed/internal/core/storer"
	"github.com/api7/apisix-seed/internal/discoverer"
)

func initLogger() error {

	opts := []log.Option{
		log.WithLogLevel(conf.LogConfig.Level),
		log.WithSkipFrames(3),
	}
	if conf.LogConfig.Path != "" {
		writer, err := rotatelogs.New(
			conf.LogConfig.Path+"-%Y%m%d%H%M%S",
			rotatelogs.WithLinkName(conf.LogConfig.Path),
			rotatelogs.WithMaxAge(conf.LogConfig.MaxAge),
			rotatelogs.WithRotationSize(conf.LogConfig.MaxSize),
			rotatelogs.WithRotationTime(conf.LogConfig.RotationTime),
		)
		if err != nil {
			return err
		}
		opts = append(opts, log.WithWriteSyncer(zapcore.AddSync(writer)))
	} else {
		opts = append(opts, log.WithOutputFile("stderr"))
	}
	l, err := log.NewLogger(opts...)
	if err != nil {
		return err
	}
	log.DefaultLogger = l

	return nil
}

func main() {
	conf.InitConf()

	if err := initLogger(); err != nil {
		panic(err)
	}

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

	rewriter := components.Rewriter{
		Prefix: conf.ETCDConfig.Prefix,
	}
	rewriter.Init()
	defer rewriter.Close()

	watcher := components.Watcher{}
	err = watcher.Init()
	if err != nil {
		log.Error(err.Error())
		return
	}
	watcher.Watch()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Infof("APISIX-Seed receive %s and start shutting down", sig.String())
}
